package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration194AddsManualPaymentEvidenceConstraints(t *testing.T) {
	content, err := FS.ReadFile("194_manual_qr_payments.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS manual_payment_qr_assets")
	require.Contains(t, sql, "amount IN (10, 20, 50, 100, 200, 500, 1000)")
	require.Contains(t, sql, "idx_manual_qr_assets_active_general")
	require.Contains(t, sql, "idx_manual_qr_assets_active_tier")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS manual_payment_proofs")
	require.Contains(t, sql, "submission_no BETWEEN 1 AND 3")
	require.Contains(t, sql, "UNIQUE (channel, normalized_transaction_no)")
	require.Contains(t, sql, "idx_manual_payment_proofs_one_submitted")
	require.Contains(t, sql, "REFERENCES payment_orders(id) ON DELETE RESTRICT")
	require.False(t, strings.Contains(strings.ToUpper(sql), "DROP TABLE"), "rollback-compatible migration must remain additive")
}
