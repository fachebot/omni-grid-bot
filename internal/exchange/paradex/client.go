package paradex

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/dontpanicdao/caigo"
	"github.com/dontpanicdao/caigo/types"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/golang-jwt/jwt/v5"
	gocache "github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

const (
	PARADEX_HTTP_URL = "https://api.prod.paradex.trade/v1"
)

type Client struct {
	endpoint     string
	httpClient   *http.Client
	systemConfig *SystemConfigRes

	jwtTokenCache *gocache.Cache
	sfGroup       singleflight.Group
}

func NewClient(httpClient *http.Client) *Client {
	c := Client{
		endpoint:      PARADEX_HTTP_URL,
		httpClient:    httpClient,
		jwtTokenCache: gocache.New(300*time.Second, 600*time.Second),
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

func (c *Client) GetMarkets(ctx context.Context) (*MarketRes, error) {
	var res MarketRes
	var errRes *ErrorRes
	err := requests.URL(fmt.Sprintf("%s/markets", c.endpoint)).Client(c.httpClient).
		Header("Content-Type", "application/json").
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetMarketSummary(ctx context.Context, market string) (*MarketSummaryRes, error) {
	var errRes *ErrorRes
	var res MarketSummaryRes
	err := requests.URL(fmt.Sprintf("%s/markets/summary?market=%s", c.endpoint, market)).Client(c.httpClient).
		Header("Content-Type", "application/json").
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetSymtemConfig(ctx context.Context) (*SystemConfigRes, error) {
	var errRes *ErrorRes
	var res SystemConfigRes
	err := requests.URL(fmt.Sprintf("%s/system/config", c.endpoint)).Client(c.httpClient).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *Client) LoadSymtemConfig(ctx context.Context) (*SystemConfigRes, error) {
	if c.systemConfig != nil {
		return c.systemConfig, nil
	}

	systemConfig, err := c.GetSymtemConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load system config: %w", err)
	}

	c.systemConfig = systemConfig
	return systemConfig, nil
}

func (c *Client) GetJwtToken(ctx context.Context, dexAccount, dexPrivateKey string) (string, error) {
	systemConfig, err := c.LoadSymtemConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load system config for JWT token generation: %w", err)
	}

	dexPrivateKeyBN := types.HexToBN(dexPrivateKey)
	if dexPrivateKeyBN == nil {
		return "", fmt.Errorf("invalid private key")
	}

	dexPublicKeyBN, _, err := caigo.Curve.PrivateToPoint(dexPrivateKeyBN)
	if err != nil {
		return "", fmt.Errorf("failed to derive public key from private key (invalid private key?): %w", err)
	}
	dexPublicKey := types.BigToHex(dexPublicKeyBN)

	now := time.Now().Unix()
	timestampStr := strconv.FormatInt(now, 10)
	expirationStr := strconv.FormatInt(now+300, 10)

	sc := caigo.StarkCurve{}
	message := &AuthPayload{
		Method:     "POST",
		Path:       "/v1/auth",
		Body:       "",
		Timestamp:  timestampStr,
		Expiration: expirationStr,
	}
	typedData, err := NewVerificationTypedData(VerificationTypeAuth, systemConfig.ChainId)
	if err != nil {
		return "", fmt.Errorf("failed to create verification typed data for chainId=%s: %w", systemConfig.ChainId, err)
	}

	domEnc, err := typedData.GetTypedMessageHash("StarkNetDomain", typedData.Domain, sc)
	if err != nil {
		return "", fmt.Errorf("failed to get domain encoded hash: %w", err)
	}

	dexAccountAddressBN := types.HexToBN(dexAccount)
	if dexAccountAddressBN == nil {
		return "", fmt.Errorf("invalid public key")
	}

	messageHash, err := GnarkGetMessageHash(typedData, domEnc, dexAccountAddressBN, message, sc)
	if err != nil {
		return "", fmt.Errorf("failed to compute message hash for account=%s: %w", dexAccount, err)
	}

	r, s, err := GnarkSign(messageHash, dexPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	signature, err := GetSignatureStr(r, s)
	if err != nil {
		return "", fmt.Errorf("failed to generate signature string: %w", err)
	}

	var res AuthRes
	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/auth/%s", c.endpoint, dexPublicKey)).Client(c.httpClient).Post().
		Header("Content-Type", "application/json").
		Header("PARADEX-STARKNET-ACCOUNT", dexAccount).
		Header("PARADEX-STARKNET-SIGNATURE", signature).
		Header("PARADEX-TIMESTAMP", timestampStr).
		Header("PARADEX-SIGNATURE-EXPIRATION", expirationStr).
		Header("PARADEX-AUTHORIZE-ISOLATED-MARKETS", "true").
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return "", errRes
		}
		return "", err
	}

	return res.JwtToken, nil
}

func (c *Client) EnsureJwtToken(ctx context.Context, dexAccount, dexPrivateKey string) (string, error) {
	if v, ok := c.jwtTokenCache.Get(dexAccount); ok {
		if token, isString := v.(string); isString && token != "" {
			return token, nil
		}
	}

	cacheKey := dexAccount
	result, err, _ := c.sfGroup.Do(cacheKey, func() (any, error) {
		if v, ok := c.jwtTokenCache.Get(dexAccount); ok {
			if token, isString := v.(string); isString && token != "" {
				return token, nil
			}
		}

		jwtToken, err := c.GetJwtToken(ctx, dexAccount, dexPrivateKey)
		if err != nil {
			return "", fmt.Errorf("failed to get JWT token: %w", err)
		}

		expirationTime, err := parseExpirationTime(jwtToken)
		if err != nil {
			expirationTime = time.Now().Add(3 * time.Minute)
			logger.Warnf("[paradex.Client] 未找到Token过期时间, account: %s", dexAccount)
		}

		c.jwtTokenCache.Set(dexAccount, jwtToken, time.Until(expirationTime.Add(-10*time.Second)))

		return jwtToken, nil
	})

	if err != nil {
		return "", err
	}

	return result.(string), nil
}
