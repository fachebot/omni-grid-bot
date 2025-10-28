package svc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/fachebot/perp-dex-grid-bot/internal/cache"
	"github.com/fachebot/perp-dex-grid-bot/internal/config"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/model"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/proxy"
	tele "gopkg.in/telebot.v4"
)

type ServiceContext struct {
	Config         *config.Config
	Bot            *tele.Bot
	DbClient       *ent.Client
	TransportProxy *http.Transport
	MessageCache   *cache.MessageCache
	LighterCache   *cache.LighterCache
	LighterClient  *lighter.Client

	GridModel         *model.GridModel
	OrderModel        *model.OrderModel
	StrategyModel     *model.StrategyModel
	SyncProgressModel *model.SyncProgressModel
}

func NewServiceContext(c *config.Config) *ServiceContext {
	// 创建数据库连接
	client, err := ent.Open("sqlite3", "file:data/sqlite.db?mode=rwc&_journal_mode=WAL&_fk=1")
	if err != nil {
		logger.Fatalf("打开数据库失败, %v", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		logger.Fatalf("创建数据库Schema失败, %v", err)
	}

	// 创建SOCKS5代理
	var transportProxy *http.Transport
	if c.Sock5Proxy.Enable {
		socks5Proxy := fmt.Sprintf("%s:%d", c.Sock5Proxy.Host, c.Sock5Proxy.Port)
		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			logger.Fatalf("创建SOCKS5代理失败, %v", err)
		}

		transportProxy = &http.Transport{
			Dial:            dialer.Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// 创建LighterClient
	lighterClient := lighter.NewClient()

	// 创建Telegram Bot
	httpClient := new(http.Client)
	if transportProxy != nil {
		httpClient.Transport = transportProxy
	}

	pref := tele.Settings{
		Token:  c.TelegramBot.ApiToken,
		Poller: &tele.LongPoller{Timeout: 5 * time.Second},
		Client: httpClient,
	}
	bot, err := tele.NewBot(pref)
	if err != nil {
		logger.Fatalf("创建Telegram Bot失败, %v", err)
	}
	logger.Infof("[TeleBot], BotID: %d, Username: %s", bot.Me.ID, bot.Me.Username)

	svcCtx := &ServiceContext{
		Config:         c,
		Bot:            bot,
		DbClient:       client,
		TransportProxy: transportProxy,
		MessageCache:   cache.NewMessageCache(),
		LighterCache:   cache.NewLighterCache(lighterClient),
		LighterClient:  lighterClient,

		GridModel:         model.NewGridModel(client.Grid),
		OrderModel:        model.NewOrderModel(client.Order),
		StrategyModel:     model.NewStrategyModel(client.Strategy),
		SyncProgressModel: model.NewSyncProgressModel(client.SyncProgress),
	}
	return svcCtx
}

func (svcCtx *ServiceContext) Close() {
	if err := svcCtx.DbClient.Close(); err != nil {
		logger.Errorf("关闭数据库失败, %v", err)
	}
}
