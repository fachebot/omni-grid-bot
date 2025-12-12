package lighter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/samber/lo"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const (
	reconnectInitial = 1 * time.Second
	reconnectMax     = 30 * time.Second
)

type MarketResolver interface {
	GetMarketIdBySymbol(ctx context.Context, symbol string) (int16, error)
	GetSymbolByMarketId(ctx context.Context, marketIndex int16) (string, error)
}

type LighterSubscriber struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	url       string
	conn      *websocket.Conn
	proxy     config.Sock5Proxy
	reconnect chan struct{}

	mutex          sync.Mutex
	accounts       map[int64]*Signer
	marketResolver MarketResolver

	subMsgChan chan exchange.SubMessage
}

func NewLighterSubscriber(marketResolver MarketResolver, proxy config.Sock5Proxy) *LighterSubscriber {
	ctx, cancel := context.WithCancel(context.Background())
	subscriber := &LighterSubscriber{
		ctx:            ctx,
		cancel:         cancel,
		url:            "wss://mainnet.zklighter.elliot.ai/stream",
		proxy:          proxy,
		reconnect:      make(chan struct{}, 1),
		accounts:       make(map[int64]*Signer),
		marketResolver: marketResolver,
	}
	return subscriber
}

func (subscriber *LighterSubscriber) Stop() {
	logger.Infof("[LighterSubscriber] 准备停止服务")

	subscriber.cancel()

	if subscriber.conn != nil {
		subscriber.conn.Close()
	}

	<-subscriber.stopChan

	close(subscriber.stopChan)
	subscriber.stopChan = nil

	if subscriber.subMsgChan != nil {
		close(subscriber.subMsgChan)
	}
	subscriber.subMsgChan = nil

	logger.Infof("[LighterSubscriber] 服务已经停止")
}

func (subscriber *LighterSubscriber) Start() {
	if subscriber.stopChan != nil {
		return
	}

	subscriber.stopChan = make(chan struct{}, 1)

	if subscriber.conn == nil {
		logger.Infof("[LighterSubscriber] 开始运行服务")
		go subscriber.run()
	}
}

func (subscriber *LighterSubscriber) WaitUntilConnected() {
	for subscriber.conn == nil {
		time.Sleep(time.Second * 1)
	}
}

func (subscriber *LighterSubscriber) SubscriptionChan() <-chan exchange.SubMessage {
	if subscriber.subMsgChan == nil {
		subscriber.subMsgChan = make(chan exchange.SubMessage, 1024)
	}
	return subscriber.subMsgChan
}

func (subscriber *LighterSubscriber) SubscribeMarketStats(symbol string) error {
	if subscriber.conn == nil {
		return errors.New("connection is not established")
	}

	marketIndex, err := subscriber.marketResolver.GetMarketIdBySymbol(subscriber.ctx, symbol)
	if err != nil {
		return err
	}

	message := fmt.Sprintf(`{ "type": "subscribe", "channel": "market_stats/%d" }`, marketIndex)
	return subscriber.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (subscriber *LighterSubscriber) UnsubscribeMarketStats(symbol string) error {
	if subscriber.conn == nil {
		return errors.New("connection is not established")
	}

	marketIndex, err := subscriber.marketResolver.GetMarketIdBySymbol(subscriber.ctx, symbol)
	if err != nil {
		return err
	}

	message := fmt.Sprintf(`{ "type": "unsubscribe", "channel": "market_stats/%d" }`, marketIndex)
	return subscriber.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (subscriber *LighterSubscriber) SubscribeAccountOrders(signer *Signer) error {
	if subscriber.conn == nil {
		return errors.New("connection is not established")
	}

	auth, err := signer.GetAuthToken(time.Now().Add(time.Second * 30))
	if err != nil {
		return err
	}

	logger.Debugf("[LighterSubscriber] 订阅账户订单活动, accountIndex: %d", signer.accountIndex)

	subscriber.mutex.Lock()
	subscriber.accounts[signer.accountIndex] = signer
	subscriber.mutex.Unlock()

	message := fmt.Sprintf(`{ "type": "subscribe", "channel": "account_all_orders/%d", "auth": "%s" }`, signer.accountIndex, auth)
	return subscriber.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (subscriber *LighterSubscriber) UnsubscribeAccountOrders(signer *Signer) error {
	if subscriber.conn == nil {
		return errors.New("connection is not established")
	}

	logger.Debugf("[LighterSubscriber] 取消订阅账户订单活动, accountIndex: %d", signer.accountIndex)

	subscriber.mutex.Lock()
	delete(subscriber.accounts, signer.accountIndex)
	subscriber.mutex.Unlock()

	message := fmt.Sprintf(`{ "type": "unsubscribe", "channel": "account_all_orders/%d" }`, signer.accountIndex)
	return subscriber.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (subscriber *LighterSubscriber) run() {
	subscriber.connect()

	reconnectDelay := reconnectInitial
loop:
	for {
		select {
		case <-subscriber.ctx.Done():
			break loop
		case <-subscriber.reconnect:
			select {
			case <-subscriber.ctx.Done():
				break loop
			case <-time.After(reconnectDelay):
				logger.Infof("[LighterSubscriber] 重新建立连接...")
				subscriber.connect()

				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	subscriber.stopChan <- struct{}{}
}

func (subscriber *LighterSubscriber) connect() {
	sock5Proxy := ""
	if subscriber.proxy.Enable {
		sock5Proxy = fmt.Sprintf("%s:%d", subscriber.proxy.Host, subscriber.proxy.Port)
	}

	netDial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if sock5Proxy == "" {
			netDialer := &net.Dialer{}
			return netDialer.DialContext(ctx, network, addr)
		}

		dialer, err := proxy.SOCKS5(network, sock5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return dialer.Dial(network, addr)
	}

	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return netDial(subscriber.ctx, network, addr)
		},
		HandshakeTimeout: 45 * time.Second,
	}
	conn, _, err := dialer.Dial(subscriber.url, nil)
	if err != nil {
		logger.Errorf("[LighterSubscriber] 连接失败, %v", err)
		subscriber.scheduleReconnect()
		return
	}

	subscriber.conn = conn
	logger.Infof("[LighterSubscriber] 连接已建立")

	go subscriber.readMessages()
}

func (subscriber *LighterSubscriber) readMessages() {
	defer subscriber.conn.Close()

	processedAccounts := make(map[int64]struct{})
	for {
		_, data, err := subscriber.conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.Errorf("[LighterSubscriber] 读取出错, %v", err)
			subscriber.scheduleReconnect()
			return
		}

		logger.Debugf("[LighterSubscriber] 收到新消息, %s", data)

		var message WebSocketMessage
		if err = json.Unmarshal(data, &message); err != nil {
			logger.Errorf("[LighterSubscriber] 解析消息失败, %v", err)
			continue
		}

		switch message.Type {
		case "ping":
			msg := `{ "type": "pong" }`
			if err := subscriber.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				logger.Errorf("[LighterSubscriber] 发送心跳消息失败, %v", err)
			}
		case "connected":
			accounts := make(map[int64]*Signer)
			subscriber.mutex.Lock()
			maps.Copy(accounts, subscriber.accounts)
			subscriber.mutex.Unlock()

			for _, signer := range accounts {
				subscriber.SubscribeAccountOrders(signer)
			}
		case "unsubscribed":
			const prefix = "account_all_orders:"
			if !strings.HasPrefix(message.Channel, prefix) {
				continue
			}

			accountIndex, err := strconv.ParseInt(message.Channel[len(prefix):], 10, 64)
			if err != nil {
				logger.Errorf("[LighterSubscriber] 解析channel失败, channel: %s, %v", message.Channel, err)
				continue
			}

			delete(processedAccounts, accountIndex)
		case "update/market_stats":
			subscriber.handleMarketStatsMessage(message)
		case "subscribed/account_all_orders", "update/account_all_orders":
			subscriber.handleOrdersMessage(processedAccounts, message)
		}
	}
}

func (subscriber *LighterSubscriber) scheduleReconnect() {
	if subscriber.ctx.Err() == nil {
		select {
		case subscriber.reconnect <- struct{}{}:
		default:
		}
	}
}

func (subscriber *LighterSubscriber) handleMarketStatsMessage(message WebSocketMessage) {
	const prefix = "market_stats:"
	if !strings.HasPrefix(message.Channel, prefix) {
		logger.Errorf("[LighterSubscriber] 解析channel失败, channel: %s", message.Channel)
		return
	}

	marketIndex, err := strconv.ParseInt(message.Channel[len(prefix):], 10, 64)
	if err != nil {
		logger.Errorf("[LighterSubscriber] 解析channel失败, channel: %s, %v", message.Channel, err)
		return
	}

	if subscriber.subMsgChan != nil && message.MarketStats != nil {
		symbol, err := subscriber.marketResolver.GetSymbolByMarketId(subscriber.ctx, int16(marketIndex))
		if err != nil {
			logger.Errorf("[LighterSubscriber] 解析MarketIndex失败, marketIndex: %d, %v", marketIndex, err)
			return
		}

		marketStats := exchange.MarketStats{
			Symbol:    symbol,
			Price:     message.MarketStats.LastTradePrice,
			MarkPrice: message.MarketStats.MarkPrice,
		}

		logger.Debugf("[LighterSubscriber] 分发 MarketStats 数据, %+v", marketStats)
		subscriber.subMsgChan <- exchange.SubMessage{Exchange: exchange.Lighter, MarketStats: &marketStats}
	}
}

func (subscriber *LighterSubscriber) handleOrdersMessage(processedAccounts map[int64]struct{}, message WebSocketMessage) {
	const prefix = "account_all_orders:"
	if !strings.HasPrefix(message.Channel, prefix) {
		logger.Errorf("[LighterSubscriber] 解析channel失败, channel: %s", message.Channel)
		return
	}

	accountIndex, err := strconv.ParseInt(message.Channel[len(prefix):], 10, 64)
	if err != nil {
		logger.Errorf("[LighterSubscriber] 解析channel失败, channel: %s, %v", message.Channel, err)
		return
	}

	if subscriber.subMsgChan != nil && message.Orders != nil {
		_, ok := processedAccounts[accountIndex]
		userOrders := exchange.UserOrders{
			Exchange:   exchange.Lighter,
			Account:    strconv.FormatInt(accountIndex, 10),
			Orders:     make([]*exchange.Order, 0),
			IsSnapshot: !ok,
		}

		for marketIndex, marketOrders := range message.Orders {
			marketIndexN, err := strconv.Atoi(marketIndex)
			if err != nil {
				logger.Errorf("[LighterSubscriber] 解析MarketIndex失败, marketIndex: %s, %v", marketIndex, err)
				continue
			}

			symbol, err := subscriber.marketResolver.GetSymbolByMarketId(subscriber.ctx, int16(marketIndexN))
			if err != nil {
				logger.Errorf("[LighterSubscriber] 解析MarketIndex失败, marketIndex: %s, %v", marketIndex, err)
				continue
			}

			for _, ord := range marketOrders {
				userOrders.Orders = append(userOrders.Orders, &exchange.Order{
					Symbol:            symbol,
					OrderID:           strconv.FormatInt(ord.OrderIndex, 10),
					ClientOrderID:     strconv.FormatInt(ord.ClientOrderIndex, 10),
					Side:              lo.If(ord.IsAsk, order.SideSell).Else(order.SideBuy),
					Price:             ord.Price,
					BaseAmount:        ord.InitialBaseAmount,
					FilledBaseAmount:  ord.FilledBaseAmount,
					FilledQuoteAmount: ord.FilledQuoteAmount,
					Timestamp:         ord.Timestamp * 1000, // 转化为毫秒数
					Status:            ConvertOrderStatus(ord.Status),
				})
			}
		}

		logger.Debugf("[LighterSubscriber] 分发 UserOders 数据, account: %d, isSnapshot: %v", accountIndex, userOrders.IsSnapshot)
		subscriber.subMsgChan <- exchange.SubMessage{Exchange: exchange.Lighter, UserOrders: &userOrders}
	}

	processedAccounts[accountIndex] = struct{}{}
}
