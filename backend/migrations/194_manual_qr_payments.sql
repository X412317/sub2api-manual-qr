-- Manual QR payment assets and user-submitted payment proofs.
-- This migration is additive so the application can be rolled back without
-- dropping payment evidence or invalidating order snapshots.

CREATE TABLE IF NOT EXISTS manual_payment_qr_assets (
    id BIGSERIAL PRIMARY KEY,
    provider_instance_id BIGINT NOT NULL
        REFERENCES payment_provider_instances(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL,
    amount DECIMAL(20,2),
    storage_key VARCHAR(255) NOT NULL UNIQUE,
    mime_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    qr_payload TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT manual_payment_qr_assets_channel_check
        CHECK (channel IN ('alipay', 'wxpay')),
    CONSTRAINT manual_payment_qr_assets_amount_check
        CHECK (amount IS NULL OR amount IN (10, 20, 50, 100, 200, 500, 1000)),
    CONSTRAINT manual_payment_qr_assets_file_size_check CHECK (file_size > 0),
    CONSTRAINT manual_payment_qr_assets_payload_check CHECK (length(qr_payload) BETWEEN 1 AND 4096)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_manual_qr_assets_active_general
    ON manual_payment_qr_assets(provider_instance_id, channel)
    WHERE amount IS NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_manual_qr_assets_active_tier
    ON manual_payment_qr_assets(provider_instance_id, channel, amount)
    WHERE amount IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_manual_qr_assets_provider
    ON manual_payment_qr_assets(provider_instance_id, channel, deleted_at);

CREATE TABLE IF NOT EXISTS manual_payment_proofs (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES payment_orders(id) ON DELETE RESTRICT,
    submission_no SMALLINT NOT NULL,
    channel VARCHAR(20) NOT NULL,
    transaction_no VARCHAR(128) NOT NULL,
    normalized_transaction_no VARCHAR(128) NOT NULL,
    storage_key VARCHAR(255),
    mime_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'submitted',
    received_amount DECIMAL(20,2),
    reviewer_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    rejection_reason TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    proof_deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT manual_payment_proofs_submission_check
        CHECK (submission_no BETWEEN 1 AND 3),
    CONSTRAINT manual_payment_proofs_channel_check
        CHECK (channel IN ('alipay', 'wxpay')),
    CONSTRAINT manual_payment_proofs_status_check
        CHECK (status IN ('submitted', 'approved', 'rejected')),
    CONSTRAINT manual_payment_proofs_file_size_check CHECK (file_size > 0),
    CONSTRAINT manual_payment_proofs_review_check CHECK (
        (status = 'submitted' AND reviewed_at IS NULL AND reviewer_user_id IS NULL)
        OR (status = 'approved' AND reviewed_at IS NOT NULL AND reviewer_user_id IS NOT NULL AND received_amount IS NOT NULL)
        OR (status = 'rejected' AND reviewed_at IS NOT NULL AND reviewer_user_id IS NOT NULL AND rejection_reason IS NOT NULL)
    ),
    UNIQUE (order_id, submission_no),
    UNIQUE (channel, normalized_transaction_no)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_manual_payment_proofs_one_submitted
    ON manual_payment_proofs(order_id)
    WHERE status = 'submitted';

CREATE INDEX IF NOT EXISTS idx_manual_payment_proofs_review_queue
    ON manual_payment_proofs(status, submitted_at)
    WHERE status = 'submitted';

CREATE INDEX IF NOT EXISTS idx_manual_payment_proofs_order
    ON manual_payment_proofs(order_id, submission_no DESC);

