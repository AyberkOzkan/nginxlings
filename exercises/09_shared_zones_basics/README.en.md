# 09 – Shared Memory Zones Fundamentals

## Goal

In this exercise you will declare your first **shared memory zone** and use it
for rate limiting. You will see how a zone is created once, with a name and a
size, and then referenced by name wherever it is needed.

---

## Background – shared memory zones

NGINX runs several worker processes. State that all of them must agree on, such
as how many requests a client has made, cannot live in one worker's memory, so
NGINX keeps it in **shared memory zones**. A zone is declared once with a name
and a size and then used by name; rate limiting, connection limiting, upstream
state and caching all work this way.

For rate limiting:

- `limit_req_zone` (in the `http` context) declares the zone. It takes a **key**
  (what to count per, for example per client), `zone=<name>:<size>`, and a
  `rate=` such as `10r/s` or `30r/m`.
- `limit_req` (in `http`, `server` or `location`) applies a zone by name.
  `burst=` lets some excess requests wait in a queue; `nodelay` serves those
  queued requests at once instead of spacing them out.
- A rejected request gets status 503 unless `limit_req_status` says otherwise.

A generic example (different names, key and values from this lab):

    limit_req_zone $server_name zone=per_site:1m rate=50r/s;

    server {
        location /search/ {
            limit_req zone=per_site burst=20 nodelay;
        }
    }

`return` answers in the rewrite phase, before `limit_req` runs in the preaccess
phase, so a location that uses `return` is never limited.

Reference: [ngx_http_limit_req_module](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).

---

## Lab environment

- `Dockerfile` copies your `nginx.conf` into an `nginx:1.28.3-alpine` image;
  `start.sh` runs NGINX in the foreground.
- `nginx.conf` already has a `server` on port 8080 that serves `index.html` from
  `/usr/share/nginx/html`; `location /` is empty.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image, runs
`nginx -t`, reads the effective configuration with `nginx -T` (comment lines are
ignored), and starts the container (host port 19090). It waits for NGINX from
**inside** the container, so the wait does not use up the host client's
allowance, then sends 20 parallel requests from the host and counts the status
codes.

---

## Task

Edit `nginx.conf`:

1. In the `http` block, declare a rate-limit zone named `req_zone` with a size
   of `64k`. Its key must be the client address in binary form,
   `$binary_remote_addr`. Choose a rate low enough that 20 simultaneous
   requests from one client cannot all get through.
2. Apply `req_zone` in `location /`.
3. Keep `location /` serving the file.

---

## Success criteria

- `nginx -t` accepts the configuration.
- The effective configuration declares `req_zone` with a size of `64k` and the
  key `$binary_remote_addr`, and a `limit_req` uses it.
- Of 20 parallel requests, at least one gets HTTP 200 and at least one is
  rejected (429 or 503).

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `No active limit_req_zone yet`: the zone is not declared, or it is still a
  comment.
- `NGINX rejected the configuration: "limit_req_zone" directive is not allowed
  here`: zones are declared in the `http` block, not inside `server` or
  `location`.
- `must create a zone named req_zone with a size of 64k`: look at the
  `zone=<name>:<size>` parameter.
- `The key of req_zone must be ...`: the key is the first parameter of
  `limit_req_zone`.
- `Nothing uses req_zone yet`: declaring a zone does not limit anything until
  a `limit_req` refers to it.
- `All 20 parallel requests got through`: the rate or the burst is too generous,
  or `location /` answers with `return`.
- `requests got no answer within 2s instead of being rejected`: excess requests
  are queued by `burst` and released slowly. Should they wait, or be rejected?

---

## Review questions

- Why must the request counters live in shared memory instead of in each worker
  process?
- Which status code does a rejected request get by default, and how would you
  make it 429?
- What is the difference between `burst=5` with and without `nodelay`?
- Why is `$binary_remote_addr` a better key than `$remote_addr` for the same
  purpose?
