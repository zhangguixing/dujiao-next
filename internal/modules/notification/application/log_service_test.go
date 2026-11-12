package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/notification/contract"
	"github.com/dujiao-next/internal/modules/notification/domain"
	settingsmessaging "github.com/dujiao-next/internal/modules/settings/schema/messaging"
	settingsstorefront "github.com/dujiao-next/internal/modules/settings/schema/storefront"
	"github.com/dujiao-next/internal/queue"
)

type notificationSettingsStub struct {
	notification settingsmessaging.NotificationCenterSetting
	dashboard    settingsstorefront.DashboardSetting
}

func (s notificationSettingsStub) GetNotificationCenterSetting() (settingsmessaging.NotificationCenterSetting, error) {
	return s.notification, nil
}

func (s notificationSettingsStub) GetDashboardSetting() (settingsstorefront.DashboardSetting, error) {
	return s.dashboard, nil
}

type notificationEmailStub struct{}

func (notificationEmailStub) SendCustomEmail(recipient, _, _ string) error {
	if strings.Contains(recipient, "failure") {
		return errors.New("simulated email failure")
	}
	return nil
}

type notificationFeishuCall struct {
	appID         string
	appSecret     string
	receiveIDType string
	receiveID     string
	message       string
}

type notificationFeishuStub struct {
	calls []notificationFeishuCall
}

func (s *notificationFeishuStub) SendMessage(_ context.Context, appID, appSecret, receiveIDType, receiveID, message string) error {
	s.calls = append(s.calls, notificationFeishuCall{
		appID:         appID,
		appSecret:     appSecret,
		receiveIDType: receiveIDType,
		receiveID:     receiveID,
		message:       message,
	})
	if strings.Contains(receiveID, "failure") {
		return errors.New("simulated feishu failure")
	}
	return nil
}

type notificationLogRepositoryStub struct {
	items []domain.NotificationLog
}

func (r *notificationLogRepositoryStub) Create(item *domain.NotificationLog) error {
	if item != nil {
		copy := *item
		copy.ID = uint(len(r.items) + 1)
		r.items = append(r.items, copy)
	}
	return nil
}

func (r *notificationLogRepositoryStub) ListAdmin(filter contract.LogListFilter) ([]domain.NotificationLog, int64, error) {
	result := make([]domain.NotificationLog, 0, len(r.items))
	for _, item := range r.items {
		if filter.EventType != "" && item.EventType != filter.EventType {
			continue
		}
		if filter.IsTest != nil && item.IsTest != *filter.IsTest {
			continue
		}
		result = append(result, item)
	}
	return result, int64(len(result)), nil
}

func setupLogService(t *testing.T) (*Service, *LogService) {
	t.Helper()
	logService := NewLogService(&notificationLogRepositoryStub{})
	setting := settingsmessaging.NotificationCenterSetting{
		DefaultLocale: constants.LocaleEnUS,
		Scenes: settingsmessaging.NotificationSceneSetting{
			OrderPaidSuccess: true,
			ExceptionAlert:   true,
		},
		Channels: settingsmessaging.NotificationChannelsSetting{
			Email: settingsmessaging.NotificationChannelSetting{
				Enabled:    true,
				Recipients: []string{"success@example.com", "failure@example.com"},
			},
		},
		Templates: settingsmessaging.NotificationTemplatesSetting{
			OrderPaidSuccess: settingsmessaging.NotificationSceneTemplate{
				ENUS: settingsmessaging.NotificationLocalizedTemplate{
					Title: "Order {{order_no}}",
					Body:  "Customer {{customer_email}}",
				},
			},
			ExceptionAlert: settingsmessaging.NotificationSceneTemplate{
				ENUS: settingsmessaging.NotificationLocalizedTemplate{
					Title: "Alert {{message}}",
					Body:  "{{message}}",
				},
			},
		},
	}
	service := NewService(notificationSettingsStub{notification: setting}, notificationEmailStub{}, nil, nil, logService, nil, nil)
	return service, logService
}

func TestServiceSendTestRecordsSuccessLog(t *testing.T) {
	service, logService := setupLogService(t)
	if err := service.SendTest(context.Background(), contract.TestSendInput{
		Channel: "email",
		Target:  "success@example.com",
		Scene:   constants.NotificationEventOrderPaidSuccess,
		Locale:  constants.LocaleEnUS,
	}); err != nil {
		t.Fatalf("SendTest failed: %v", err)
	}

	isTest := true
	items, total, err := logService.ListForAdmin(contract.LogListFilter{
		Page: 1, PageSize: 10, EventType: constants.NotificationEventOrderPaidSuccess, IsTest: &isTest,
	})
	if err != nil {
		t.Fatalf("list notification logs failed: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 test log, total=%d len=%d", total, len(items))
	}
	if items[0].Status != notificationLogStatusSuccess || !items[0].IsTest {
		t.Fatalf("unexpected log: %#v", items[0])
	}
	if !strings.Contains(items[0].Title, "DJ202603230001") {
		t.Fatalf("title should include sample order no, got %s", items[0].Title)
	}
}

func TestServiceDispatchSingleEventRecordsPerRecipientResult(t *testing.T) {
	service, logService := setupLogService(t)
	setting, err := service.settingService.GetNotificationCenterSetting()
	if err != nil {
		t.Fatalf("get notification center setting failed: %v", err)
	}
	dispatchErr := service.dispatchSingleEvent(context.Background(), setting, queue.NotificationDispatchPayload{
		EventType: constants.NotificationEventOrderPaidSuccess,
		BizType:   constants.NotificationBizTypeOrder,
		BizID:     88,
		Locale:    constants.LocaleEnUS,
		Force:     true,
		Data: map[string]interface{}{
			"order_no":       "DJ-LOG-88",
			"customer_email": "member@example.com",
		},
	})
	if !errors.Is(dispatchErr, contract.ErrSendFailed) {
		t.Fatalf("expected send failure, got %v", dispatchErr)
	}

	items, total, err := logService.ListForAdmin(contract.LogListFilter{Page: 1, PageSize: 10, EventType: constants.NotificationEventOrderPaidSuccess})
	if err != nil {
		t.Fatalf("list notification logs failed: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected 2 recipient logs, total=%d len=%d", total, len(items))
	}
	statuses := map[string]string{}
	for _, item := range items {
		statuses[item.Recipient] = item.Status
	}
	if statuses["success@example.com"] != notificationLogStatusSuccess {
		t.Fatalf("success recipient status mismatch: %v", statuses)
	}
	if statuses["failure@example.com"] != notificationLogStatusFailed {
		t.Fatalf("failure recipient status mismatch: %v", statuses)
	}
}

func TestServiceDispatchRendersExceptionAlertDetails(t *testing.T) {
	tests := []struct {
		name    string
		bizType string
		locale  string
		data    map[string]interface{}
		want    string
	}{
		{
			name:    "queued procurement failure",
			bizType: constants.NotificationBizTypeProcurement,
			locale:  constants.LocaleZhCN,
			data:    map[string]interface{}{"procurement_order_id": float64(42), "error": "upstream stock unavailable"},
			want:    "系统异常告警\n\n告警类型：采购异常\n告警级别：error\n当前值：不适用\n阈值：不适用\n详情：upstream stock unavailable",
		},
		{
			name:    "reconciliation mismatch",
			bizType: constants.NotificationBizTypeReconciliation,
			locale:  constants.LocaleEnUS,
			data:    map[string]interface{}{"mismatched_count": float64(3), "message": "Found 3 discrepancies"},
			want:    "System Exception Alert\n\nType: Reconciliation Mismatch\nLevel: warning\nCurrent: 3\nThreshold: N/A\nDetails: Found 3 discrepancies",
		},
		{
			name:    "payment callback preserves supplied details",
			bizType: constants.NotificationBizTypePaymentCallback,
			locale:  constants.LocaleZhTW,
			data:    map[string]interface{}{"alert_type": "callback_unrecognized", "alert_level": "warning", "message": "支付回調格式無法識別"},
			want:    "系統異常告警\n\n告警類型：callback_unrecognized\n告警級別：warning\n當前值：不適用\n閾值：不適用\n詳情：支付回調格式無法識別",
		},
		{
			name:    "dashboard alert preserves actual values",
			bizType: constants.NotificationBizTypeDashboardAlert,
			locale:  constants.LocaleZhCN,
			data: map[string]interface{}{
				"alert_type": "售罄商品", "alert_level": "warning", "alert_value": "2", "alert_threshold": "0",
				"message": "检测到 2 个售罄商品", "affected_items_summary": "1. 示例商品",
			},
			want: "系统异常告警\n\n告警类型：售罄商品\n告警级别：warning\n当前值：2\n阈值：0\n详情：检测到 2 个售罄商品\n1. 示例商品",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting := settingsmessaging.NotificationCenterDefaultSetting()
			setting.Channels.Feishu.Enabled = true
			setting.Channels.Feishu.Recipients = []string{"oc_alerts"}
			sender := &notificationFeishuStub{}
			service := NewService(notificationSettingsStub{notification: setting}, nil, nil, nil, nil, nil, sender)
			if err := service.Dispatch(t.Context(), queue.NotificationDispatchPayload{
				EventType: constants.NotificationEventExceptionAlert,
				BizType:   tt.bizType,
				Locale:    tt.locale,
				Force:     true,
				Data:      tt.data,
			}); err != nil {
				t.Fatalf("Dispatch failed: %v", err)
			}
			if len(sender.calls) != 1 {
				t.Fatalf("expected one notification, got %d", len(sender.calls))
			}
			if got := sender.calls[0].message; got != tt.want {
				t.Errorf("notification = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceDispatchSingleEventSendsFeishuAndRecordsEachRecipient(t *testing.T) {
	logService := NewLogService(&notificationLogRepositoryStub{})
	feishuSender := &notificationFeishuStub{}
	setting := settingsmessaging.NotificationCenterDefaultSetting()
	setting.DefaultLocale = constants.LocaleZhCN
	setting.Channels.Feishu = settingsmessaging.FeishuNotificationChannelSetting{
		Enabled:       true,
		AppID:         "cli_demo",
		AppSecret:     "secret",
		ReceiveIDType: settingsmessaging.FeishuReceiveIDTypeChatID,
		Recipients:    []string{"oc_success", "oc_failure"},
	}
	service := NewService(notificationSettingsStub{notification: setting}, nil, nil, nil, logService, nil, feishuSender)

	err := service.dispatchSingleEvent(context.Background(), setting, queue.NotificationDispatchPayload{
		EventType: constants.NotificationEventOrderPaidSuccess,
		BizType:   constants.NotificationBizTypeOrder,
		BizID:     99,
		Locale:    constants.LocaleZhCN,
		Force:     true,
		Data: map[string]interface{}{
			"order_no":       "DJ-FEISHU-99",
			"customer_email": "member@example.com",
		},
	})
	if !errors.Is(err, contract.ErrSendFailed) {
		t.Fatalf("expected one feishu failure, got %v", err)
	}
	if len(feishuSender.calls) != 2 {
		t.Fatalf("expected two feishu calls, got %d", len(feishuSender.calls))
	}
	firstCall := feishuSender.calls[0]
	if firstCall.appID != "cli_demo" || firstCall.appSecret != "secret" || firstCall.receiveIDType != settingsmessaging.FeishuReceiveIDTypeChatID {
		t.Fatalf("unexpected feishu credentials or id type: %#v", firstCall)
	}
	if !strings.Contains(firstCall.message, "DJ-FEISHU-99") {
		t.Fatalf("rendered feishu message missing order number: %q", firstCall.message)
	}

	items, total, listErr := logService.ListForAdmin(contract.LogListFilter{
		Page: 1, PageSize: 10, Channel: constants.NotificationChannelFeishu,
	})
	if listErr != nil {
		t.Fatalf("list feishu notification logs: %v", listErr)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected two feishu logs, total=%d len=%d", total, len(items))
	}
	statuses := map[string]string{}
	for _, item := range items {
		statuses[item.Recipient] = item.Status
	}
	if statuses["oc_success"] != notificationLogStatusSuccess || statuses["oc_failure"] != notificationLogStatusFailed {
		t.Fatalf("unexpected feishu log statuses: %#v", statuses)
	}
}
