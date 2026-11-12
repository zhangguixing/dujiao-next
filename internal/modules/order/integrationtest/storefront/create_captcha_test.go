package storefront_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	captcha "github.com/dujiao-next/internal/modules/captcha/contract"
	captchahttp "github.com/dujiao-next/internal/modules/captcha/transport/http"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderhttp "github.com/dujiao-next/internal/modules/order/transport/http"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

func TestStorefrontOrderCreationVerifiesCaptcha(t *testing.T) {
	orders := &captchaTestOrderCreator{}
	handler := orderhttp.NewCreateHandler(orders, nil, orderCaptchaStub{}, nil)
	router := gin.New()
	user := router.Group("", func(c *gin.Context) { c.Set("user_id", uint(1)) })
	guest := router.Group("/guest")
	orderhttp.RegisterUserCreateRoute(user, handler)
	orderhttp.RegisterUserCreateAndPayRoute(user, handler)
	orderhttp.RegisterGuestCreateRoute(guest, handler)
	orderhttp.RegisterGuestCreateAndPayRoute(guest, handler)

	for _, path := range []string{"/orders", "/orders/create-and-pay", "/guest/orders", "/guest/orders/create-and-pay"} {
		for _, test := range []struct {
			name       string
			payload    string
			wantStatus int
			wantOrders int
		}{
			{name: "missing", wantStatus: response.CodeBadRequest},
			{name: "invalid", payload: `,"captcha_payload":{"turnstile_token":"invalid"}`, wantStatus: response.CodeBadRequest},
			{name: "valid", payload: `,"captcha_payload":{"turnstile_token":"valid"}`, wantOrders: 1},
		} {
			t.Run(path+"/"+test.name, func(t *testing.T) {
				orders.created = 0
				body := `{"items":[{"product_id":1,"quantity":1}],"email":"guest@example.com","order_password":"example-password"` + test.payload + `}`
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				res := httptest.NewRecorder()
				router.ServeHTTP(res, req)
				var result response.Response
				if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
					t.Fatalf("decode response: %v (%s)", err, res.Body.String())
				}
				if result.StatusCode != test.wantStatus || orders.created != test.wantOrders {
					t.Fatalf("response = %s, created = %d; want status %d and %d orders", res.Body.String(), orders.created, test.wantStatus, test.wantOrders)
				}
			})
		}
	}
}

type captchaTestOrderCreator struct{ created int }

func (s *captchaTestOrderCreator) CreateOrder(orderhttp.CreateOrderInput) (*orderdomain.Order, error) {
	s.created++
	return &orderdomain.Order{OrderNo: "test-order"}, nil
}

func (s *captchaTestOrderCreator) CreateGuestOrder(orderhttp.CreateGuestOrderInput) (*orderdomain.Order, error) {
	s.created++
	return &orderdomain.Order{OrderNo: "test-order"}, nil
}

type orderCaptchaStub struct{}

func (orderCaptchaStub) VerifyGuestCreateOrder(payload captchahttp.CaptchaPayloadRequest, _ string) error {
	switch payload.TurnstileToken {
	case "":
		return captcha.ErrRequired
	case "valid":
		return nil
	default:
		return captcha.ErrInvalid
	}
}
