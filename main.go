package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/engine"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/strategy"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot"
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

	// 启动Lighter订阅器
	lighterSubscriber := lighter.NewLighterSubscriber(c.Sock5Proxy)
	lighterSubscriber.Start()
	lighterSubscriber.WaitUntilConnected()

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c, lighterSubscriber)

	// 启动网格策略引擎
	strategyEngine := engine.NewStrategyEngine(svcCtx, lighterSubscriber)
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
	botService.Stop()

	svcCtx.Close()
	logger.Infof("服务已停止")
}
