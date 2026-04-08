package variational

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/enetx/g"
	"github.com/enetx/surf"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/util/ethutil"
	"github.com/golang-jwt/jwt/v5"
	gocache "github.com/patrickmn/go-cache"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/singleflight"
)

// VARIATIONAL_HTTP_URL Variational主网API地址
const (
	VARIATIONAL_HTTP_URL = "https://omni.variational.io/api"
)

// Client Variational交易所HTTP客户端
// 用于与Variational交易所API进行交互，支持Ethereum签名认证
type Client struct {
	endpoint string       // API端点地址
	client   *surf.Client // HTTP客户端

	jwtTokenCache *gocache.Cache     // JWT令牌缓存
	sfGroup       singleflight.Group // 单飞组(防止并发刷新token)
	rateLimiter   *RateLimiter       // 速率限制器
}

// NewClient 创建Variational交易所客户端
func NewClient(proxy config.Sock5Proxy) *Client {
	builder := surf.NewClient().
		Builder().
		Impersonate().
		Chrome()
	if proxy.Enable {
		builder = builder.Proxy(fmt.Sprintf("socks5://%s:%d", proxy.Host, proxy.Port))
	}

	c := Client{
		client:        builder.Build(),
		endpoint:      VARIATIONAL_HTTP_URL,
		jwtTokenCache: gocache.New(24*time.Hour, 24*time.Hour*2),
		rateLimiter:   NewRateLimiter(1.0, 1),
	}
	return &c
}

// parseExpirationTime 解析JWT令牌过期时间
func parseExpirationTime(jwtToken string) (time.Time, error) {
	token, _, err := jwt.NewParser().ParseUnverified(jwtToken, jwt.MapClaims{})
	if err != nil {
		return time.Time{}, err
	}

	v, err := token.Claims.GetExpirationTime()
	if err != nil {
		return time.Time{}, err
	}

	return v.Time, nil
}

// Auth 用户认证
// 使用Ethereum私钥签名进行身份验证，返回JWT令牌
func (c *Client) Auth(ctx context.Context, account, privateKey string) (string, error) {
	pk, address, err := ethutil.ParsePrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	if !strings.EqualFold(account, address.Hex()) {
		return "", errors.New("address and private key do not match")
	}

	signingData, err := c.GenerateSigningData(ctx, address.Hex())
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(signingData), signingData)
	hash := crypto.Keccak256Hash([]byte(msg))
	sig, err := crypto.Sign(hash.Bytes(), pk)
	if err != nil {
		return "", err
	}
	sig[64] += 27
	signature := hexutil.Encode(sig)

	url := fmt.Sprintf("%s/auth/login", c.endpoint)
	payload := fmt.Sprintf(`{"address":"%s","signed_message":"%s"}`, address, signature)
	res := c.client.Post(g.String(url), payload).WithContext(ctx).Do()

	var r LoginRes
	if err := c.parseRespone(res, &r); err != nil {
		return "", err
	}
	return r.Token, nil
}

// SimpleQuote 获取简单报价
// 用于获取当前市场价格信息
func (c *Client) SimpleQuote(ctx context.Context, symbol string, qty decimal.Decimal) (*SimpleQuoteRes, error) {
	url := fmt.Sprintf("%s/quotes/simple", c.endpoint)
	payload := fmt.Sprintf(`{"instrument":{"underlying":"%s","funding_interval_s":3600,"settlement_asset":"USDC","instrument_type":"perpetual_future"},"qty":"%s"}`, symbol, qty)
	res := c.client.Post(g.String(url), payload).WithContext(ctx).Do()

	var r SimpleQuoteRes
	if err := c.parseRespone(res, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// GenerateSigningData 生成签名数据
// 获取用于签名的消息数据
func (c *Client) GenerateSigningData(ctx context.Context, address string) (string, error) {
	url := fmt.Sprintf("%s/auth/generate_signing_data", c.endpoint)
	payload := fmt.Sprintf(`{"address":"%s"}`, common.HexToAddress(address).Hex())
	res := c.client.Post(g.String(url), payload).WithContext(ctx).Do()

	if err := res.Err(); err != nil {
		return "", err
	}

	return res.Ok().Body.String().Std(), nil
}

// EnsureJwtToken 确保JWT令牌有效
// 如果缓存中有有效令牌则直接返回，否则重新获取
func (c *Client) EnsureJwtToken(ctx context.Context, account, privateKey string) (string, error) {
	cacheKey := account
	if v, ok := c.jwtTokenCache.Get(cacheKey); ok {
		if token, isString := v.(string); isString && token != "" {
			return token, nil
		}
	}

	result, err, _ := c.sfGroup.Do(cacheKey, func() (any, error) {
		if v, ok := c.jwtTokenCache.Get(cacheKey); ok {
			if token, isString := v.(string); isString && token != "" {
				return token, nil
			}
		}

		jwtToken, err := c.Auth(ctx, account, privateKey)
		if err != nil {
			return "", fmt.Errorf("failed to get JWT token: %w", err)
		}

		expirationTime, err := parseExpirationTime(jwtToken)
		if err != nil {
			expirationTime = time.Now().Add(5 * time.Minute)
			logger.Warnf("[variational.Client] 未找到Token过期时间, account: %s", account)
		}

		c.jwtTokenCache.Set(account, jwtToken, time.Until(expirationTime.Add(-10*time.Second)))
		return jwtToken, nil
	})

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// parseRespone 解析HTTP响应
func (c *Client) parseRespone(res g.Result[*surf.Response], data any) error {
	if res.Err() != nil {
		return res.Err()
	}

	defer res.Ok().Body.Close()

	content := res.Ok().Body.Bytes()
	statusCode := res.Ok().StatusCode
	if statusCode < 200 || statusCode >= 300 {
		var errRes ErrorRes
		if err := json.Unmarshal(content, &errRes); err == nil {
			return &errRes
		}

		return fmt.Errorf("http error - status_code: %d, body: %s", statusCode, string(content))
	}

	if data != nil {
		if err := json.Unmarshal(content, data); err != nil {
			return fmt.Errorf("invalid json content, body: %s", string(content))
		}
	}
	return nil
}
