package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// ManualQR is an offline provider. It never reports a payment as paid; an
// authenticated administrator must approve the user's proof instead.
type ManualQR struct {
	instanceID string
}

func NewManualQR(instanceID string, config map[string]string) (*ManualQR, error) {
	if mode := strings.TrimSpace(config["paymentMode"]); mode != "" && mode != "qrcode" {
		return nil, fmt.Errorf("manual_qr only supports qrcode payment mode")
	}
	return &ManualQR{instanceID: instanceID}, nil
}

func (m *ManualQR) Name() string        { return "Manual QR" }
func (m *ManualQR) ProviderKey() string { return payment.TypeManualQR }
func (m *ManualQR) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeAlipay, payment.TypeWxpay}
}

func (m *ManualQR) CreatePayment(context.Context, payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	return &payment.CreatePaymentResponse{
		ResultType: payment.CreatePaymentResultOrderCreated,
		Currency:   payment.DefaultPaymentCurrency,
	}, nil
}

func (m *ManualQR) QueryOrder(context.Context, string) (*payment.QueryOrderResponse, error) {
	return &payment.QueryOrderResponse{Status: payment.ProviderStatusPending}, nil
}

func (m *ManualQR) VerifyNotification(context.Context, string, map[string]string) (*payment.PaymentNotification, error) {
	return nil, fmt.Errorf("manual_qr does not accept payment callbacks")
}

func (m *ManualQR) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("manual_qr does not support automatic refunds")
}
