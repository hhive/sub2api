---
name: usage-record-triage
description: Triage missing or empty usage records by checking user ownership, API keys, date boundaries, gateway logs, and usage write-path failures.
---

# Usage Record Triage

## Use When

Use this skill when a user says they are actively using the service, but the usage page is empty, partial, or missing for a date range.

## Workflow

1. Identify the user by email and confirm `user_id`.
2. List all API keys for that user.
3. Check `usage_logs` for the target date in both UTC and the app timezone.
4. Break usage down by `api_key_id`.
5. Compare with gateway logs for the same window.
6. If requests exist but usage is missing, inspect the write path:
   - `usage_record_worker_pool` overflow mode
   - queue full / dropped / sync fallback behavior
   - billing or account-selection failures before usage write
7. Conclude whether the issue is:
   - no traffic
   - traffic on a different API key
   - write-path drops
   - date-boundary mismatch

## Checks That Matter

- Always compare the requested date in UTC and in `Asia/Shanghai`.
- Do not assume "no usage" if one API key is empty; check all keys for the user.
- Treat `openai.account_select_failed` and `billing_check_failed` as signals that requests may have stopped before usage was written.
- If the service uses async usage recording, remember queue overflow can drop records or only sample them.

## Reference

See [references/checklist.md](references/checklist.md) for SQL snippets and log patterns.
