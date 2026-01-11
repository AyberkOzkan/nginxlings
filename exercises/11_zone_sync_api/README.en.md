# 11 – Zone Synchronization & API (zone_sync)

## Goal

Expose a Zone Sync status endpoint that behaves like a small JSON API.

You will configure NGINX so that the following path returns a JSON payload:

    /api/8/stream/zone_sync

The response must be:

- HTTP 200
- `Content-Type: application/json`
- Valid JSON (must be parseable by `jq`)
- The JSON must include at least these fields:

    .zones.sync_zone.state
    .zones.sync_zone.last_sync
    .zones.sync_zone.peers

The `peers` field must be an array with exactly 2 items.

---

## Task

Edit `nginx.conf` for this exercise and make NGINX return the provided JSON data
when the request path is exactly:

    /api/8/stream/zone_sync

Requirements:

1. The path must be an exact match (not a prefix match).
2. The response must be served as JSON (`application/json`).
3. The response body must be valid JSON and must include:
   - `zones.sync_zone.state`
   - `zones.sync_zone.last_sync`
   - `zones.sync_zone.peers` (2 items)

---

## Success criteria

This exercise is successful when the verification for this exercise passes and:

- The API path returns HTTP 200.
- The response has `Content-Type: application/json`.
- The JSON is valid and contains the required fields.
- Both peers are present (exactly 2).

---

## Tips

- Use an exact-match location:

    location = /api/8/stream/zone_sync { ... }

- Use a simple static JSON response (a file or a direct return) and ensure the
  correct Content-Type.
- If the check fails, inspect the failure output and adjust the config until
  the endpoint matches the requirements.
