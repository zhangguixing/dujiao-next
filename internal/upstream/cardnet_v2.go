package upstream

import (
	"context"
	"errors"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

const CardNetMaxOrderQuantity = 100

var ErrAuthenticationPaused = errors.New("upstream authentication paused; verify credentials with a manual ping")

type authenticationPausedError struct{}

func (authenticationPausedError) Error() string     { return ErrAuthenticationPaused.Error() }
func (authenticationPausedError) Is(err error) bool { return err == ErrAuthenticationPaused }
func (authenticationPausedError) Retryable() bool   { return false }

func AuthenticationPausedError() error { return authenticationPausedError{} }

// CardNetV2Adapter implements CardNet's balance-based upstream API. CardNet
// deliberately exposes the same resource contract as Dujiao-Next, so the
// shared adapter can be reused while switching only the signing scheme.
type CardNetV2Adapter struct {
	*DujiaoNextAdapter
}

func NewCardNetV2Adapter(conn *siteconnectiondomain.Connection, uploadsDir string) *CardNetV2Adapter {
	adapter := NewDujiaoNextAdapter(conn, uploadsDir)
	adapter.cardNetV2 = true
	return &CardNetV2Adapter{DujiaoNextAdapter: adapter}
}

// CreateOrder omits CardNet-unsupported callback and manual-form payloads.
// CardNet fulfills automatic card products synchronously; the existing
// procurement polling workflow retrieves the delivery payload afterwards.
func (a *CardNetV2Adapter) CreateOrder(ctx context.Context, req CreateUpstreamOrderReq) (*CreateUpstreamOrderResp, error) {
	if req.Quantity < 1 || req.Quantity > CardNetMaxOrderQuantity {
		retryable := false
		return &CreateUpstreamOrderResp{ErrorCode: "invalid_request", ErrorMessage: "invalid_request", Retryable: &retryable}, nil
	}
	req.CallbackURL = ""
	req.ManualFormData = nil
	return a.DujiaoNextAdapter.CreateOrder(ctx, req)
}

// SetAuthenticationGuard keeps existing adapters subject to the persisted pause.
func (a *CardNetV2Adapter) SetAuthenticationGuard(before func() error, failed func() error) {
	a.beforeRequest, a.authenticationFailed = before, failed
}

// Ping is explicitly invoked by an operator and is the only recovery probe.
func (a *CardNetV2Adapter) Ping(ctx context.Context) (*PingResult, error) {
	return a.DujiaoNextAdapter.Ping(context.WithValue(ctx, cardNetManualPingKey{}, true))
}

var _ Adapter = (*CardNetV2Adapter)(nil)
