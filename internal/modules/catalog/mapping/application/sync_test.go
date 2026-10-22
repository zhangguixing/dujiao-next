package application

import (
	"errors"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	settingsintegration "github.com/dujiao-next/internal/modules/settings/schema/integration"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

// fakeSettingsProvider 以固定同步间隔实现 SettingsProvider。
type fakeSettingsProvider struct {
	interval time.Duration
}

type quantityLimitSKUs struct {
	mappingcontract.SKUMappingRepository
	stock int
}

func (r quantityLimitSKUs) GetByLocalSKUID(uint) (*mappingdomain.SKUMapping, error) {
	return &mappingdomain.SKUMapping{ProductMappingID: 1, UpstreamStock: r.stock}, nil
}

type quantityLimitMappings struct {
	mappingcontract.MappingRepository
}

func (quantityLimitMappings) GetByID(uint) (*mappingdomain.Mapping, error) {
	return &mappingdomain.Mapping{ConnectionID: 1}, nil
}

type quantityLimitConnections struct {
	mappingcontract.ConnectionProvider
	protocol string
}

func (r quantityLimitConnections) GetByID(uint) (*siteconnectiondomain.Connection, error) {
	return &siteconnectiondomain.Connection{Protocol: r.protocol}, nil
}

type stockCheckDisabledSettings struct{ fakeSettingsProvider }

func (stockCheckDisabledSettings) GetUpstreamSyncConfig(string) (settingsintegration.UpstreamSyncConfig, error) {
	return settingsintegration.UpstreamSyncConfig{PreOrderStockCheckEnabled: false}, nil
}

func TestCardNetQuantityLimitRunsBeforeOptionalStockChecks(t *testing.T) {
	for _, stock := range []int{-1, 1000} {
		for _, disabled := range []bool{true, false} {
			service := &Service{skuMappings: quantityLimitSKUs{stock: stock}, mappings: quantityLimitMappings{}, connections: quantityLimitConnections{protocol: constants.ConnectionProtocolCardNetV2}}
			if disabled {
				service.settings = stockCheckDisabledSettings{}
			}
			if err := service.EnsureUpstreamStockForOrder(1, 101); !errors.Is(err, productdomain.ErrMaxPurchaseExceeded) {
				t.Fatalf("stock=%d disabled=%v err=%v", stock, disabled, err)
			}
			if err := service.EnsureUpstreamStockForOrder(1, 100); err != nil {
				t.Fatalf("quantity=100 err=%v", err)
			}
			service.connections = quantityLimitConnections{protocol: constants.ConnectionProtocolDujiaoNext}
			if err := service.EnsureUpstreamStockForOrder(1, 101); err != nil {
				t.Fatalf("Dujiao-Next limit changed: %v", err)
			}
		}
	}
}

func (f fakeSettingsProvider) GetUpstreamSyncConfig(fallbackInterval string) (settingsintegration.UpstreamSyncConfig, error) {
	cfg := settingsintegration.DefaultUpstreamSyncConfig()
	cfg.IntervalMinutes = int(f.interval / time.Minute)
	return cfg, nil
}

func (f fakeSettingsProvider) GetUpstreamSyncInterval(fallbackInterval string) (time.Duration, error) {
	return f.interval, nil
}

func TestComputeFullSyncIntervalFloorsAt24h(t *testing.T) {
	svc := &Service{settings: fakeSettingsProvider{interval: 5 * time.Minute}}

	// 5m 同步间隔 × 3 = 15m < 24h，期望落到 24h floor
	got := svc.computeFullSyncInterval()
	if got != fullSyncIntervalFloor {
		t.Fatalf("expected floor=24h, got %v", got)
	}
}

func TestComputeFullSyncIntervalScalesWithLongInterval(t *testing.T) {
	svc := &Service{settings: fakeSettingsProvider{interval: 12 * time.Hour}}

	// 12h * 3 = 36h，应使用 scaled 值
	got := svc.computeFullSyncInterval()
	want := 36 * time.Hour
	if got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestComputeFullSyncIntervalWithoutSettings(t *testing.T) {
	svc := &Service{}
	got := svc.computeFullSyncInterval()
	if got != fullSyncIntervalFloor {
		t.Fatalf("expected floor when settings=nil, got %v", got)
	}
}
