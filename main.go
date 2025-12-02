package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/engine"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	entstrategy "github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/exchange/variational"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/model"
	"github.com/fachebot/omni-grid-bot/internal/strategy"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot"
	"github.com/fachebot/omni-grid-bot/internal/telebot/handler"
	"github.com/fachebot/omni-grid-bot/internal/util"
)

var (
	version     = "dev"
	showVersion = flag.Bool("version", false, "显示版本信息")
	configFile  = flag.String("f", "etc/config.yaml", "the config file")
)

func startAllStrategy(svcCtx *svc.ServiceContext, strategyEngine *engine.StrategyEngine) {
	offset := 0
	const limit = 100

	for {
		data, err := svcCtx.StrategyModel.FindAllByActiveStatus(context.TODO(), offset, limit)
		if err != nil {
			logger.Fatalf("[startAllStrategy] 加载活跃的策略列表失败, %v", err)
		}

		if len(data) == 0 {
			break
		}

		for _, item := range data {
			s := strategy.NewGridStrategy(svcCtx, item)
			err = strategyEngine.StartStrategy(s)
			if err != nil {
				logger.Fatalf("[startAllStrategy] 启动策略失败, id: %s, symbol: %s, %v", item.GUID, item.Symbol, err)
			}
			logger.Infof("[startAllStrategy] 启动策略成功, id: %s, symbol: %s", item.GUID, item.Symbol)
		}

		offset = offset + len(data)
	}
}

func handleOrderCancelled(ctx context.Context, svcCtx *svc.ServiceContext, strategyEngine *engine.StrategyEngine, record *ent.Strategy) {
	// 停止网格策略
	strategyEngine.StopStrategy(record.GUID)

	// 取消用户订单
	err := handler.CancelAllOrders(ctx, svcCtx, record)
	if err != nil {
		logger.Warnf("[handleOrderCancelled] 取消用户所有订单失败, exchange: %s, account: %s, symbol: %s, side: %s, %v",
			record.Exchange, record.Account, record.Symbol, record.Mode, err)
	}

	// 更新策略状态
	err = util.Tx(ctx, svcCtx.DbClient, func(tx *ent.Tx) error {
		err = model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		err = model.NewMatchedTradeModel(tx.MatchedTrade).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatus(ctx, record.ID, entstrategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[handleOrderCancelled] 更新策略状态失败, guid: %s, %v", record.GUID, err)
	}

	// 发送通知消息
	chatId := util.ChatId(record.Owner)
	name := handler.StrategyName(record)
	link := fmt.Sprintf("[%s](https://t.me/%s?start=%s)",
		name, svcCtx.Bot.Me.Username, record.GUID)
	text := fmt.Sprintf("🚨 **%s %s** 策略已停止 %s\n\n",
		record.Symbol, strings.ToUpper(string(record.Mode)), link)
	text += "由于订单被意外取消，策略已自动停止，请手动关闭仓位。\n\n**注意**：`策略运行中请勿手动进行操作，以免干扰策略正常运行。`"
	_, err = util.SendMarkdownMessage(svcCtx.Bot, chatId, text, nil)
	if err != nil {
		logger.Debugf("[handleOrderCancelled] 发送通知失败, chat: %d, %v", chatId, err)
	}
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\n", version)
		return
	}

	// 读取配置文件
	c, err := config.LoadFromFile(*configFile)
	if err != nil {
		logger.Fatalf("读取配置文件失败, %s", err)
	}

	// 创建数据目录
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			logger.Fatalf("创建数据目录失败, %s", err)
		}
	}

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c)

	// 启动Lighter订阅器
	lighterSubscriber := lighter.NewLighterSubscriber(c.Sock5Proxy)
	lighterSubscriber.Start()
	lighterSubscriber.WaitUntilConnected()

	// 启动Paradex订阅器
	paradexSubscriber := paradex.NewParadexSubscriber(c.Sock5Proxy)
	paradexSubscriber.Start()

	// 启动Variational订阅器
	variationalSubscriber := variational.NewVariationalSubscriber(svcCtx.PendingOrdersCache, c.Sock5Proxy)
	variationalSubscriber.Start()

	// 启动网格策略引擎
	strategyEngine := engine.NewStrategyEngine(
		svcCtx, lighterSubscriber, paradexSubscriber, handleOrderCancelled)
	strategyEngine.Start()

	// 启动所有网络
	startAllStrategy(svcCtx, strategyEngine)

	// 运行机器人服务
	botService := telebot.NewTeleBot(svcCtx, strategyEngine)
	if err != nil {
		logger.Fatalf("创建机器人服务失败, %s", err)
	}
	botService.Start()

	// 等待程序退出
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	strategyEngine.Stop()
	lighterSubscriber.Stop()
	paradexSubscriber.Stop()
	variationalSubscriber.Stop()
	botService.Stop()

	svcCtx.Close()
	logger.Infof("服务已停止")
}
