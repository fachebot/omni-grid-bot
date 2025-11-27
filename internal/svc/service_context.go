package svc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/cache"
	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/model"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/proxy"
	tele "gopkg.in/telebot.v4"
)

type ServiceContext struct {
	Config            *config.Config
	Bot               *tele.Bot
	DbClient          *ent.Client
	TransportProxy    *http.Transport
	MessageCache      *cache.MessageCache
	LighterCache      *cache.LighterCache
	ParadexCache      *cache.ParadexCache
	ParadexClient     *paradex.Client
	LighterClient     *lighter.Client
	LighterSubscriber *lighter.LighterSubscriber

	GridModel         *model.GridModel
	OrderModel        *model.OrderModel
	StrategyModel     *model.StrategyModel
	SyncProgressModel *model.SyncProgressModel
	MatchedTradeModel *model.MatchedTradeModel

	userLocks  map[int64]*sync.Mutex
	locksMutex sync.RWMutex
}

func NewServiceContext(c *config.Config, lighterSubscriber *lighter.LighterSubscriber) *ServiceContext {
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

	// 创建ParadexClient
	paraClient := new(http.Client)
	if transportProxy != nil {
		paraClient.Transport = transportProxy
	}
	paradexClient := paradex.NewClient(paraClient)

	// 创建LighterClient
	lighClient := new(http.Client)
	if transportProxy != nil {
		lighClient.Transport = transportProxy
	}
	lighterClient := lighter.NewClient(lighClient)

	// 创建Telegram Bot
	botHttpClient := new(http.Client)
	if transportProxy != nil {
		botHttpClient.Transport = transportProxy
	}

	pref := tele.Settings{
		Token:  c.TelegramBot.ApiToken,
		Poller: &tele.LongPoller{Timeout: 5 * time.Second},
		Client: botHttpClient,
	}
	bot, err := tele.NewBot(pref)
	if err != nil {
		logger.Fatalf("创建Telegram Bot失败, %v", err)
	}
	logger.Infof("[TeleBot] BotID: %d, Username: %s", bot.Me.ID, bot.Me.Username)

	svcCtx := &ServiceContext{
		Config:            c,
		Bot:               bot,
		DbClient:          client,
		TransportProxy:    transportProxy,
		MessageCache:      cache.NewMessageCache(),
		ParadexClient:     paradexClient,
		LighterCache:      cache.NewLighterCache(lighterClient),
		ParadexCache:      cache.NewParadexCache(paradexClient),
		LighterClient:     lighterClient,
		LighterSubscriber: lighterSubscriber,

		GridModel:         model.NewGridModel(client.Grid),
		OrderModel:        model.NewOrderModel(client.Order),
		StrategyModel:     model.NewStrategyModel(client.Strategy),
		SyncProgressModel: model.NewSyncProgressModel(client.SyncProgress),
		MatchedTradeModel: model.NewMatchedTradeModel(client.MatchedTrade),

		userLocks: make(map[int64]*sync.Mutex),
	}
	return svcCtx
}

func (svcCtx *ServiceContext) GetUserLock(userId int64) *sync.Mutex {
	svcCtx.locksMutex.RLock()
	if lock, exists := svcCtx.userLocks[userId]; exists {
		svcCtx.locksMutex.RUnlock()
		return lock
	}
	svcCtx.locksMutex.RUnlock()

	svcCtx.locksMutex.Lock()
	defer svcCtx.locksMutex.Unlock()

	if lock, exists := svcCtx.userLocks[userId]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	svcCtx.userLocks[userId] = lock
	return lock
}

func (svcCtx *ServiceContext) Close() {
	if err := svcCtx.DbClient.Close(); err != nil {
		logger.Errorf("关闭数据库失败, %v", err)
	}
}
