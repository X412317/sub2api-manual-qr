package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

func (s *ManualPaymentService) SaveQRAsset(ctx context.Context, providerID int64, channel string, amount *float64, reader io.Reader) (*ManualQRAsset, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	channel, err := normalizeManualChannel(channel)
	if err != nil {
		return nil, err
	}
	amountValue, err := manualPaymentOptionalAmount(amount)
	if err != nil {
		return nil, err
	}
	if err := s.validateProviderChannel(ctx, providerID, channel); err != nil {
		return nil, err
	}
	stored, err := s.storage.storeQR(reader, channel)
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
		return nil, fmt.Errorf("begin QR asset transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if amountValue == nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE manual_payment_qr_assets SET deleted_at = NOW(), updated_at = NOW()
			WHERE provider_instance_id = $1 AND channel = $2 AND amount IS NULL AND deleted_at IS NULL`,
			providerID, channel)
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE manual_payment_qr_assets SET deleted_at = NOW(), updated_at = NOW()
			WHERE provider_instance_id = $1 AND channel = $2 AND amount = $3::numeric AND deleted_at IS NULL`,
			providerID, channel, amountValue.StringFixed(2))
	}
	if err != nil {
		return nil, fmt.Errorf("replace existing QR asset: %w", err)
	}

	var asset ManualQRAsset
	var dbAmount sql.NullString
	err = tx.QueryRowContext(ctx, `
		INSERT INTO manual_payment_qr_assets
			(provider_instance_id, channel, amount, storage_key, mime_type, file_size, sha256, qr_payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, provider_instance_id, channel, amount::text, mime_type, file_size, sha256, qr_payload, created_at, updated_at`,
		providerID, channel, nullableManualAmount(amountValue), stored.StorageKey, stored.MIMEType,
		stored.Size, stored.SHA256, stored.QRPayload,
	).Scan(
		&asset.ID, &asset.ProviderInstanceID, &asset.Channel, &dbAmount, &asset.MIMEType,
		&asset.FileSize, &asset.SHA256, &asset.QRPayload, &asset.CreatedAt, &asset.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert QR asset: %w", err)
	}
	setManualAssetAmount(&asset, dbAmount)
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit QR asset: %w", err)
	}
	keepFile = true
	return &asset, nil
}

func nullableManualAmount(amount *decimal.Decimal) any {
	if amount == nil {
		return nil
	}
	return amount.StringFixed(2)
}

func setManualAssetAmount(asset *ManualQRAsset, raw sql.NullString) {
	if asset == nil || !raw.Valid {
		return
	}
	value, err := decimal.NewFromString(raw.String)
	if err != nil {
		return
	}
	floatValue, _ := value.Float64()
	asset.Amount = &floatValue
}

func (s *ManualPaymentService) ListQRAssets(ctx context.Context, providerID int64) ([]ManualQRAsset, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	var providerKey string
	if err := s.db.QueryRowContext(ctx, `SELECT provider_key FROM payment_provider_instances WHERE id = $1`, providerID).Scan(&providerKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, infraerrors.NotFound("PAYMENT_PROVIDER_NOT_FOUND", "payment provider instance not found")
		}
		return nil, fmt.Errorf("load payment provider: %w", err)
	}
	if providerKey != payment.TypeManualQR {
		return nil, infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_PROVIDER", "provider instance is not a manual QR provider")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, provider_instance_id, channel, amount::text, mime_type, file_size, sha256, qr_payload, created_at, updated_at
		FROM manual_payment_qr_assets
		WHERE provider_instance_id = $1 AND deleted_at IS NULL
		ORDER BY channel, amount NULLS FIRST`, providerID)
	if err != nil {
		return nil, fmt.Errorf("list manual QR assets: %w", err)
	}
	defer rows.Close()
	assets := make([]ManualQRAsset, 0)
	for rows.Next() {
		var asset ManualQRAsset
		var dbAmount sql.NullString
		if err := rows.Scan(
			&asset.ID, &asset.ProviderInstanceID, &asset.Channel, &dbAmount, &asset.MIMEType,
			&asset.FileSize, &asset.SHA256, &asset.QRPayload, &asset.CreatedAt, &asset.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan manual QR asset: %w", err)
		}
		setManualAssetAmount(&asset, dbAmount)
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manual QR assets: %w", err)
	}
	return assets, nil
}

func (s *ManualPaymentService) DeleteQRAsset(ctx context.Context, providerID, assetID int64) error {
	if err := s.available(); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE manual_payment_qr_assets AS asset
		SET deleted_at = NOW(), updated_at = NOW()
		FROM payment_provider_instances AS provider
		WHERE asset.id = $1 AND asset.provider_instance_id = $2 AND asset.deleted_at IS NULL
		  AND provider.id = asset.provider_instance_id AND provider.provider_key = $3`,
		assetID, providerID, payment.TypeManualQR)
	if err != nil {
		return fmt.Errorf("delete manual QR asset: %w", err)
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return infraerrors.NotFound("MANUAL_PAYMENT_QR_NOT_FOUND", "manual payment QR asset not found")
	}
	// The file is intentionally retained because existing orders reference the
	// immutable storage key in provider_snapshot.
	return nil
}

func (s *ManualPaymentService) GetQRAssetImage(ctx context.Context, providerID, assetID int64) (*ManualPrivateImage, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	var storageKey, mimeType, hash string
	err := s.db.QueryRowContext(ctx, `
		SELECT asset.storage_key, asset.mime_type, asset.sha256
		FROM manual_payment_qr_assets AS asset
		JOIN payment_provider_instances AS provider ON provider.id = asset.provider_instance_id
		WHERE asset.id = $1 AND asset.provider_instance_id = $2 AND asset.deleted_at IS NULL
		  AND provider.provider_key = $3`, assetID, providerID, payment.TypeManualQR,
	).Scan(&storageKey, &mimeType, &hash)
	if err == sql.ErrNoRows {
		return nil, infraerrors.NotFound("MANUAL_PAYMENT_QR_NOT_FOUND", "manual payment QR asset not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load manual QR asset: %w", err)
	}
	data, err := s.storage.read(storageKey)
	if err != nil {
		return nil, err
	}
	return &ManualPrivateImage{Data: data, MIMEType: mimeType, SHA256: hash}, nil
}

func (s *ManualPaymentService) GetOrderQRImage(ctx context.Context, orderID, userID int64) (*ManualPrivateImage, error) {
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
	snapshot, err := manualSnapshotFromOrder(order)
	if err != nil {
		return nil, err
	}
	data, err := s.storage.read(snapshot.StorageKey)
	if err != nil {
		return nil, err
	}
	return &ManualPrivateImage{Data: data, MIMEType: snapshot.MIMEType, SHA256: snapshot.SHA256}, nil
}
