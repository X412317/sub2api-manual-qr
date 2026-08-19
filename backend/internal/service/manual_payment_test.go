package service

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type manualPaymentOrderServiceStub struct {
	getOrderFn     func(context.Context, int64, int64) (*dbent.PaymentOrder, error)
	getOrderByIDFn func(context.Context, int64) (*dbent.PaymentOrder, error)
	fulfillFn      func(context.Context, int64) error
}

func (s *manualPaymentOrderServiceStub) GetOrder(ctx context.Context, orderID, userID int64) (*dbent.PaymentOrder, error) {
	if s.getOrderFn == nil {
		return nil, errors.New("unexpected GetOrder call")
	}
	return s.getOrderFn(ctx, orderID, userID)
}

func (s *manualPaymentOrderServiceStub) GetOrderByID(ctx context.Context, orderID int64) (*dbent.PaymentOrder, error) {
	if s.getOrderByIDFn == nil {
		return nil, errors.New("unexpected GetOrderByID call")
	}
	return s.getOrderByIDFn(ctx, orderID)
}

func (s *manualPaymentOrderServiceStub) ExecuteBalanceFulfillment(ctx context.Context, orderID int64) error {
	if s.fulfillFn == nil {
		return errors.New("unexpected ExecuteBalanceFulfillment call")
	}
	return s.fulfillFn(ctx, orderID)
}

type timeWindowArgument struct {
	min time.Time
	max time.Time
}

func (a timeWindowArgument) Match(value driver.Value) bool {
	actual, ok := value.(time.Time)
	return ok && !actual.Before(a.min) && !actual.After(a.max)
}

func TestManualPaymentFixedAmountsAreExact(t *testing.T) {
	t.Parallel()

	for _, amount := range []float64{10, 20, 50, 100, 200, 500, 1000} {
		require.True(t, isManualPaymentFixedAmount(amount), "expected %.2f to be accepted", amount)
	}
	for _, amount := range []float64{0, 9.99, 10.001, 10.004, 10.01, 1000.01, math.NaN(), math.Inf(1)} {
		require.False(t, isManualPaymentFixedAmount(amount), "expected %v to be rejected", amount)
	}
}

func TestManualProofAuditActionIsUniquePerResubmission(t *testing.T) {
	t.Parallel()

	require.Equal(t, "MANUAL_PROOF_SUBMITTED", manualProofAuditAction("MANUAL_PROOF_SUBMITTED", 1))
	require.Equal(t, "MANUAL_PROOF_SUBMITTED_2", manualProofAuditAction("MANUAL_PROOF_SUBMITTED", 2))
	require.Equal(t, "MANUAL_PROOF_REJECTED_3", manualProofAuditAction("MANUAL_PROOF_REJECTED", 3))
}

func TestValidateManualPaymentEconomics(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateManualPaymentEconomics(0, 1))
	require.Equal(t, "MANUAL_PAYMENT_FEE_MUST_BE_ZERO", infraerrors.Reason(validateManualPaymentEconomics(0.01, 1)))
	require.Equal(t, "MANUAL_PAYMENT_MULTIPLIER_MUST_BE_ONE", infraerrors.Reason(validateManualPaymentEconomics(0, 0.99)))
	require.Equal(t, "MANUAL_PAYMENT_FEE_MUST_BE_ZERO", infraerrors.Reason(validateManualPaymentEconomics(math.NaN(), 1)))
}

func TestValidateManualQRPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		channel string
		payload string
		wantErr string
	}{
		{name: "alipay https", channel: payment.TypeAlipay, payload: "https://qr.alipay.com/example"},
		{name: "alipay scheme", channel: payment.TypeAlipay, payload: "alipays://platformapi/startapp?saId=10000007"},
		{name: "wechat wxp", channel: payment.TypeWxpay, payload: "wxp://f2f0example"},
		{name: "wechat scheme", channel: payment.TypeWxpay, payload: "weixin://wxpay/bizpayurl?pr=example"},
		{name: "channel mismatch", channel: payment.TypeWxpay, payload: "https://qr.alipay.com/example", wantErr: "MANUAL_PAYMENT_QR_CHANNEL_MISMATCH"},
		{name: "untrusted host", channel: payment.TypeAlipay, payload: "https://example.com/pay", wantErr: "MANUAL_PAYMENT_QR_CHANNEL_MISMATCH"},
		{name: "not a URL", channel: payment.TypeAlipay, payload: "plain-text", wantErr: "MANUAL_PAYMENT_QR_INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateManualQRPayload(tt.channel, tt.payload)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Equal(t, tt.wantErr, infraerrors.Reason(err))
		})
	}
}

func TestManualPaymentStorageValidatesAndReencodesImages(t *testing.T) {
	storage := &manualPaymentStorage{root: t.TempDir()}

	_, _, err := decodeManualPaymentImage(bytes.NewBufferString("\x89PNG\r\nnot-an-image"), manualQRMaxUploadBytes)
	require.Equal(t, "MANUAL_PAYMENT_INVALID_IMAGE", infraerrors.Reason(err))

	alipayPayload := "https://qr.alipay.com/manual-test"
	qrImage := manualTestQRCodePNG(t, alipayPayload)
	storedQR, err := storage.storeQR(bytes.NewReader(qrImage), payment.TypeAlipay)
	require.NoError(t, err)
	require.Equal(t, "image/png", storedQR.MIMEType)
	require.Equal(t, alipayPayload, storedQR.QRPayload)

	_, err = storage.storeQR(bytes.NewReader(qrImage), payment.TypeWxpay)
	require.Equal(t, "MANUAL_PAYMENT_QR_CHANNEL_MISMATCH", infraerrors.Reason(err))

	proofImage := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			proofImage.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 4), B: 90, A: 255})
		}
	}
	var jpegInput bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpegInput, proofImage, &jpeg.Options{Quality: 90}))
	jpegInput.WriteString("PRIVATE_EXIF_MARKER")
	storedProof, err := storage.storeProof(bytes.NewReader(jpegInput.Bytes()))
	require.NoError(t, err)
	proofBytes, err := storage.read(storedProof.StorageKey)
	require.NoError(t, err)
	require.False(t, bytes.Contains(proofBytes, []byte("PRIVATE_EXIF_MARKER")), "re-encoding must strip trailing metadata")
}

func TestResolveOrderAssetReturnsTierOrGenericSnapshot(t *testing.T) {
	tests := []struct {
		name       string
		dbAmount   any
		assetID    int64
		wantAmount bool
	}{
		{name: "tier asset", dbAmount: "100.00", assetID: 11, wantAmount: true},
		{name: "generic fallback", dbAmount: nil, assetID: 12, wantAmount: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			svc := &ManualPaymentService{db: db, storage: &manualPaymentStorage{root: t.TempDir()}}

			mock.ExpectQuery(`SELECT provider_key, supported_types FROM payment_provider_instances`).
				WithArgs(int64(42)).
				WillReturnRows(sqlmock.NewRows([]string{"provider_key", "supported_types"}).AddRow(payment.TypeManualQR, "alipay,wxpay"))
			mock.ExpectQuery(`(?s)FROM manual_payment_qr_assets.*ORDER BY CASE WHEN amount = \$3::numeric THEN 0 ELSE 1 END`).
				WithArgs(int64(42), payment.TypeAlipay, "100.00").
				WillReturnRows(sqlmock.NewRows([]string{"id", "channel", "amount", "storage_key", "mime_type", "sha256", "qr_payload"}).
					AddRow(tt.assetID, payment.TypeAlipay, tt.dbAmount, "qr/asset.png", "image/png", "hash", "https://qr.alipay.com/test"))

			snapshot, err := svc.ResolveOrderAsset(context.Background(), "42", payment.TypeAlipay, 100)
			require.NoError(t, err)
			require.Equal(t, float64(tt.assetID), snapshot["asset_id"])
			_, hasAmount := snapshot["amount"]
			require.Equal(t, tt.wantAmount, hasAmount)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetOrderQRImageUsesImmutableOrderSnapshotAndEnforcesOwnership(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	storage := &manualPaymentStorage{root: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Join(storage.root, "qr"), 0o700))
	original := []byte("immutable-order-qr")
	require.NoError(t, os.WriteFile(filepath.Join(storage.root, "qr", "original.png"), original, 0o600))

	providerKey := payment.TypeManualQR
	order := &dbent.PaymentOrder{
		ID:               123,
		UserID:           7,
		ProviderKey:      &providerKey,
		ProviderSnapshot: manualTestProviderSnapshotMap("qr/original.png"),
	}
	stub := &manualPaymentOrderServiceStub{
		getOrderFn: func(_ context.Context, orderID, userID int64) (*dbent.PaymentOrder, error) {
			require.Equal(t, int64(123), orderID)
			require.Equal(t, int64(7), userID)
			return order, nil
		},
	}
	svc := &ManualPaymentService{db: db, paymentSvc: stub, storage: storage}

	image, err := svc.GetOrderQRImage(context.Background(), 123, 7)
	require.NoError(t, err)
	require.Equal(t, original, image.Data)
	require.Equal(t, "snapshot-hash", image.SHA256)

	stub.getOrderFn = func(context.Context, int64, int64) (*dbent.PaymentOrder, error) {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this order")
	}
	_, err = svc.GetOrderQRImage(context.Background(), 123, 8)
	require.Equal(t, "FORBIDDEN", infraerrors.Reason(err))
}

func TestGetAdminProofImageRequiresManualBalanceOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	storage := &manualPaymentStorage{root: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Join(storage.root, "proof"), 0o700))
	want := []byte("private-proof")
	require.NoError(t, os.WriteFile(filepath.Join(storage.root, "proof", "latest.png"), want, 0o600))

	mock.ExpectQuery(`(?s)FROM manual_payment_proofs AS proof.*JOIN payment_orders AS payment_order.*payment_order.provider_key = \$2.*payment_order.order_type = \$3`).
		WithArgs(int64(123), payment.TypeManualQR, payment.OrderTypeBalance).
		WillReturnRows(sqlmock.NewRows([]string{"storage_key", "mime_type", "sha256"}).
			AddRow("proof/latest.png", "image/png", "proof-hash"))

	svc := &ManualPaymentService{db: db, storage: storage}
	image, err := svc.GetAdminProofImage(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, want, image.Data)
	require.Equal(t, "proof-hash", image.SHA256)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubmitManualProofLimitsAttemptsAndTransactionReuse(t *testing.T) {
	tests := []struct {
		name       string
		attempts   int
		insertErr  error
		wantReason string
	}{
		{name: "three attempts", attempts: 3, wantReason: "MANUAL_PAYMENT_SUBMISSION_LIMIT"},
		{name: "duplicate transaction", attempts: 0, insertErr: &pq.Error{Code: "23505", Constraint: "manual_payment_proofs_channel_normalized_transaction_no_key"}, wantReason: "MANUAL_PAYMENT_DUPLICATE_TRANSACTION_NO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			svc := &ManualPaymentService{db: db, storage: &manualPaymentStorage{root: t.TempDir()}}

			expectManualProofOrderLock(mock, 123, 7)
			mock.ExpectQuery(`(?s)SELECT COUNT\(\*\), COALESCE\(bool_or`).
				WithArgs(int64(123)).
				WillReturnRows(sqlmock.NewRows([]string{"count", "submitted"}).AddRow(tt.attempts, false))
			if tt.insertErr != nil {
				mock.ExpectQuery(`(?s)INSERT INTO manual_payment_proofs`).WillReturnError(tt.insertErr)
			}
			mock.ExpectRollback()

			_, err = svc.SubmitProof(context.Background(), 123, 7, "TXN123456", bytes.NewReader(manualTestProofPNG(t)))
			require.Equal(t, tt.wantReason, infraerrors.Reason(err))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubmitManualProofAllowsRejectedResubmission(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc := &ManualPaymentService{db: db, storage: &manualPaymentStorage{root: t.TempDir()}}
	submittedAt := time.Now().UTC()

	expectManualProofOrderLock(mock, 123, 7)
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\), COALESCE\(bool_or`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "submitted"}).AddRow(1, false))
	mock.ExpectQuery(`(?s)INSERT INTO manual_payment_proofs`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_id", "submission_no", "channel", "transaction_no", "status",
			"storage_key", "mime_type", "file_size", "sha256", "submitted_at",
		}).AddRow(8, 123, 2, payment.TypeAlipay, "TXN654321", ManualProofStatusSubmitted,
			"proof/resubmitted.png", "image/png", 128, "hash", submittedAt))
	mock.ExpectExec(`(?s)INSERT INTO payment_audit_logs`).
		WithArgs("123", "MANUAL_PROOF_SUBMITTED_2", sqlmock.AnyArg(), "user:7").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	proof, err := svc.SubmitProof(context.Background(), 123, 7, "TXN654321", bytes.NewReader(manualTestProofPNG(t)))
	require.NoError(t, err)
	require.Equal(t, 2, proof.SubmissionNo)
	require.Equal(t, 1, proof.AttemptsRemaining)
	require.False(t, proof.CanSubmit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewManualProofRejectsRoundedAmountMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc := &ManualPaymentService{
		db: db, paymentSvc: &manualPaymentOrderServiceStub{},
		storage: &manualPaymentStorage{root: t.TempDir()},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, order_type, provider_key, pay_amount.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "order_type", "provider_key", "pay_amount"}).
			AddRow(OrderStatusPending, payment.OrderTypeBalance, payment.TypeManualQR, "10.00"))
	mock.ExpectQuery(`(?s)SELECT id, submission_no, status, transaction_no FROM manual_payment_proofs.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_no", "status", "transaction_no"}).
			AddRow(5, 1, ManualProofStatusSubmitted, "TXN123456"))
	mock.ExpectRollback()

	_, err = svc.ReviewProof(context.Background(), 123, ManualProofReviewRequest{
		Action: "approve", ReceivedAmount: decimal.RequireFromString("10.001"), ReviewerUserID: 9,
	})
	require.Equal(t, "MANUAL_PAYMENT_AMOUNT_MISMATCH", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewManualProofRejectResetsFullTimeout(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	providerKey := payment.TypeManualQR
	resetExpiry := time.Now().Add(time.Duration(defaultOrderTimeoutMin) * time.Minute)
	stub := &manualPaymentOrderServiceStub{
		getOrderByIDFn: func(context.Context, int64) (*dbent.PaymentOrder, error) {
			return &dbent.PaymentOrder{
				ID:               123,
				Status:           OrderStatusPending,
				ExpiresAt:        resetExpiry,
				ProviderKey:      &providerKey,
				ProviderSnapshot: manualTestProviderSnapshotMap("qr/original.png"),
			}, nil
		},
	}
	svc := &ManualPaymentService{db: db, paymentSvc: stub, storage: &manualPaymentStorage{root: t.TempDir()}}
	submittedAt := time.Now().Add(-time.Minute).UTC()
	reviewedAt := time.Now().UTC()
	deadline := time.Now().Add(time.Duration(defaultOrderTimeoutMin) * time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, order_type, provider_key, pay_amount.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "order_type", "provider_key", "pay_amount"}).
			AddRow(OrderStatusPending, payment.OrderTypeBalance, payment.TypeManualQR, "10.00"))
	mock.ExpectQuery(`(?s)SELECT id, submission_no, status, transaction_no FROM manual_payment_proofs.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_no", "status", "transaction_no"}).
			AddRow(5, 1, ManualProofStatusSubmitted, "TXN123456"))
	mock.ExpectExec(`(?s)UPDATE manual_payment_proofs.*SET status = 'rejected'`).
		WithArgs(int64(5), nil, int64(9), "payment unclear", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE payment_orders SET expires_at`).
		WithArgs(int64(123), timeWindowArgument{min: deadline.Add(-time.Second), max: deadline.Add(2 * time.Second)}, sqlmock.AnyArg(), OrderStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO payment_audit_logs`).
		WithArgs("123", "MANUAL_PROOF_REJECTED", sqlmock.AnyArg(), "admin:9").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`(?s)SELECT id, submission_no, channel, transaction_no, status, storage_key`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "submission_no", "channel", "transaction_no", "status", "storage_key", "mime_type",
			"file_size", "sha256", "received_amount", "reviewer_user_id", "rejection_reason",
			"submitted_at", "reviewed_at", "proof_deleted_at",
		}).AddRow(5, 1, payment.TypeAlipay, "TXN123456", ManualProofStatusRejected, "proof/test.png", "image/png",
			128, "proof-hash", nil, 9, "payment unclear", submittedAt, reviewedAt, nil))

	proof, err := svc.ReviewProof(context.Background(), 123, ManualProofReviewRequest{
		Action: "reject", Reason: "payment unclear", ReviewerUserID: 9,
	})
	require.NoError(t, err)
	require.Equal(t, ManualProofStatusRejected, proof.Status)
	require.True(t, proof.CanSubmit)
	require.Equal(t, 2, proof.AttemptsRemaining)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewManualProofConcurrentApprovalChangesNothing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	fulfillmentCalls := 0
	stub := &manualPaymentOrderServiceStub{fulfillFn: func(context.Context, int64) error {
		fulfillmentCalls++
		return nil
	}}
	svc := &ManualPaymentService{db: db, paymentSvc: stub, storage: &manualPaymentStorage{root: t.TempDir()}}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, order_type, provider_key, pay_amount.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "order_type", "provider_key", "pay_amount"}).
			AddRow(OrderStatusPending, payment.OrderTypeBalance, payment.TypeManualQR, "10.00"))
	mock.ExpectQuery(`(?s)SELECT id, submission_no, status, transaction_no FROM manual_payment_proofs.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_no", "status", "transaction_no"}).
			AddRow(5, 1, ManualProofStatusSubmitted, "TXN123456"))
	mock.ExpectExec(`(?s)UPDATE manual_payment_proofs.*SET status = 'approved'`).
		WithArgs(int64(5), "10.00", int64(9), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	_, err = svc.ReviewProof(context.Background(), 123, ManualProofReviewRequest{
		Action: "approve", ReceivedAmount: decimal.RequireFromString("10.00"), ReviewerUserID: 9,
	})
	require.Equal(t, "MANUAL_PAYMENT_ALREADY_REVIEWED", infraerrors.Reason(err))
	require.Zero(t, fulfillmentCalls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewManualProofRetriesIdempotentFulfillmentAfterFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	fulfillmentCalls := 0
	stub := &manualPaymentOrderServiceStub{fulfillFn: func(_ context.Context, orderID int64) error {
		require.Equal(t, int64(123), orderID)
		fulfillmentCalls++
		return errors.New("crediting unavailable")
	}}
	svc := &ManualPaymentService{db: db, paymentSvc: stub, storage: &manualPaymentStorage{root: t.TempDir()}}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, order_type, provider_key, pay_amount.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "order_type", "provider_key", "pay_amount"}).
			AddRow(OrderStatusPending, payment.OrderTypeBalance, payment.TypeManualQR, "10.00"))
	mock.ExpectQuery(`(?s)SELECT id, submission_no, status, transaction_no FROM manual_payment_proofs.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_no", "status", "transaction_no"}).
			AddRow(5, 1, ManualProofStatusSubmitted, "TXN123456"))
	mock.ExpectExec(`(?s)UPDATE manual_payment_proofs.*SET status = 'approved'`).
		WithArgs(int64(5), "10.00", int64(9), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE payment_orders.*SET status = \$2, payment_trade_no`).
		WithArgs(int64(123), OrderStatusPaid, "TXN123456", sqlmock.AnyArg(), OrderStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO payment_audit_logs`).
		WithArgs("123", "MANUAL_PROOF_APPROVED", sqlmock.AnyArg(), "admin:9").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	_, err = svc.ReviewProof(context.Background(), 123, ManualProofReviewRequest{
		Action: "approve", ReceivedAmount: decimal.RequireFromString("10.00"), ReviewerUserID: 9,
	})
	require.EqualError(t, err, "crediting unavailable")

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, order_type, provider_key, pay_amount.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "order_type", "provider_key", "pay_amount"}).
			AddRow(OrderStatusFailed, payment.OrderTypeBalance, payment.TypeManualQR, "10.00"))
	mock.ExpectQuery(`(?s)SELECT id, submission_no, status, transaction_no FROM manual_payment_proofs.*FOR UPDATE`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_no", "status", "transaction_no"}).
			AddRow(5, 1, ManualProofStatusApproved, "TXN123456"))
	mock.ExpectCommit()

	_, err = svc.ReviewProof(context.Background(), 123, ManualProofReviewRequest{
		Action: "approve", ReceivedAmount: decimal.RequireFromString("10.00"), ReviewerUserID: 9,
	})
	require.EqualError(t, err, "crediting unavailable")
	require.Equal(t, 2, fulfillmentCalls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestManualSubmittedProofBlocksExpiryAndCancellation(t *testing.T) {
	for _, blockAnyProof := range []bool{false, true} {
		t.Run(map[bool]string{false: "expiry", true: "cancellation"}[blockAnyProof], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			svc := &ManualPaymentService{db: db, storage: &manualPaymentStorage{root: t.TempDir()}}

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT status, provider_key FROM payment_orders.*FOR UPDATE`).
				WithArgs(int64(123)).
				WillReturnRows(sqlmock.NewRows([]string{"status", "provider_key"}).
					AddRow(OrderStatusPending, payment.TypeManualQR))
			mock.ExpectQuery(`SELECT EXISTS .*manual_payment_proofs`).
				WithArgs(int64(123)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			mock.ExpectRollback()

			changed, blocked, err := svc.TryTransitionPendingOrder(context.Background(), 123, OrderStatusExpired, blockAnyProof)
			require.NoError(t, err)
			require.False(t, changed)
			require.True(t, blocked)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCleanupTerminalManualProofFilesIncludesFailedOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	storage := &manualPaymentStorage{root: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Join(storage.root, "proof"), 0o700))
	proofPath := filepath.Join(storage.root, "proof", "expired.png")
	require.NoError(t, os.WriteFile(proofPath, []byte("expired-proof"), 0o600))
	now := time.Date(2026, time.August, 3, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)payment_order.status IN \('COMPLETED', 'FAILED'.*'REFUNDED'\)`).
		WithArgs(now.Add(-manualProofRetention)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "storage_key"}).AddRow(77, "proof/expired.png"))
	mock.ExpectExec(`(?s)UPDATE manual_payment_proofs.*SET storage_key = NULL`).
		WithArgs(int64(77), now, "proof/expired.png").
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &ManualPaymentService{db: db, storage: storage}
	deleted, err := svc.CleanupTerminalProofFiles(context.Background(), now)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	_, err = os.Stat(proofPath)
	require.True(t, os.IsNotExist(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestValidateManualCreateOrderRejectsSubscriptionAndNonTierAmount(t *testing.T) {
	svc := &PaymentService{}
	selection := &payment.InstanceSelection{ProviderKey: payment.TypeManualQR, PaymentMode: "qrcode"}
	cfg := &PaymentConfig{RechargeFeeRate: 0, BalanceRechargeMultiplier: 1}

	err := svc.validateManualCreateOrder(CreateOrderRequest{
		Amount: 10, PaymentType: payment.TypeAlipay, OrderType: payment.OrderTypeSubscription,
	}, cfg, selection)
	require.Equal(t, "MANUAL_PAYMENT_BALANCE_ONLY", infraerrors.Reason(err))

	err = svc.validateManualCreateOrder(CreateOrderRequest{
		Amount: 10.001, PaymentType: payment.TypeAlipay, OrderType: payment.OrderTypeBalance,
	}, cfg, selection)
	require.Equal(t, "MANUAL_PAYMENT_INVALID_TIER", infraerrors.Reason(err))
}

func expectManualProofOrderLock(mock sqlmock.Sqlmock, orderID, userID int64) {
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT user_id, status, order_type, provider_key, expires_at, provider_snapshot.*FOR UPDATE`).
		WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "status", "order_type", "provider_key", "expires_at", "provider_snapshot",
		}).AddRow(userID, OrderStatusPending, payment.OrderTypeBalance, payment.TypeManualQR,
			time.Now().Add(time.Hour), manualTestProviderSnapshot()))
}

func manualTestProviderSnapshot() []byte {
	return []byte(`{"manual_qr":{"asset_id":1,"channel":"alipay","storage_key":"qr/test.png","mime_type":"image/png","sha256":"hash","qr_payload":"https://qr.alipay.com/test"}}`)
}

func manualTestProviderSnapshotMap(storageKey string) map[string]any {
	return map[string]any{
		"manual_qr": map[string]any{
			"asset_id": 1, "channel": payment.TypeAlipay, "storage_key": storageKey,
			"mime_type": "image/png", "sha256": "snapshot-hash", "qr_payload": "https://qr.alipay.com/test",
		},
	}
}

func manualTestProofPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func manualTestQRCodePNG(t *testing.T, payload string) []byte {
	t.Helper()
	matrix, err := qrcode.NewQRCodeWriter().EncodeWithoutHint(payload, gozxing.BarcodeFormat_QR_CODE, 256, 256)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, matrix))
	return buf.Bytes()
}
