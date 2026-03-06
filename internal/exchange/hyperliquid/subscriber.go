package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/shopspring/decimal"

	"github.com/gorilla/websocket"
)

const (
	reconnectInitial = 1 * time.Second
	reconnectMax     = 30 * time.Second
)

type Subscriber struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	url       string
	conn      *websocket.Conn
	proxy     config.Sock5Proxy
	reconnect chan struct{}

	mutex     sync.Mutex
	accounts  map[string]*Signer
	cache     *Cache
	processed map[string]bool

	subMsgChan chan exchange.SubMessage
}

func NewSubscriber(signer *Signer, proxy config.Sock5Proxy, isMainnet bool, cache *Cache) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())

	url := MainnetWebSocketURL
	if !isMainnet {
		url = TestnetWebSocketURL
	}

	return &Subscriber{
		ctx:       ctx,
		cancel:    cancel,
		url:       url,
		proxy:     proxy,
		reconnect: make(chan struct{}, 1),
		accounts:  make(map[string]*Signer),
		cache:     cache,
		processed: make(map[string]bool),
	}
}

func (s *Subscriber) Stop() {
	logger.Infof("[HyperliquidSubscriber] 准备停止服务")

	s.cancel()

	if s.conn != nil {
		s.conn.Close()
	}

	<-s.stopChan

	close(s.stopChan)
	s.stopChan = nil

	if s.subMsgChan != nil {
		close(s.subMsgChan)
	}
	s.subMsgChan = nil

	logger.Infof("[HyperliquidSubscriber] 服务已经停止")
}

func (s *Subscriber) Start() {
	if s.stopChan != nil {
		return
	}

	s.stopChan = make(chan struct{}, 1)
	logger.Infof("[HyperliquidSubscriber] 开始运行服务")
	go s.run()
}

func (s *Subscriber) SubscriptionChan() <-chan exchange.SubMessage {
	if s.subMsgChan == nil {
		s.subMsgChan = make(chan exchange.SubMessage, 1024)
	}
	return s.subMsgChan
}

func (s *Subscriber) SubscribeUser(address string, signer *Signer) error {
	s.mutex.Lock()
	s.accounts[address] = signer
	s.mutex.Unlock()

	if s.conn == nil {
		return nil
	}

	subscribeMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "orderUpdates",
			"user": address,
		},
	}

	return s.conn.WriteJSON(subscribeMsg)
}

func (s *Subscriber) SubscribeMarket(coin string) error {
	if s.conn == nil {
		return fmt.Errorf("connection is not established")
	}

	subscribeMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "allMids",
		},
	}

	return s.conn.WriteJSON(subscribeMsg)
}

func (s *Subscriber) UnsubscribeUser(address string) error {
	s.mutex.Lock()
	delete(s.accounts, address)
	s.mutex.Unlock()

	if s.conn == nil {
		return nil
	}

	unsubscribeMsg := map[string]interface{}{
		"method": "unsubscribe",
		"subscription": map[string]interface{}{
			"type": "orderUpdates",
			"user": address,
		},
	}

	return s.conn.WriteJSON(unsubscribeMsg)
}

func (s *Subscriber) run() {
	s.connect()

	reconnectDelay := reconnectInitial
loop:
	for {
		select {
		case <-s.ctx.Done():
			break loop
		case <-s.reconnect:
			select {
			case <-s.ctx.Done():
				break loop
			case <-time.After(reconnectDelay):
				logger.Infof("[HyperliquidSubscriber] 重新建立连接...")
				s.connect()

				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	s.stopChan <- struct{}{}
}

func (s *Subscriber) connect() {
	dialer := &websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.Dial(s.url, nil)
	if err != nil {
		logger.Errorf("[HyperliquidSubscriber] 连接失败, %v", err)
		s.scheduleReconnect()
		return
	}

	s.conn = conn
	logger.Infof("[HyperliquidSubscriber] 连接已建立")

	s.resubscribe()

	go s.readMessages()
}

func (s *Subscriber) resubscribe() {
	s.mutex.Lock()
	accounts := make(map[string]*Signer)
	for k, v := range s.accounts {
		accounts[k] = v
	}
	s.mutex.Unlock()

	for address := range accounts {
		subscribeMsg := map[string]interface{}{
			"method": "subscribe",
			"subscription": map[string]interface{}{
				"type": "orderUpdates",
				"user": address,
			},
		}
		s.conn.WriteJSON(subscribeMsg)
	}

	subscribeMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "allMids",
		},
	}
	s.conn.WriteJSON(subscribeMsg)
}

func (s *Subscriber) readMessages() {
	defer s.conn.Close()

	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.Errorf("[HyperliquidSubscriber] 读取出错, %v", err)
			s.scheduleReconnect()
			return
		}

		logger.Tracef("[HyperliquidSubscriber] 收到新消息, %s", data)

		var msg WsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Errorf("[HyperliquidSubscriber] 解析消息失败, %v", err)
			continue
		}

		s.handleMessage(msg)
	}
}

func (s *Subscriber) handleMessage(msg WsMessage) {
	switch msg.Channel {
	case "orderUpdates":
		s.handleOrderUpdates(msg.Data)
	case "userFills":
		s.handleUserFills(msg.Data)
	case "userEvents":
		s.handleUserEvents(msg.Data)
	case "allMids":
		s.handleAllMids(msg.Data)
	case "pong":
		// 心跳响应
	}
}

func (s *Subscriber) handleOrderUpdates(data interface{}) {
	orders, ok := data.([]interface{})
	if !ok || len(orders) == 0 {
		return
	}

	for _, o := range orders {
		orderData, ok := o.(map[string]interface{})
		if !ok {
			continue
		}

		orderMap, ok := orderData["order"].(map[string]interface{})
		if !ok {
			continue
		}

		coin, _ := orderMap["coin"].(string)
		side, _ := orderMap["side"].(string)
		limitPx, _ := orderMap["limitPx"].(string)
		sz, _ := orderMap["sz"].(string)
		oid, _ := orderMap["oid"].(float64)
		timestamp, _ := orderMap["timestamp"].(float64)
		status, _ := orderData["status"].(string)

		order := &exchange.Order{
			Symbol:     coin,
			OrderID:    fmt.Sprintf("%d", int64(oid)),
			Side:       ConvertOrderSide(side),
			Price:      ParseDecimal(limitPx),
			BaseAmount: ParseDecimal(sz),
			Timestamp:  int64(timestamp),
			Status:     ConvertOrderStatus(status),
		}

		key := fmt.Sprintf("order_%d", int64(oid))
		isSnapshot := !s.processed[key]
		s.processed[key] = true

		userOrders := &exchange.UserOrders{
			Exchange:   exchange.Hyperliquid,
			Account:    "",
			Orders:     []*exchange.Order{order},
			IsSnapshot: isSnapshot,
		}

		if s.subMsgChan != nil {
			s.subMsgChan <- exchange.SubMessage{
				Exchange:   exchange.Hyperliquid,
				UserOrders: userOrders,
			}
		}
	}
}

func (s *Subscriber) handleUserFills(data interface{}) {
	// 处理成交记录
}

func (s *Subscriber) handleUserEvents(data interface{}) {
	// 处理用户事件
}

func (s *Subscriber) handleAllMids(data interface{}) {
	mids, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	for coin, price := range mids {
		var priceVal float64
		switch p := price.(type) {
		case float64:
			priceVal = p
		case string:
			_, err := fmt.Sscanf(p, "%f", &priceVal)
			if err != nil {
				continue
			}
		default:
			continue
		}

		marketStats := &exchange.MarketStats{
			Symbol: coin,
			Price:  decimal.NewFromFloat(priceVal),
		}

		if s.subMsgChan != nil {
			s.subMsgChan <- exchange.SubMessage{
				Exchange:    exchange.Hyperliquid,
				MarketStats: marketStats,
			}
		}
	}
}

func (s *Subscriber) scheduleReconnect() {
	if s.ctx.Err() == nil {
		select {
		case s.reconnect <- struct{}{}:
		default:
		}
	}
}
