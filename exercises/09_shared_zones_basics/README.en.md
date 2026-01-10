# 09 – Shared Memory Zones Fundamentals

## Goal

In this exercise you will practice shared memory zones using NGINX rate limiting.

You will:

- Define a shared memory zone with `limit_req_zone`.
- Use that zone with `limit_req` to enforce a request rate limit.
- Confirm that the zone size is defined in the configuration (for example `:64k`).

The automated check will:

- Validate your NGINX configuration syntax.
- Confirm that a `limit_req_zone` with a specific zone name and size exists.
- Confirm that `limit_req` actually uses that zone.
- Send several quick requests and verify that rate limiting is active.

---

## What you edit

You will edit:

    nginx.conf


---

## Requirements

Your `nginx.conf` must include:

1) A shared memory zone definition:

    limit_req_zone $binary_remote_addr zone=req_zone:64k rate=1r/m;

2) A place where that zone is applied:

    limit_req zone=req_zone burst=1 nodelay;

Notes:

- The zone name must be `req_zone`.
- The zone size must be `64k` (this is checked).

---

## Success criteria

This exercise is successful when:

- `nginx -t` succeeds.
- The configuration contains `zone=req_zone:64k`.
- The configuration contains `limit_req zone=req_zone`.
- Under a burst of quick requests:
  - At least one request returns 200,
  - At least one request is rate-limited (HTTP 429 or 503).

---

## Tips

- `limit_req_zone` is typically defined in the `http` context.
- `limit_req` is typically placed in `server` or `location`.
- If rate limiting never triggers, reduce the rate and burst values.
