package lighter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	lighterclient "github.com/elliottech/lighter-go/client"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/order"
)

func ConvertOrderStatus(status OrderStatus) order.Status {
	switch status {
	case OrderStatusInProgress:
		return order.StatusInProgress
	case OrderStatusPending:
		return order.StatusPending
	case OrderStatusOpen:
		return order.StatusOpen
	case OrderStatusFilled:
		return order.StatusFilled
	default:
		if strings.HasPrefix(string(status), "canceled") {
			return order.StatusCanceled
		}
		return order.StatusOpen
	}
}

func parseResultStatus(respBody []byte) error {
	resultStatus := &lighterclient.ResultCode{}
	if err := json.Unmarshal(respBody, resultStatus); err != nil {
		return err
	}
	if resultStatus.Code != lighterclient.CodeOK {
		return errors.New(resultStatus.Message)
	}
	return nil
}

func getAndParseL2HTTPResponse(ctx context.Context, httpClient *http.Client, endpoint, path string, params map[string]any, result interface{}) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	u.Path = path

	q := u.Query()
	for k, v := range params {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	if err = parseResultStatus(body); err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}
	return nil
}
