package variational

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

type VariationalPubWS struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	url       string
	conn      *websocket.Conn
	proxy     config.Sock5Proxy
	reconnect chan struct{}

	marketStatsChan chan<- exchange.MarketStats
}

func NewVariationalPubWS(
	ctx context.Context,
	marketStatsChan chan<- exchange.MarketStats,
	proxy config.Sock5Proxy,
) *VariationalPubWS {
	ctx, cancel := context.WithCancel(ctx)
	ws := &VariationalPubWS{
		ctx:             ctx,
		cancel:          cancel,
		url:             "wss://omni-ws-server.prod.ap-northeast-1.variational.io/prices",
		proxy:           proxy,
		reconnect:       make(chan struct{}, 1),
		marketStatsChan: marketStatsChan,
	}
	return ws
}

func (ws *VariationalPubWS) Stop() {
	if ws.stopChan == nil {
		return
	}

	logger.Infof("[VariationalPubWS] 准备停止服务")

	ws.cancel()
	if ws.conn != nil {
		ws.conn.Close()
	}

	<-ws.stopChan

	close(ws.stopChan)
	ws.stopChan = nil

	logger.Infof("[VariationalPubWS] 服务已经停止")
}

func (ws *VariationalPubWS) Start() {
	if ws.stopChan != nil {
		return
	}

	ws.stopChan = make(chan struct{}, 1)

	if ws.conn == nil {
		logger.Infof("[VariationalPubWS] 开始运行服务")
		go ws.run()
	}
}

func (ws *VariationalPubWS) WaitUntilConnected() {
	for ws.conn == nil {
		time.Sleep(time.Second * 1)
	}
}

func (ws *VariationalPubWS) SubscribeMarketStats(symbol string) error {
	if ws.conn == nil {
		return errors.New("connection is not established")
	}

	message := fmt.Sprintf(`{"action":"subscribe","instruments":[{"underlying":"%s","instrument_type":"perpetual_future","settlement_asset":"USDC","funding_interval_s":3600}]}`, symbol)
	return ws.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (ws *VariationalPubWS) UnsubscribeMarketStats(symbol string) error {
	if ws.conn == nil {
		return errors.New("connection is not established")
	}

	message := fmt.Sprintf(`{"action":"unsubscribe","instruments":[{"underlying":"%s","instrument_type":"perpetual_future","settlement_asset":"USDC","funding_interval_s":3600}]}`, symbol)
	return ws.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (ws *VariationalPubWS) run() {
	ws.connect()

	reconnectDelay := reconnectInitial
loop:
	for {
		select {
		case <-ws.ctx.Done():
			break loop
		case <-ws.reconnect:
			select {
			case <-ws.ctx.Done():
				break loop
			case <-time.After(reconnectDelay):
				logger.Infof("[VariationalPubWS] 重新建立连接...")
				ws.connect()

				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	ws.stopChan <- struct{}{}
}

func (ws *VariationalPubWS) connect() {
	sock5Proxy := ""
	if ws.proxy.Enable {
		sock5Proxy = fmt.Sprintf("%s:%d", ws.proxy.Host, ws.proxy.Port)
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
			return netDial(ws.ctx, network, addr)
		},
		HandshakeTimeout: 45 * time.Second,
	}
	conn, _, err := dialer.Dial(ws.url, nil)
	if err != nil {
		logger.Errorf("[VariationalPubWS] 连接失败, %v", err)
		ws.scheduleReconnect()
		return
	}

	ws.conn = conn
	logger.Infof("[VariationalPubWS] 连接已建立")

	go ws.readMessages()
}

func (ws *VariationalPubWS) readMessages() {
	defer ws.conn.Close()

	// 定时心跳
	ctx, cancel := context.WithCancel(ws.ctx)
	defer cancel()
	go ws.heartbeat(ctx)

	// 消息循环
	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.Warnf("[VariationalPubWS] 读取出错, %v", err)
			ws.scheduleReconnect()
			return
		}

		logger.Tracef("[VariationalPubWS] 收到新消息, %s", data)

		var subscription SubscriptionPayload
		if err = json.Unmarshal(data, &subscription); err != nil {
			logger.Warnf("[VariationalPubWS] 解析订阅数据失败, %s, %v", data, err)
			continue
		}

		const priceChannelPrefix = "instrument_price:"
		if strings.HasPrefix(subscription.Channel, priceChannelPrefix) && subscription.Pricing != nil {
			slice := strings.Split(subscription.Channel[len(priceChannelPrefix):], "-")
			if len(slice) != 4 {
				continue
			}

			symbol := slice[1]
			marketStats := exchange.MarketStats{
				Symbol:    symbol,
				Price:     subscription.Pricing.Price,
				MarkPrice: subscription.Pricing.Price,
			}

			logger.Tracef("[VariationalPubWS] 分发 MarketStats 数据, %+v", marketStats)
			ws.marketStatsChan <- marketStats
		}
	}
}

func (ws *VariationalPubWS) scheduleReconnect() {
	if ws.ctx.Err() == nil {
		select {
		case ws.reconnect <- struct{}{}:
		default:
		}
	}
}

func (ws *VariationalPubWS) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if err := ws.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.Errorf("[VariationalPubWS] 发送心跳消息失败, %v", err)
				return
			}

			duration := time.Second * 20
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}
