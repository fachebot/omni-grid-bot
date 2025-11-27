package paradex

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const (
	reconnectInitial = 1 * time.Second
	reconnectMax     = 30 * time.Second
)

type StoppedCallback func(dexAccount string)

type ParadexWS struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	url       string
	conn      *websocket.Conn
	proxy     config.Sock5Proxy
	reconnect chan struct{}

	userClient     *UserClient
	userOrdersChan chan<- exchange.UserOrders
	callback       StoppedCallback
}

func NewParadexWS(
	ctx context.Context,
	userClient *UserClient,
	userOrdersChan chan<- exchange.UserOrders,
	proxy config.Sock5Proxy,
	callback StoppedCallback,
) *ParadexWS {
	ctx, cancel := context.WithCancel(ctx)
	ws := &ParadexWS{
		ctx:            ctx,
		cancel:         cancel,
		url:            "wss://ws.api.prod.paradex.trade/v1",
		proxy:          proxy,
		reconnect:      make(chan struct{}, 1),
		userClient:     userClient,
		userOrdersChan: userOrdersChan,
		callback:       callback,
	}
	return ws
}

func (ws *ParadexWS) Stop() {
	if ws.stopChan == nil {
		return
	}

	logger.Infof("[ParadexWS-%s] 准备停止服务", ws.userClient.DexAccount())

	ws.cancel()

	if ws.conn != nil {
		ws.conn.Close()
	}

	<-ws.stopChan

	close(ws.stopChan)
	ws.stopChan = nil

	logger.Infof("[ParadexWS-%s] 服务已经停止", ws.userClient.DexAccount())
}

func (ws *ParadexWS) Start() {
	if ws.stopChan != nil {
		return
	}

	ws.stopChan = make(chan struct{}, 1)

	if ws.conn == nil {
		logger.Infof("[ParadexWS-%s] 开始运行服务", ws.userClient.DexAccount())
		go ws.run()
	}
}

func (ws *ParadexWS) run() {
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
				logger.Infof("[ParadexWS-%s] 重新建立连接...", ws.userClient.DexAccount())
				ws.connect()

				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	if ws.callback != nil {
		ws.callback(ws.userClient.DexAccount())
	}

	ws.stopChan <- struct{}{}
}

func (ws *ParadexWS) connect() {
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
		logger.Errorf("[ParadexWS-%s] 连接失败, %v", ws.userClient.DexAccount(), err)
		ws.scheduleReconnect()
		return
	}

	ws.conn = conn
	logger.Infof("[ParadexWS-%s] 连接已建立", ws.userClient.DexAccount())

	go ws.readMessages()
}

func (ws *ParadexWS) readMessages() {
	defer ws.conn.Close()
	account := ws.userClient.DexAccount()

	ctx, cancel := context.WithCancel(ws.ctx)
	defer cancel()
	go ws.heartbeat(ctx)

	jwtToken, err := ws.userClient.EnsureJwtToken(ws.ctx)
	if err != nil {
		logger.Errorf("[ParadexWS-%s] 用户鉴权失败, %v", account, err)
		ws.Stop()
		return
	}

	message := fmt.Sprintf(`{"jsonrpc":"2.0","method":"auth","params":{"bearer":"%s"},"id":1}`, jwtToken)
	err = ws.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		logger.Errorf("[ParadexWS-%s] 用户鉴权失败, %v", account, err)
		ws.Stop()
		return
	}

	message = `{"jsonrpc":"2.0","method":"subscribe","params":{"channel":"orders.ALL"},"id":1}`
	err = ws.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		logger.Errorf("[ParadexWS-%s] 订阅订单活动失败, %v", account, err)
		ws.Stop()
		return
	}

	isFirstOrder := true
	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.Warnf("[ParadexWS-%s] 读取出错, %v", account, err)
			ws.scheduleReconnect()
			return
		}

		logger.Debugf("[ParadexWS-%s] 收到新消息, %s", account, data)

		var res JsonRpcMessage
		if err = json.Unmarshal(data, &res); err != nil {
			logger.Warnf("[ParadexWS-%s] 解析响应失败, %s, %v", account, string(data), err)
			continue
		}

		if res.Error != nil {
			logger.Errorf("[ParadexWS-%s] 请求处理失败, %s, %v", account, string(res.Error), err)
			ws.Stop()
			return
		}

		if res.Method == "subscription" {
			var subscription SubscriptionPayload
			if err = json.Unmarshal(res.Params, &subscription); err != nil {
				logger.Warnf("[ParadexWS-%s] 解析订阅数据失败, %s, %v", account, string(res.Params), err)
				continue
			}

			switch subscription.Channel {
			case "orders.ALL":
				var ord Order
				if err = json.Unmarshal(subscription.Data, &ord); err != nil {
					logger.Warnf("[ParadexWS-%s] 解析订阅订单数据失败, %s, %v", account, string(subscription.Data), err)
					continue
				}

				symbol, err := ParseUsdPerpMarket(ord.Market)
				if err != nil {
					continue
				}

				var clientId int64
				if ord.ClientID != "" {
					clientId, err = strconv.ParseInt(ord.ClientID, 10, 64)
					if err != nil {
						clientId = 0
					}
				}

				filledQuoteAmount := decimal.Zero
				if ord.AvgFillPrice != "" {
					avgFillPrice, err := decimal.NewFromString(ord.AvgFillPrice)
					if err == nil {
						filledQuoteAmount = ord.Size.Mul(avgFillPrice)
					}
				}

				userOrders := exchange.UserOrders{
					Exchange: exchange.Paradex,
					Account:  account,
					Orders: []*exchange.Order{
						{
							Symbol:            symbol,
							OrderID:           ord.ID,
							ClientOrderID:     clientId,
							Side:              lo.If(ord.Side == OrderSideSell, order.SideSell).Else(order.SideBuy),
							Price:             ord.Price,
							BaseAmount:        ord.Size,
							FilledBaseAmount:  ord.Size.Sub(ord.RemainingSize),
							FilledQuoteAmount: filledQuoteAmount,
							Timestamp:         ord.LastUpdatedAt,
							Status:            ConvertOrderStatus(&ord),
						},
					},
					IsSnapshot: isFirstOrder,
				}

				isFirstOrder = false
				ws.userOrdersChan <- userOrders
			}
		}
	}
}

func (ws *ParadexWS) scheduleReconnect() {
	if ws.ctx.Err() == nil {
		select {
		case ws.reconnect <- struct{}{}:
		default:
		}
	}
}

func (ws *ParadexWS) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if err := ws.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.Errorf("[ParadexWS-%s] 发送心跳消息失败, %v", ws.userClient.DexAccount(), err)
				return
			}

			duration := time.Second * 20
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}
