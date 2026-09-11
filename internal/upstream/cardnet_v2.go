package upstream

import (
	"context"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

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
	req.CallbackURL = ""
	req.ManualFormData = nil
	return a.DujiaoNextAdapter.CreateOrder(ctx, req)
}

var _ Adapter = (*CardNetV2Adapter)(nil)
