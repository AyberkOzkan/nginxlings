# 16 – Shared Memory Zone Sizing & Limits

## Goal

In this exercise you will see that the `<size>` in `zone=<name>:<size>` is a
real limit, and what NGINX does when a `limit_req` zone runs out of room.

Focus:

- Every distinct key (`$binary_remote_addr$arg_id` here) needs one state in the zone
- When the zone is full, NGINX evicts the least recently used state instead of
  failing, so a small zone quietly "forgets" clients it should still be limiting
- A bigger zone keeps remembering them under the same load

## Task

In `nginx.conf`, in the `limit_req_zone` line:

- Keep the zone name as `req_zone`
- Keep `rate=1r/m`
- Increase the zone size (e.g., 256k, 512k, 1m)

The check sends one request as `?id=limited`, then a second one that must be
rejected with 429. Then it sends 1500 requests with other distinct keys and asks
again as `?id=limited`. With the starting 64k zone the limited client has been
evicted by then and gets 200 again; with a big enough zone it still gets 429.

The location serves a file instead of using `return`: `return` runs in the
rewrite phase, before `limit_req` gets a chance in the preaccess phase, so it
would bypass the limit.

Sizing tip: the NGINX documentation says a 1 MB zone keeps about 16 thousand
64-byte states. On 64-bit systems a `limit_req` state needs a 128-byte slot, so
plan for about 8 thousand states per megabyte and leave headroom.

## Success criteria

- Configuration is valid
- The limited client gets 429 on its second request
- It still gets 429 after 1500 other distinct keys
- Zone size is at least 256k
