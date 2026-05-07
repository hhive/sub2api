# Usage Record Triage Checklist

## SQL

```sql
-- 1) user
select id, email, role, status, created_at
from users
where email = $1;

-- 2) keys
select id, name, group_id, status, created_at
from api_keys
where user_id = $1
order by id;

-- 3) usage in a date window
select count(*) as usage_count, min(created_at), max(created_at)
from usage_logs
where user_id = $1
  and created_at >= $2
  and created_at < $3;

-- 4) usage by key
select api_key_id, count(*) as usage_count, min(created_at), max(created_at)
from usage_logs
where user_id = $1
group by api_key_id
order by api_key_id;
```

## Log Patterns

- `openai.account_select_failed`
- `billing_check_failed`
- `usage record`
- `queue full`
- `sync_fallback`
- `Dropped`

## Interpretation

- `user` exists, `usage_logs` empty: check the date boundary first.
- Some keys have usage, others do not: the client likely rotated keys.
- Requests exist in logs, but no rows were written: check usage writer overflow or pre-write failures.
