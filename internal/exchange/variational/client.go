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
	"golang.org/x/sync/singleflight"
)

const (
	VARIATIONAL_HTTP_URL = "https://omni.variational.io/api"
)

type Client struct {
	endpoint string
	client   *surf.Client

	jwtTokenCache *gocache.Cache
	sfGroup       singleflight.Group
}

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
	}
	return &c
}

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

func (c *Client) GenerateSigningData(ctx context.Context, address string) (string, error) {
	url := fmt.Sprintf("%s/auth/generate_signing_data", c.endpoint)
	payload := fmt.Sprintf(`{"address":"%s"}`, common.HexToAddress(address).Hex())
	res := c.client.Post(g.String(url), payload).WithContext(ctx).Do()

	var r GenerateSigningDataRes
	if err := c.parseRespone(res, &r); err != nil {
		return "", err
	}
	return r.Message, nil
}

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

func (c *Client) parseRespone(res g.Result[*surf.Response], data any) error {
	if res.Err() != nil {
		return res.Err()
	}

	defer res.Ok().Body.Close()

	statusCode := res.Ok().StatusCode
	if statusCode < 200 || statusCode >= 300 {
		var errRes ErrorRes
		if err := res.Ok().Body.JSON(&errRes); err == nil {
			return &errRes
		}

		return fmt.Errorf("http error - status_code: %d, body: %s", statusCode, res.Ok().Body.String())
	}

	content := res.Ok().Body.Bytes()
	if err := json.Unmarshal(content, data); err != nil {
		return fmt.Errorf("invalid json content, body: %s", string(content))
	}
	return nil
}
