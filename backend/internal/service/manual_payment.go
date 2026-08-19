package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	ManualProofStatusSubmitted = "submitted"
	ManualProofStatusApproved  = "approved"
	ManualProofStatusRejected  = "rejected"
	manualProofMaxSubmissions  = 3
	manualProofRetention       = 180 * 24 * time.Hour
)

var (
	manualPaymentFixedAmounts = []int64{10, 20, 50, 100, 200, 500, 1000}
	manualTransactionNoRE     = regexp.MustCompile(`^[A-Za-z0-9_-]{6,128}$`)
)

type ManualPaymentService struct {
	db            *sql.DB
	paymentSvc    manualPaymentOrderService
	configService *PaymentConfigService
	storage       *manualPaymentStorage
	storageErr    error
}

type manualPaymentOrderService interface {
	GetOrder(context.Context, int64, int64) (*dbent.PaymentOrder, error)
	GetOrderByID(context.Context, int64) (*dbent.PaymentOrder, error)
	ExecuteBalanceFulfillment(context.Context, int64) error
}

func NewManualPaymentService(db *sql.DB, paymentSvc *PaymentService) *ManualPaymentService {
	storage, err := newManualPaymentStorage()
	service := &ManualPaymentService{db: db, storage: storage, storageErr: err}
	if paymentSvc != nil {
		service.paymentSvc = paymentSvc
		service.configService = paymentSvc.configService
	}
	return service
}

func ManualPaymentFixedAmounts() []int64 {
	return append([]int64(nil), manualPaymentFixedAmounts...)
}

type ManualQRAsset struct {
	ID                 int64      `json:"id"`
	ProviderInstanceID int64      `json:"provider_instance_id"`
	Channel            string     `json:"channel"`
	Amount             *float64   `json:"amount,omitempty"`
	MIMEType           string     `json:"mime_type"`
	FileSize           int64      `json:"file_size"`
	SHA256             string     `json:"sha256"`
	QRPayload          string     `json:"qr_payload,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type manualQRAssetSnapshot struct {
	AssetID    int64    `json:"asset_id"`
	Channel    string   `json:"channel"`
	Amount     *float64 `json:"amount,omitempty"`
	StorageKey string   `json:"storage_key"`
	MIMEType   string   `json:"mime_type"`
	SHA256     string   `json:"sha256"`
	QRPayload  string   `json:"qr_payload"`
}

type ManualPaymentProof struct {
	ID                  int64      `json:"id"`
	OrderID             int64      `json:"order_id"`
	SubmissionNo        int        `json:"submission_no"`
	Channel             string     `json:"channel"`
	TransactionNo       string     `json:"transaction_no"`
	Status              string     `json:"status"`
	MIMEType            string     `json:"mime_type"`
	FileSize            int64      `json:"file_size"`
	SHA256              string     `json:"sha256"`
	HasImage            bool       `json:"has_image"`
	ReceivedAmount      *float64   `json:"received_amount,omitempty"`
	ReviewerUserID      *int64     `json:"reviewer_user_id,omitempty"`
	RejectionReason     *string    `json:"rejection_reason,omitempty"`
	SubmittedAt         time.Time  `json:"submitted_at"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	ProofDeletedAt      *time.Time `json:"proof_deleted_at,omitempty"`
	AttemptsUsed        int        `json:"attempts_used"`
	AttemptsRemaining   int        `json:"attempts_remaining"`
	CanSubmit           bool       `json:"can_submit"`
	QRCodeURL           string     `json:"qr_code_url,omitempty"`
	AlipayLaunchPayload string     `json:"alipay_launch_payload,omitempty"`
}

type ManualProofReviewRequest struct {
	Action         string
	ReceivedAmount decimal.Decimal
	Reason         string
	ReviewerUserID int64
}

type ManualPrivateImage struct {
	Data     []byte
	MIMEType string
	SHA256   string
}

func (s *ManualPaymentService) available() error {
	if s == nil || s.db == nil {
		return infraerrors.ServiceUnavailable("MANUAL_PAYMENT_UNAVAILABLE", "manual payment service is unavailable")
	}
	if s.storageErr != nil || s.storage == nil {
		return infraerrors.ServiceUnavailable("MANUAL_PAYMENT_STORAGE_UNAVAILABLE", "private payment storage is unavailable")
	}
	return nil
}

func normalizeManualChannel(channel string) (string, error) {
	channel = NormalizeVisibleMethod(channel)
	if channel != payment.TypeAlipay && channel != payment.TypeWxpay {
		return "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_CHANNEL", "channel must be alipay or wxpay")
	}
	return channel, nil
}

func manualPaymentAmountDecimal(amount float64) decimal.Decimal {
	return decimal.NewFromFloat(amount).Round(2)
}

func isManualPaymentFixedAmount(amount float64) bool {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return false
	}
	want := decimal.NewFromFloat(amount)
	if !want.Equal(want.Round(2)) {
		return false
	}
	for _, fixed := range manualPaymentFixedAmounts {
		if want.Equal(decimal.NewFromInt(fixed)) {
			return true
		}
	}
	return false
}

func validateManualPaymentEconomics(feeRate, multiplier float64) error {
	if math.IsNaN(feeRate) || math.IsInf(feeRate, 0) || !decimal.NewFromFloat(feeRate).Equal(decimal.Zero) {
		return infraerrors.BadRequest("MANUAL_PAYMENT_FEE_MUST_BE_ZERO", "manual QR payment requires a 0% recharge fee")
	}
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || !decimal.NewFromFloat(multiplier).Equal(decimal.NewFromInt(1)) {
		return infraerrors.BadRequest("MANUAL_PAYMENT_MULTIPLIER_MUST_BE_ONE", "manual QR payment requires a 1:1 balance recharge multiplier")
	}
	return nil
}

func manualPaymentOptionalAmount(amount *float64) (*decimal.Decimal, error) {
	if amount == nil {
		return nil, nil
	}
	if !isManualPaymentFixedAmount(*amount) {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_TIER", "amount is not an allowed manual payment tier")
	}
	value := manualPaymentAmountDecimal(*amount)
	return &value, nil
}

func normalizeManualTransactionNo(raw string) (string, string, error) {
	display := strings.TrimSpace(raw)
	if !manualTransactionNoRE.MatchString(display) {
		return "", "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_TRANSACTION_NO", "transaction number must be 6-128 letters, digits, underscores, or hyphens")
	}
	return display, strings.ToUpper(display), nil
}

func (s *ManualPaymentService) validateProviderChannel(ctx context.Context, providerID int64, channel string) error {
	var providerKey, supportedTypes string
	err := s.db.QueryRowContext(ctx,
		`SELECT provider_key, supported_types FROM payment_provider_instances WHERE id = $1`, providerID,
	).Scan(&providerKey, &supportedTypes)
	if err == sql.ErrNoRows {
		return infraerrors.NotFound("PAYMENT_PROVIDER_NOT_FOUND", "payment provider instance not found")
	}
	if err != nil {
		return fmt.Errorf("load manual payment provider: %w", err)
	}
	if providerKey != payment.TypeManualQR {
		return infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_PROVIDER", "provider instance is not a manual QR provider")
	}
	if !payment.InstanceSupportsType(supportedTypes, channel) {
		return infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_CHANNEL", "channel is not enabled for this provider")
	}
	return nil
}

func (s *ManualPaymentService) ResolveOrderAsset(ctx context.Context, providerInstanceID, channel string, amount float64) (map[string]any, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	providerID, err := strconv.ParseInt(strings.TrimSpace(providerInstanceID), 10, 64)
	if err != nil || providerID <= 0 {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_PROVIDER", "manual payment provider instance is invalid")
	}
	channel, err = normalizeManualChannel(channel)
	if err != nil {
		return nil, err
	}
	if err := s.validateProviderChannel(ctx, providerID, channel); err != nil {
		return nil, err
	}
	amountValue := manualPaymentAmountDecimal(amount)
	var snapshot manualQRAssetSnapshot
	var dbAmount sql.NullString
	err = s.db.QueryRowContext(ctx, `
		SELECT id, channel, amount::text, storage_key, mime_type, sha256, qr_payload
		FROM manual_payment_qr_assets
		WHERE provider_instance_id = $1 AND channel = $2 AND deleted_at IS NULL
		  AND (amount = $3::numeric OR amount IS NULL)
		ORDER BY CASE WHEN amount = $3::numeric THEN 0 ELSE 1 END, id DESC
		LIMIT 1`, providerID, channel, amountValue.StringFixed(2)).Scan(
		&snapshot.AssetID, &snapshot.Channel, &dbAmount, &snapshot.StorageKey,
		&snapshot.MIMEType, &snapshot.SHA256, &snapshot.QRPayload,
	)
	if err == sql.ErrNoRows {
		return nil, infraerrors.ServiceUnavailable("MANUAL_PAYMENT_QR_NOT_CONFIGURED", "no QR code is configured for this amount and channel")
	}
	if err != nil {
		return nil, fmt.Errorf("resolve manual payment QR asset: %w", err)
	}
	if dbAmount.Valid {
		parsed, parseErr := decimal.NewFromString(dbAmount.String)
		if parseErr != nil {
			return nil, fmt.Errorf("parse manual payment QR amount: %w", parseErr)
		}
		value, _ := parsed.Float64()
		snapshot.Amount = &value
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal manual payment QR snapshot: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("build manual payment QR snapshot: %w", err)
	}
	return result, nil
}

func manualSnapshotFromOrder(order *dbent.PaymentOrder) (*manualQRAssetSnapshot, error) {
	if order == nil || order.ProviderSnapshot == nil {
		return nil, infraerrors.NotFound("MANUAL_PAYMENT_QR_NOT_FOUND", "order does not contain a manual payment QR snapshot")
	}
	raw, ok := order.ProviderSnapshot["manual_qr"]
	if !ok || raw == nil {
		return nil, infraerrors.NotFound("MANUAL_PAYMENT_QR_NOT_FOUND", "order does not contain a manual payment QR snapshot")
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal order QR snapshot: %w", err)
	}
	var snapshot manualQRAssetSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode order QR snapshot: %w", err)
	}
	if snapshot.AssetID <= 0 || snapshot.StorageKey == "" || snapshot.Channel == "" {
		return nil, infraerrors.NotFound("MANUAL_PAYMENT_QR_NOT_FOUND", "order manual payment QR snapshot is incomplete")
	}
	return &snapshot, nil
}

func manualSnapshotFromJSON(raw []byte) (*manualQRAssetSnapshot, error) {
	var providerSnapshot map[string]any
	if err := json.Unmarshal(raw, &providerSnapshot); err != nil {
		return nil, fmt.Errorf("decode provider snapshot: %w", err)
	}
	order := &dbent.PaymentOrder{ProviderSnapshot: providerSnapshot}
	return manualSnapshotFromOrder(order)
}
