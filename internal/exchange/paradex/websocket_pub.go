package paradex

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

type ParadexPubWS struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	url       string
	conn      *websocket.Conn
	proxy     config.Sock5Proxy
	reconnect chan struct{}

	marketStatsChan chan<- exchange.MarketStats
}

func NewParadexPubWS(
	ctx context.Context,
	marketStatsChan chan<- exchange.MarketStats,
	proxy config.Sock5Proxy,
) *ParadexPubWS {
	ctx, cancel := context.WithCancel(ctx)
	ws := &ParadexPubWS{
		ctx:             ctx,
		cancel:          cancel,
		url:             "wss://ws.api.prod.paradex.trade/v1",
		proxy:           proxy,
		reconnect:       make(chan struct{}, 1),
		marketStatsChan: marketStatsChan,
	}
	return ws
}

func (ws *ParadexPubWS) Stop() {
	if ws.stopChan == nil {
		return
	}

	logger.Infof("[ParadexPubWS] 准备停止服务")

	ws.cancel()
	if ws.conn != nil {
		ws.conn.Close()
	}

	<-ws.stopChan

	close(ws.stopChan)
	ws.stopChan = nil

	logger.Infof("[ParadexPubWS] 服务已经停止")
}

func (ws *ParadexPubWS) Start() {
	if ws.stopChan != nil {
		return
	}

	ws.stopChan = make(chan struct{}, 1)

	if ws.conn == nil {
		logger.Infof("[ParadexPubWS] 开始运行服务")
		go ws.run()
	}
}

func (ws *ParadexPubWS) WaitUntilConnected() {
	for ws.conn == nil {
		time.Sleep(time.Second * 1)
	}
}

func (ws *ParadexPubWS) SubscribeMarketStats(symbol string) error {
	if ws.conn == nil {
		return errors.New("connection is not established")
	}

	marketSymbol := FormatUsdPerpMarket(symbol)
	message := fmt.Sprintf(`{"jsonrpc":"2.0","method":"subscribe","params":{"channel":"markets_summary.%s"},"id":1}`, marketSymbol)
	return ws.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (ws *ParadexPubWS) UnsubscribeMarketStats(symbol string) error {
	if ws.conn == nil {
		return errors.New("connection is not established")
	}

	marketSymbol := FormatUsdPerpMarket(symbol)
	message := fmt.Sprintf(`{"jsonrpc":"2.0","method":"unsubscribe","params":{"channel":"markets_summary.%s"},"id":1}`, marketSymbol)
	return ws.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (ws *ParadexPubWS) run() {
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
				logger.Infof("[ParadexPubWS] 重新建立连接...")
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

func (ws *ParadexPubWS) connect() {
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
		logger.Errorf("[ParadexPubWS] 连接失败, %v", err)
		ws.scheduleReconnect()
		return
	}

	ws.conn = conn
	logger.Infof("[ParadexPubWS] 连接已建立")

	go ws.readMessages()
}

func (ws *ParadexPubWS) readMessages() {
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
			logger.Warnf("[ParadexPubWS] 读取出错, %v", err)
			ws.scheduleReconnect()
			return
		}

		logger.Tracef("[ParadexPubWS] 收到新消息, %s", data)

		var res JsonRpcMessage
		if err = json.Unmarshal(data, &res); err != nil {
			logger.Warnf("[ParadexPubWS] 解析响应失败, %s, %v", string(data), err)
			continue
		}

		if res.Error != nil {
			logger.Errorf("[ParadexPubWS] 请求处理失败, %s, %v", string(res.Error), err)
			ws.Stop()
			return
		}

		if res.Method == "subscription" {
			var subscription SubscriptionPayload
			if err = json.Unmarshal(res.Params, &subscription); err != nil {
				logger.Warnf("[ParadexPubWS] 解析订阅数据失败, %s, %v", string(res.Params), err)
				continue
			}

			if strings.HasPrefix(subscription.Channel, "markets_summary") {
				var v MarketSummary
				if err = json.Unmarshal(subscription.Data, &v); err != nil {
					logger.Warnf("[ParadexPubWS] 解析订阅市场数据失败, %s, %v", string(subscription.Data), err)
					continue
				}

				symbol, err := ParseUsdPerpMarket(v.Symbol)
				if err != nil {
					continue
				}

				marketStats := exchange.MarketStats{
					Symbol:    symbol,
					Price:     v.LastTradedPrice,
					MarkPrice: v.MarkPrice,
				}

				logger.Tracef("[ParadexPubWS] 分发 MarketStats 数据, %+v", marketStats)
				ws.marketStatsChan <- marketStats
			}
		}
	}
}

func (ws *ParadexPubWS) scheduleReconnect() {
	if ws.ctx.Err() == nil {
		select {
		case ws.reconnect <- struct{}{}:
		default:
		}
	}
}

func (ws *ParadexPubWS) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if err := ws.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.Errorf("[ParadexPubWS] 发送心跳消息失败, %v", err)
				return
			}

			duration := time.Second * 20
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}
