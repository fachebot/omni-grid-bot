package variational

import (
	"context"
	"encoding/json"
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

const (
	reconnectInitial = 1 * time.Second
	reconnectMax     = 30 * time.Second
)

type StoppedCallback func(dexAccount string)

type VariationalWS struct {
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

func NewVariationalWS(
	ctx context.Context,
	userClient *UserClient,
	userOrdersChan chan<- exchange.UserOrders,
	proxy config.Sock5Proxy,
	callback StoppedCallback,
) *VariationalWS {
	ctx, cancel := context.WithCancel(ctx)
	ws := &VariationalWS{
		ctx:            ctx,
		cancel:         cancel,
		url:            "wss://omni-ws-server.prod.ap-northeast-1.variational.io/portfolio",
		proxy:          proxy,
		reconnect:      make(chan struct{}, 1),
		userClient:     userClient,
		userOrdersChan: userOrdersChan,
		callback:       callback,
	}
	return ws
}

func (ws *VariationalWS) Stop() {
	if ws.stopChan == nil {
		return
	}

	logger.Infof("[VariationalWS-%s] 准备停止服务", ws.userClient.EthAccount())

	ws.cancel()
	if ws.conn != nil {
		ws.conn.Close()
	}

	<-ws.stopChan

	close(ws.stopChan)
	ws.stopChan = nil

	logger.Infof("[VariationalWS-%s] 服务已经停止", ws.userClient.EthAccount())
}

func (ws *VariationalWS) Start() {
	if ws.stopChan != nil {
		return
	}

	ws.stopChan = make(chan struct{}, 1)

	if ws.conn == nil {
		logger.Infof("[VariationalWS-%s] 开始运行服务", ws.userClient.EthAccount())
		go ws.run()
	}
}

func (ws *VariationalWS) run() {
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
				logger.Infof("[VariationalWS-%s] 重新建立连接...", ws.userClient.EthAccount())
				ws.connect()

				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	if ws.callback != nil {
		ws.callback(ws.userClient.EthAccount())
	}

	ws.stopChan <- struct{}{}
}

func (ws *VariationalWS) connect() {
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
		logger.Errorf("[VariationalWS-%s] 连接失败, %v", ws.userClient.EthAccount(), err)
		ws.scheduleReconnect()
		return
	}

	ws.conn = conn
	logger.Infof("[VariationalWS-%s] 连接已建立", ws.userClient.EthAccount())

	go ws.readMessages()
}

func (ws *VariationalWS) readMessages() {
	defer ws.conn.Close()
	account := ws.userClient.EthAccount()

	jwtToken, err := ws.userClient.EnsureJwtToken(ws.ctx)
	if err != nil {
		logger.Errorf("[VariationalWS-%s] 用户鉴权失败, %v", account, err)
		ws.Stop()
		return
	}

	// 用户鉴权
	message := fmt.Sprintf(`{"claims":"%s"}`, jwtToken)
	err = ws.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		logger.Errorf("[VariationalWS-%s] 用户鉴权失败, %v", account, err)
		ws.Stop()
		return
	}

	// 定时心跳
	ctx, cancel := context.WithCancel(ws.ctx)
	defer cancel()
	go ws.heartbeat(ctx)

	// 消息循环
	first := true
	localSequenceMap := make(map[string]int64)
	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.Warnf("[VariationalWS-%s] 读取出错, %v", account, err)
			ws.scheduleReconnect()
			return
		}

		var result PoolPortfolioResult
		if err = json.Unmarshal(data, &result); err != nil {
			logger.Warnf("[VariationalWS-%s] 解析结果失败, %s, %v", account, string(data), err)
			continue
		}

		// 检查仓位变化
		positionChanged := false
		positionSet := make(map[string]struct{})
		for _, item := range result.Positions {
			underlying := item.PositionInfo.Instrument.Underlying
			positionSet[underlying] = struct{}{}
			lastLocalSequence, ok := localSequenceMap[underlying]
			localSequenceMap[underlying] = item.PositionInfo.LastLocalSequence

			if !ok {
				positionChanged = true
				continue
			}

			if item.PositionInfo.LastLocalSequence != lastLocalSequence {
				positionChanged = true
				continue
			}
		}

		for symbol := range localSequenceMap {
			_, ok := positionSet[symbol]
			if !ok {
				positionChanged = true
				delete(localSequenceMap, symbol)
			}
		}

		// 触发同步订单
		if first || positionChanged {
			first = false
			userOrders := exchange.UserOrders{
				Exchange:   exchange.Variational,
				Account:    account,
				Orders:     []*exchange.Order{},
				IsSnapshot: true,
			}
			ws.userOrdersChan <- userOrders
		}
	}
}

func (ws *VariationalWS) scheduleReconnect() {
	if ws.ctx.Err() == nil {
		select {
		case ws.reconnect <- struct{}{}:
		default:
		}
	}
}

func (ws *VariationalWS) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if err := ws.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.Errorf("[VariationalWS-%s] 发送心跳消息失败, %v", ws.userClient.EthAccount(), err)
				return
			}

			duration := time.Second * 20
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}
