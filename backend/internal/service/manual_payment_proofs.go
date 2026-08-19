package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

func (s *ManualPaymentService) SubmitProof(ctx context.Context, orderID, userID int64, transactionNo string, reader io.Reader) (*ManualPaymentProof, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	displayTransactionNo, normalizedTransactionNo, err := normalizeManualTransactionNo(transactionNo)
	if err != nil {
		return nil, err
	}
	stored, err := s.storage.storeProof(reader)
	if err != nil {
		return nil, err
	}
	keepFile := false
	defer func() {
		if !keepFile {
			_ = s.storage.remove(stored.StorageKey)
		}
	}()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin manual proof transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var orderUserID int64
	var orderStatus, orderType string
	var providerKey sql.NullString
	var expiresAt time.Time
	var snapshotJSON []byte
	err = tx.QueryRowContext(ctx, `
		SELECT user_id, status, order_type, provider_key, expires_at, provider_snapshot
		FROM payment_orders WHERE id = $1 FOR UPDATE`, orderID,
	).Scan(&orderUserID, &orderStatus, &orderType, &providerKey, &expiresAt, &snapshotJSON)
	if err == sql.ErrNoRows {
		return nil, infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("lock manual payment order: %w", err)
	}
	if orderUserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this order")
	}
	if !providerKey.Valid || providerKey.String != payment.TypeManualQR || orderType != payment.OrderTypeBalance {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_ORDER", "order is not a manual balance recharge")
	}
	if orderStatus != OrderStatusPending {
		return nil, infraerrors.BadRequest("INVALID_STATUS", "order cannot accept payment proof in its current status")
	}
	if !expiresAt.After(time.Now()) {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_ORDER_EXPIRED", "order has expired")
	}
	snapshot, err := manualSnapshotFromJSON(snapshotJSON)
	if err != nil {
		return nil, err
	}

	var attempts int
	var submittedExists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(bool_or(status = 'submitted'), FALSE)
		FROM manual_payment_proofs WHERE order_id = $1`, orderID,
	).Scan(&attempts, &submittedExists); err != nil {
		return nil, fmt.Errorf("check manual proof attempts: %w", err)
	}
	if submittedExists {
		return nil, infraerrors.Conflict("MANUAL_PAYMENT_ALREADY_SUBMITTED", "payment proof is already awaiting review")
	}
	if attempts >= manualProofMaxSubmissions {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_SUBMISSION_LIMIT", "payment proof may only be submitted three times")
	}

	proof := &ManualPaymentProof{}
	var storageKey string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO manual_payment_proofs
			(order_id, submission_no, channel, transaction_no, normalized_transaction_no,
			 storage_key, mime_type, file_size, sha256)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, order_id, submission_no, channel, transaction_no, status,
		          storage_key, mime_type, file_size, sha256, submitted_at`,
		orderID, attempts+1, snapshot.Channel, displayTransactionNo, normalizedTransactionNo,
		stored.StorageKey, stored.MIMEType, stored.Size, stored.SHA256,
	).Scan(
		&proof.ID, &proof.OrderID, &proof.SubmissionNo, &proof.Channel, &proof.TransactionNo,
		&proof.Status, &storageKey, &proof.MIMEType, &proof.FileSize, &proof.SHA256, &proof.SubmittedAt,
	)
	if err != nil {
		if isManualTransactionNoConflict(err) {
			return nil, infraerrors.Conflict("MANUAL_PAYMENT_DUPLICATE_TRANSACTION_NO", "this transaction number has already been submitted for the selected channel")
		}
		return nil, fmt.Errorf("insert manual payment proof: %w", err)
	}
	proof.HasImage = storageKey != ""
	proof.AttemptsUsed = attempts + 1
	proof.AttemptsRemaining = manualProofMaxSubmissions - proof.AttemptsUsed
	proof.CanSubmit = false
	proof.QRCodeURL = fmt.Sprintf("/api/v1/payment/orders/%d/manual-qr", orderID)

	if err := insertManualPaymentAudit(ctx, tx, orderID, manualProofAuditAction("MANUAL_PROOF_SUBMITTED", proof.SubmissionNo), fmt.Sprintf("user:%d", userID), map[string]any{
		"submission_no":  proof.SubmissionNo,
		"channel":        proof.Channel,
		"transaction_no": proof.TransactionNo,
		"sha256":         proof.SHA256,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit manual payment proof: %w", err)
	}
	keepFile = true
	return proof, nil
}

func isManualTransactionNoConflict(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == "23505" &&
		strings.Contains(pqErr.Constraint, "normalized_transaction_no")
}

func (s *ManualPaymentService) GetUserProof(ctx context.Context, orderID, userID int64) (*ManualPaymentProof, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if s.paymentSvc == nil {
		return nil, infraerrors.ServiceUnavailable("MANUAL_PAYMENT_UNAVAILABLE", "manual payment service is unavailable")
	}
	order, err := s.paymentSvc.GetOrder(ctx, orderID, userID)
	if err != nil {
		return nil, err
	}
	if order.ProviderKey == nil || *order.ProviderKey != payment.TypeManualQR {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_ORDER", "order is not a manual payment order")
	}
	return s.getProofStatus(ctx, orderID, order.Status, order.ExpiresAt, order.ProviderSnapshot)
}

func (s *ManualPaymentService) GetAdminProof(ctx context.Context, orderID int64) (*ManualPaymentProof, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if s.paymentSvc == nil {
		return nil, infraerrors.ServiceUnavailable("MANUAL_PAYMENT_UNAVAILABLE", "manual payment service is unavailable")
	}
	order, err := s.paymentSvc.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.ProviderKey == nil || *order.ProviderKey != payment.TypeManualQR {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_ORDER", "order is not a manual payment order")
	}
	return s.getProofStatus(ctx, orderID, order.Status, order.ExpiresAt, order.ProviderSnapshot)
}

func (s *ManualPaymentService) getProofStatus(ctx context.Context, orderID int64, orderStatus string, expiresAt time.Time, providerSnapshot map[string]any) (*ManualPaymentProof, error) {
	snapshot, err := manualSnapshotFromOrder(&dbent.PaymentOrder{ProviderSnapshot: providerSnapshot})
	if err != nil {
		return nil, err
	}
	proof := &ManualPaymentProof{
		OrderID:           orderID,
		Channel:           snapshot.Channel,
		Status:            "not_submitted",
		AttemptsRemaining: manualProofMaxSubmissions,
		CanSubmit:         orderStatus == OrderStatusPending && expiresAt.After(time.Now()),
		QRCodeURL:         fmt.Sprintf("/api/v1/payment/orders/%d/manual-qr", orderID),
	}
	if snapshot.Channel == payment.TypeAlipay && validateManualQRPayload(snapshot.Channel, snapshot.QRPayload) == nil {
		proof.AlipayLaunchPayload = snapshot.QRPayload
	}

	var storageKey sql.NullString
	var receivedAmount sql.NullString
	var reviewerUserID sql.NullInt64
	var rejectionReason sql.NullString
	var reviewedAt, proofDeletedAt sql.NullTime
	err = s.db.QueryRowContext(ctx, `
		SELECT id, submission_no, channel, transaction_no, status, storage_key, mime_type,
		       file_size, sha256, received_amount::text, reviewer_user_id, rejection_reason,
		       submitted_at, reviewed_at, proof_deleted_at
		FROM manual_payment_proofs WHERE order_id = $1
		ORDER BY submission_no DESC LIMIT 1`, orderID,
	).Scan(
		&proof.ID, &proof.SubmissionNo, &proof.Channel, &proof.TransactionNo, &proof.Status,
		&storageKey, &proof.MIMEType, &proof.FileSize, &proof.SHA256, &receivedAmount,
		&reviewerUserID, &rejectionReason, &proof.SubmittedAt, &reviewedAt, &proofDeletedAt,
	)
	if err == sql.ErrNoRows {
		return proof, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load manual payment proof: %w", err)
	}
	proof.HasImage = storageKey.Valid && storageKey.String != ""
	proof.AttemptsUsed = proof.SubmissionNo
	proof.AttemptsRemaining = manualProofMaxSubmissions - proof.SubmissionNo
	proof.CanSubmit = proof.Status == ManualProofStatusRejected && proof.AttemptsRemaining > 0 &&
		orderStatus == OrderStatusPending && expiresAt.After(time.Now())
	setManualProofNullableFields(proof, receivedAmount, reviewerUserID, rejectionReason, reviewedAt, proofDeletedAt)
	return proof, nil
}

func setManualProofNullableFields(proof *ManualPaymentProof, amount sql.NullString, reviewer sql.NullInt64, reason sql.NullString, reviewedAt, deletedAt sql.NullTime) {
	if amount.Valid {
		if value, err := decimal.NewFromString(amount.String); err == nil {
			floatValue, _ := value.Float64()
			proof.ReceivedAmount = &floatValue
		}
	}
	if reviewer.Valid {
		proof.ReviewerUserID = &reviewer.Int64
	}
	if reason.Valid {
		proof.RejectionReason = &reason.String
	}
	if reviewedAt.Valid {
		proof.ReviewedAt = &reviewedAt.Time
	}
	if deletedAt.Valid {
		proof.ProofDeletedAt = &deletedAt.Time
	}
}

func (s *ManualPaymentService) GetAdminProofImage(ctx context.Context, orderID int64) (*ManualPrivateImage, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	var storageKey sql.NullString
	var mimeType, hash string
	err := s.db.QueryRowContext(ctx, `
		SELECT proof.storage_key, proof.mime_type, proof.sha256
		FROM manual_payment_proofs AS proof
		JOIN payment_orders AS payment_order ON payment_order.id = proof.order_id
		WHERE proof.order_id = $1 AND payment_order.provider_key = $2
		  AND payment_order.order_type = $3
		ORDER BY proof.submission_no DESC LIMIT 1`, orderID, payment.TypeManualQR, payment.OrderTypeBalance,
	).Scan(&storageKey, &mimeType, &hash)
	if err == sql.ErrNoRows || !storageKey.Valid || storageKey.String == "" {
		return nil, infraerrors.NotFound("MANUAL_PAYMENT_PROOF_IMAGE_NOT_FOUND", "payment proof image is no longer available")
	}
	if err != nil {
		return nil, fmt.Errorf("load manual payment proof image: %w", err)
	}
	data, err := s.storage.read(storageKey.String)
	if err != nil {
		return nil, err
	}
	return &ManualPrivateImage{Data: data, MIMEType: mimeType, SHA256: hash}, nil
}

func (s *ManualPaymentService) ReviewProof(ctx context.Context, orderID int64, req ManualProofReviewRequest) (*ManualPaymentProof, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "approve" && action != "reject" {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_REVIEW", "review action must be approve or reject")
	}
	if req.ReviewerUserID <= 0 {
		return nil, infraerrors.Forbidden("FORBIDDEN", "reviewer identity is required")
	}
	if action == "approve" && s.paymentSvc == nil {
		return nil, infraerrors.ServiceUnavailable("MANUAL_PAYMENT_UNAVAILABLE", "manual payment fulfillment is unavailable")
	}
	reason := strings.TrimSpace(req.Reason)
	if action == "reject" && (len(reason) < 2 || len(reason) > 1000) {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_REJECTION_REASON_REQUIRED", "rejection reason must be 2-1000 characters")
	}
	timeout := s.manualOrderTimeout(ctx)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin manual payment review: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var orderStatus, orderType string
	var providerKey sql.NullString
	var payAmount decimal.Decimal
	err = tx.QueryRowContext(ctx, `
		SELECT status, order_type, provider_key, pay_amount
		FROM payment_orders WHERE id = $1 FOR UPDATE`, orderID,
	).Scan(&orderStatus, &orderType, &providerKey, &payAmount)
	if err == sql.ErrNoRows {
		return nil, infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("lock manual payment order for review: %w", err)
	}
	if !providerKey.Valid || providerKey.String != payment.TypeManualQR || orderType != payment.OrderTypeBalance {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_ORDER", "order is not a manual balance recharge")
	}

	var proofID int64
	var submissionNo int
	var proofStatus, transactionNo string
	err = tx.QueryRowContext(ctx, `
		SELECT id, submission_no, status, transaction_no FROM manual_payment_proofs
		WHERE order_id = $1 ORDER BY submission_no DESC LIMIT 1 FOR UPDATE`, orderID,
	).Scan(&proofID, &submissionNo, &proofStatus, &transactionNo)
	if err == sql.ErrNoRows {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_PROOF_REQUIRED", "payment proof has not been submitted")
	}
	if err != nil {
		return nil, fmt.Errorf("lock manual payment proof: %w", err)
	}

	if action == "approve" && proofStatus == ManualProofStatusApproved {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("finish repeated manual review: %w", err)
		}
		if orderStatus != OrderStatusCompleted {
			if err := s.paymentSvc.ExecuteBalanceFulfillment(ctx, orderID); err != nil {
				return nil, err
			}
		}
		return s.GetAdminProof(ctx, orderID)
	}
	if proofStatus != ManualProofStatusSubmitted || orderStatus != OrderStatusPending {
		return nil, infraerrors.Conflict("MANUAL_PAYMENT_ALREADY_REVIEWED", "payment proof is not awaiting review")
	}

	now := time.Now().UTC()
	operator := fmt.Sprintf("admin:%d", req.ReviewerUserID)
	if action == "approve" {
		received := req.ReceivedAmount
		if received.LessThanOrEqual(decimal.Zero) || !received.Equal(received.Round(2)) || !received.Equal(payAmount) {
			return nil, infraerrors.BadRequest("MANUAL_PAYMENT_AMOUNT_MISMATCH", "received amount must exactly match the order pay amount").
				WithMetadata(map[string]string{"expected": payAmount.StringFixed(2)})
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE manual_payment_proofs
			SET status = 'approved', received_amount = $2, reviewer_user_id = $3,
			    rejection_reason = NULL, reviewed_at = $4, updated_at = $4
			WHERE id = $1 AND status = 'submitted'`, proofID, received.StringFixed(2), req.ReviewerUserID, now)
		if err != nil {
			return nil, fmt.Errorf("approve manual payment proof: %w", err)
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			return nil, infraerrors.Conflict("MANUAL_PAYMENT_ALREADY_REVIEWED", "payment proof was reviewed concurrently")
		}
		result, err = tx.ExecContext(ctx, `
			UPDATE payment_orders
			SET status = $2, payment_trade_no = $3, paid_at = $4,
			    failed_at = NULL, failed_reason = NULL, updated_at = $4
			WHERE id = $1 AND status = $5`, orderID, OrderStatusPaid, transactionNo, now, OrderStatusPending)
		if err != nil {
			return nil, fmt.Errorf("mark manual payment order paid: %w", err)
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			return nil, infraerrors.Conflict("MANUAL_PAYMENT_ALREADY_REVIEWED", "payment order was reviewed concurrently")
		}
		if err := insertManualPaymentAudit(ctx, tx, orderID, "MANUAL_PROOF_APPROVED", operator, map[string]any{
			"proof_id": proofID, "transaction_no": transactionNo, "received_amount": received.StringFixed(2),
		}); err != nil {
			return nil, err
		}
	} else {
		var received any
		if req.ReceivedAmount.GreaterThan(decimal.Zero) {
			received = req.ReceivedAmount.Round(2).StringFixed(2)
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE manual_payment_proofs
			SET status = 'rejected', received_amount = $2, reviewer_user_id = $3,
			    rejection_reason = $4, reviewed_at = $5, updated_at = $5
			WHERE id = $1 AND status = 'submitted'`, proofID, received, req.ReviewerUserID, reason, now)
		if err != nil {
			return nil, fmt.Errorf("reject manual payment proof: %w", err)
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			return nil, infraerrors.Conflict("MANUAL_PAYMENT_ALREADY_REVIEWED", "payment proof was reviewed concurrently")
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE payment_orders SET expires_at = $2, updated_at = $3
			WHERE id = $1 AND status = $4`, orderID, now.Add(timeout), now, OrderStatusPending)
		if err != nil {
			return nil, fmt.Errorf("reset manual payment order timeout: %w", err)
		}
		if err := insertManualPaymentAudit(ctx, tx, orderID, manualProofAuditAction("MANUAL_PROOF_REJECTED", submissionNo), operator, map[string]any{
			"proof_id": proofID, "transaction_no": transactionNo, "reason": reason,
		}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit manual payment review: %w", err)
	}
	if action == "approve" {
		if err := s.paymentSvc.ExecuteBalanceFulfillment(ctx, orderID); err != nil {
			return nil, err
		}
	}
	return s.GetAdminProof(ctx, orderID)
}

func (s *ManualPaymentService) manualOrderTimeout(ctx context.Context) time.Duration {
	minutes := defaultOrderTimeoutMin
	if s != nil && s.configService != nil {
		if cfg, err := s.configService.GetPaymentConfig(ctx); err == nil && cfg.OrderTimeoutMin > 0 {
			minutes = cfg.OrderTimeoutMin
		}
	}
	return time.Duration(minutes) * time.Minute
}

func insertManualPaymentAudit(ctx context.Context, tx *sql.Tx, orderID int64, action, operator string, detail map[string]any) error {
	raw, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("marshal manual payment audit: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment_audit_logs(order_id, action, detail, operator)
		VALUES ($1, $2, $3, $4)`, strconv.FormatInt(orderID, 10), action, string(raw), operator)
	if err != nil {
		return fmt.Errorf("write manual payment audit: %w", err)
	}
	return nil
}

func manualProofAuditAction(base string, submissionNo int) string {
	if submissionNo <= 1 {
		return base
	}
	return fmt.Sprintf("%s_%d", base, submissionNo)
}

func (s *ManualPaymentService) TryTransitionPendingOrder(ctx context.Context, orderID int64, nextStatus string, blockAnyProof bool) (changed, blocked bool, err error) {
	if s == nil || s.db == nil {
		return false, false, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, false, fmt.Errorf("begin manual order transition: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	var providerKey sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT status, provider_key FROM payment_orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&status, &providerKey); err != nil {
		if err == sql.ErrNoRows {
			return false, false, nil
		}
		return false, false, fmt.Errorf("lock manual order transition: %w", err)
	}
	if !providerKey.Valid || providerKey.String != payment.TypeManualQR {
		return false, false, nil
	}
	if status != OrderStatusPending {
		return false, false, nil
	}
	proofPredicate := "status = 'submitted'"
	if blockAnyProof {
		proofPredicate = "TRUE"
	}
	var proofExists bool
	query := `SELECT EXISTS (SELECT 1 FROM manual_payment_proofs WHERE order_id = $1 AND ` + proofPredicate + `)`
	if err := tx.QueryRowContext(ctx, query, orderID).Scan(&proofExists); err != nil {
		return false, false, fmt.Errorf("check manual order proof: %w", err)
	}
	if proofExists {
		return false, true, nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE payment_orders SET status = $2, updated_at = NOW() WHERE id = $1 AND status = $3`, orderID, nextStatus, OrderStatusPending)
	if err != nil {
		return false, false, fmt.Errorf("transition manual payment order: %w", err)
	}
	rows, _ := result.RowsAffected()
	if err := tx.Commit(); err != nil {
		return false, false, fmt.Errorf("commit manual order transition: %w", err)
	}
	return rows == 1, false, nil
}

func (s *ManualPaymentService) PendingReviewSummary(ctx context.Context) (int, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM manual_payment_proofs WHERE status = 'submitted'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending manual payment reviews: %w", err)
	}
	return count, nil
}

func (s *ManualPaymentService) PendingReviewOrderIDs(ctx context.Context) ([]int64, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT order_id FROM manual_payment_proofs WHERE status = 'submitted' ORDER BY submitted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list pending manual payment reviews: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan pending manual payment review: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *ManualPaymentService) CleanupTerminalProofFiles(ctx context.Context, now time.Time) (int, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	cutoff := now.Add(-manualProofRetention)
	rows, err := s.db.QueryContext(ctx, `
		SELECT proof.id, proof.storage_key
		FROM manual_payment_proofs AS proof
		JOIN payment_orders AS payment_order ON payment_order.id = proof.order_id
		WHERE proof.storage_key IS NOT NULL
		  AND payment_order.status IN ('COMPLETED', 'FAILED', 'CANCELLED', 'EXPIRED', 'PARTIALLY_REFUNDED', 'REFUNDED')
		  AND COALESCE(proof.reviewed_at, proof.submitted_at) < $1
		ORDER BY proof.id LIMIT 200`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("query expired manual payment proofs: %w", err)
	}
	type expiredProof struct {
		id  int64
		key string
	}
	items := make([]expiredProof, 0)
	for rows.Next() {
		var item expiredProof
		if err := rows.Scan(&item.id, &item.key); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan expired manual payment proof: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	deleted := 0
	for _, item := range items {
		if err := s.storage.remove(item.key); err != nil {
			continue
		}
		result, err := s.db.ExecContext(ctx, `
			UPDATE manual_payment_proofs
			SET storage_key = NULL, proof_deleted_at = $2, updated_at = $2
			WHERE id = $1 AND storage_key = $3`, item.id, now, item.key)
		if err != nil {
			return deleted, fmt.Errorf("mark manual payment proof deleted: %w", err)
		}
		if changed, _ := result.RowsAffected(); changed == 1 {
			deleted++
		}
	}
	return deleted, nil
}
