//go:build integration

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestRedeemBalanceValidityMigration(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	_, err := tx.ExecContext(ctx, `
		CREATE TEMP TABLE redeem_codes (code text);
		INSERT INTO redeem_codes (code) VALUES ('legacy');
	`)
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("253_redeem_balance_credit_validity_days.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO redeem_codes (code, balance_credit_validity_days) VALUES ('permanent', 0), ('custom', 14);
		INSERT INTO redeem_codes (code) VALUES ('old-writer');
	`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migration))
	require.NoError(t, err)

	for code, want := range map[string]sql.NullInt64{
		"legacy": {}, "old-writer": {},
		"permanent": {Int64: 0, Valid: true}, "custom": {Int64: 14, Valid: true},
	} {
		var got sql.NullInt64
		require.NoError(t, tx.QueryRowContext(ctx, `SELECT balance_credit_validity_days FROM pg_temp.redeem_codes WHERE code = $1`, code).Scan(&got))
		require.Equal(t, want, got, code)
	}
}

func TestRedeemBalanceValidityRepositoryRoundTrip(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	repo := NewRedeemCodeRepository(tx.Client())
	permanent, custom := 0, 14
	codes := []service.RedeemCode{
		{Code: "validity-nil", Type: service.RedeemTypeBalance, Value: 5, Status: service.StatusUnused},
		{Code: "validity-zero", Type: service.RedeemTypeBalance, Value: 5, Status: service.StatusUnused, BalanceCreditValidityDays: &permanent},
		{Code: "validity-custom", Type: service.RedeemTypeBalance, Value: 5, Status: service.StatusUnused, BalanceCreditValidityDays: &custom},
	}
	require.NoError(t, repo.CreateBatch(ctx, codes))
	for _, code := range codes {
		stored, err := repo.GetByCode(ctx, code.Code)
		require.NoError(t, err)
		require.Equal(t, code.BalanceCreditValidityDays, stored.BalanceCreditValidityDays)
		stored.Notes = "updated metadata"
		stored.BalanceCreditValidityDays = nil
		require.NoError(t, repo.Update(ctx, stored))
		updated, err := repo.GetByID(ctx, stored.ID)
		require.NoError(t, err)
		require.Equal(t, "updated metadata", updated.Notes)
		require.Equal(t, code.BalanceCreditValidityDays, updated.BalanceCreditValidityDays)
		batchNotes := "batch metadata"
		affected, err := repo.BatchUpdate(ctx, []int64{stored.ID}, service.RedeemCodeBatchUpdateFields{Notes: &batchNotes})
		require.NoError(t, err)
		require.EqualValues(t, 1, affected)
		updated, err = repo.GetByID(ctx, stored.ID)
		require.NoError(t, err)
		require.Equal(t, batchNotes, updated.Notes)
		require.Equal(t, code.BalanceCreditValidityDays, updated.BalanceCreditValidityDays)
	}
}

type balanceValiditySettings struct {
	service.SettingRepository
	days string
}

func (s *balanceValiditySettings) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{service.SettingKeyBalanceCreditValidityDays: s.days}, nil
}

type invalidBalanceCreditSource struct {
	service.BalanceCreditRepository
}

func (r invalidBalanceCreditSource) CreateCredit(ctx context.Context, credit service.BalanceCreditCreate) error {
	credit.SourceCode = strings.Repeat("x", 129)
	return r.BalanceCreditRepository.CreateCredit(ctx, credit)
}

type concurrentBalanceCodeUse struct {
	service.RedeemCodeRepository
	ready   chan struct{}
	release chan struct{}
}

func (r concurrentBalanceCodeUse) Use(ctx context.Context, id, userID int64) error {
	r.ready <- struct{}{}
	select {
	case <-r.release:
		return r.RedeemCodeRepository.Use(ctx, id, userID)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestRedeemBalanceValidityCredits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client := testEntClient(t)
	user, err := client.User.Create().SetEmail("balance-validity@example.com").SetPasswordHash("test").Save(ctx)
	require.NoError(t, err)
	codes := NewRedeemCodeRepository(client)
	var codeIDs []int64
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, err := integrationDB.ExecContext(cleanupCtx, `DELETE FROM user_balance_credits WHERE user_id = $1`, user.ID)
		require.NoError(t, err)
		for _, id := range codeIDs {
			require.NoError(t, codes.Delete(cleanupCtx, id))
		}
		_, err = integrationDB.ExecContext(cleanupCtx, `DELETE FROM users WHERE id = $1`, user.ID)
		require.NoError(t, err)
	})
	users := NewUserRepository(client, integrationDB)
	credits := NewBalanceCreditRepository(client, integrationDB)
	settings := &balanceValiditySettings{days: "3"}
	settingService := service.NewSettingService(settings, &config.Config{})
	newRedeem := func(codeRepo service.RedeemCodeRepository, creditRepo service.BalanceCreditRepository) *service.RedeemService {
		return service.NewRedeemService(codeRepo, users, nil, nil, nil, client, nil, nil, creditRepo, settingService)
	}
	redeem := newRedeem(codes, credits)
	permanent, custom := 0, 14
	fixtures := []struct {
		name string
		days *int
		want int
	}{
		{name: "global", want: 9},
		{name: "permanent", days: &permanent},
		{name: "custom", days: &custom, want: custom},
		{name: "retry", days: &custom, want: custom},
		{name: "concurrent", days: &custom, want: custom},
	}
	for _, fixture := range fixtures {
		code := &service.RedeemCode{
			Code: "credit-validity-" + fixture.name, Type: service.RedeemTypeBalance,
			Value: 5, Status: service.StatusUnused, BalanceCreditValidityDays: fixture.days,
		}
		require.NoError(t, codes.Create(ctx, code))
		codeIDs = append(codeIDs, code.ID)
	}
	settings.days = "9"

	for i, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			code := "credit-validity-" + fixture.name
			balance := func() float64 {
				got, err := users.GetByID(ctx, user.ID)
				require.NoError(t, err)
				return got.Balance
			}
			if fixture.name == "retry" {
				_, err := newRedeem(codes, invalidBalanceCreditSource{credits}).RedeemForAdminFulfillment(ctx, user.ID, code)
				require.ErrorContains(t, err, "value too long")
				row, err := codes.GetByCode(ctx, code)
				require.NoError(t, err)
				require.Equal(t, service.StatusUnused, row.Status)
				require.Nil(t, row.UsedBy)
				require.Nil(t, row.UsedAt)
				require.Equal(t, float64(i*5), balance())
				var count int
				require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT count(*) FROM user_balance_credits WHERE user_id = $1 AND source_id = $2`, user.ID, row.ID).Scan(&count))
				require.Zero(t, count)
			}

			start := time.Now()
			if fixture.name == "concurrent" {
				racing := concurrentBalanceCodeUse{codes, make(chan struct{}, 2), make(chan struct{})}
				contender := newRedeem(racing, credits)
				results := make(chan error, 2)
				for range 2 {
					go func() {
						_, err := contender.RedeemForAdminFulfillment(ctx, user.ID, code)
						results <- err
					}()
				}
				for range 2 {
					select {
					case <-racing.ready:
					case <-ctx.Done():
						t.Fatal("both requests did not reach the code update")
					}
				}
				close(racing.release)
				first, second := <-results, <-results
				if first == nil {
					first, second = second, first
				}
				require.ErrorIs(t, first, service.ErrRedeemCodeUsed)
				require.NoError(t, second)
			} else {
				_, err := redeem.RedeemForAdminFulfillment(ctx, user.ID, code)
				require.NoError(t, err)
			}
			end := time.Now()
			_, err := redeem.RedeemForAdminFulfillment(ctx, user.ID, code)
			require.ErrorIs(t, err, service.ErrRedeemCodeUsed)
			require.Equal(t, float64((i+1)*5), balance())

			var expiry sql.NullTime
			var count int
			require.NoError(t, integrationDB.QueryRowContext(ctx, `
				SELECT count(*), min(expires_at) FROM user_balance_credits
				WHERE user_id = $1 AND source_type = $2 AND source_code = $3
			`, user.ID, service.BalanceCreditSourceRedeem, code).Scan(&count, &expiry))
			require.Equal(t, 1, count)
			require.Equal(t, fixture.want != 0, expiry.Valid)
			if expiry.Valid {
				require.WithinRange(t, expiry.Time,
					start.AddDate(0, 0, fixture.want).Truncate(time.Microsecond),
					end.AddDate(0, 0, fixture.want).Add(time.Microsecond))
			}
		})
	}
}
