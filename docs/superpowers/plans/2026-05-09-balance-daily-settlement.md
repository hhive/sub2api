# Balance Daily Settlement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change balance credit depletion from realtime request-path updates to an end-of-day settlement that first applies that day's balance usage, then expires remaining due credit.

**Architecture:** Keep `user_balance_credits` as the only balance batch ledger. Store per-batch `settled_until_date` so the daily worker can skip already-settled batches without adding a separate settlement table. Realtime billing still deducts `users.balance`; only batch depletion moves to the daily worker.

**Tech Stack:** Go, Ent, PostgreSQL migrations, Vue admin settings, sqlmock repository tests.

---

## Scope

- `balance_credit_validity_days = 0` means newly created balance credits never expire and store `expires_at = NULL`.
- API usage consumption is summarized from `usage_logs` where `billing_type = 0`, `actual_cost > 0`, and `created_at` is inside the settlement day.
- Daily settlement order for day `D`: deduct day `D` balance usage from active credits FIFO, mark touched credits as settled through `D`, then expire remaining credits whose `expires_at <= end(D)`.
- Realtime request path must not update `user_balance_credits`; it only updates `users.balance`.
- No independent settlement table is added in this version.

## Files

- Modify: `backend/migrations/136_add_user_balance_credits.sql`
- Modify: `backend/ent/schema/user_balance_credit.go`
- Modify: generated Ent files via `go generate ./ent`
- Modify: `backend/internal/service/balance_credit.go`
- Modify: `backend/internal/repository/balance_credit_repo.go`
- Modify: `backend/internal/repository/balance_credit_repo_test.go`
- Modify: `backend/internal/service/balance_expiry_service.go`
- Modify: `backend/internal/repository/user_repo.go`
- Modify: `backend/internal/repository/usage_billing_repo.go`
- Modify: settings backend/frontend files that currently expose `balance_credit_expiry_interval_seconds`

## Tasks

- [x] **Task 1: Update schema and migration**
  - Add `settled_until_date DATE` to `user_balance_credits`.
  - Replace `balance_credit_expiry_interval_seconds` with daily-settlement wording while keeping the key if needed for compatibility.
  - Add indexes for daily settlement scans: `(user_id, status, settled_until_date, expires_at, id)` and active expiry scans.

- [x] **Task 2: Remove realtime batch depletion from request path**
  - Delete `deductBalanceCreditsWithClient` from `user_repo.go`.
  - Remove `deductUsageBillingBalanceCredits` from `usage_billing_repo.go`.
  - Keep existing realtime `users.balance` deductions unchanged.

- [x] **Task 3: Add daily settlement repository operations**
  - Add usage aggregation by settlement day.
  - Add FIFO credit deduction that only runs from the worker and advances `settled_until_date`.
  - Add due-credit expiry bounded by settlement day end.

- [x] **Task 4: Convert worker to daily settlement**
  - Determine previous complete local day.
  - For that day, first settle usage, then expire due remaining credits.
  - Keep transaction boundaries so user balance expiry, credit expiry, and `balance_expired` history are consistent.

- [x] **Task 5: Update settings UI and DTOs**
  - Keep `balance_credit_validity_days`, with `0` documented as no expiry.
  - Rename UI label from scan interval to daily settlement interval/hour semantics without changing unrelated settings.

- [x] **Task 6: Tests**
  - Repository tests cover `settled_until_date`, daily usage aggregation, FIFO daily deduction, and due expiry.
  - Service tests cover "settle consumption before expiry" and idempotent rerun behavior.
  - Verify request billing no longer touches `user_balance_credits`.

- [x] **Task 7: Verification**
  - Run `go generate ./ent` if Ent schema changed.
  - Run `go test ./...`.
  - Run frontend typecheck.
