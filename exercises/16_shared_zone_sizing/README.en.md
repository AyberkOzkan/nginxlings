# 16 – Shared Memory Zone Sizing and Limits

## Goal

In this exercise you will see that the `<size>` in `zone=<name>:<size>` is a
real limit, find out what NGINX does when a `limit_req` zone runs out of room,
and size the zone so that it keeps working under many distinct keys.

---

## Background – what fills a zone

- A `limit_req_zone` keeps **one state per distinct key**. With the key
  `$binary_remote_addr$arg_id`, every different `?id=` value from a client is a
  new state. (The `id` part exists only so this lab can create many keys from
  one machine.)
- When the zone is full, NGINX does not fail the request. It **removes the least
  recently used state** and stores the new one. Only if that still does not make
  room is the request rejected with an error. A zone that is too small therefore
  shows no error; it quietly forgets clients it should still be limiting.
- The NGINX documentation gives the sizing rule: "One megabyte zone can keep
  about 16 thousand 64-byte states or about 8 thousand 128-byte states." A
  state takes 64 bytes on 32-bit and 128 bytes on 64-bit platforms. Leave
  headroom: the zone also holds some bookkeeping.
- NGINX refuses a zone that is far too small with `zone "..." is too small`.

The location serves a file instead of using `return`: `return` runs in the
rewrite phase, before `limit_req` gets a chance in the preaccess phase, so it
would bypass the limit.

Reference: [limit_req_zone](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html#limit_req_zone).

---

## Lab environment

- `Dockerfile` builds on `nginx:1.28.3-alpine` and writes `zone-sizing-ok` into
  `/usr/share/nginx/html/index.html`.
- `nginx.conf` declares `req_zone` with `64k` and `rate=1r/m`, answers rejected
  requests with 429, and applies the zone in `location /`, which serves the
  file. The slow rate keeps a limited client limited for a whole minute, long
  enough for the check to see whether the zone still remembers it.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image, reads
the effective configuration with `nginx -T`, and starts the container (host
port 18160). Then it:

1. requests `?id=limited` twice: the first must get 200, the second 429;
2. sends 1500 requests with other distinct `?id=` values over one keep-alive
   connection;
3. requests `?id=limited` again: it must still get 429;
4. confirms the zone is at least `256k`.

---

## Task

In `nginx.conf`, make `req_zone` big enough to remember the limited client
through step 3. Keep the zone name `req_zone`, its key, `rate=1r/m`, the
`limit_req` in `location /`, and the file serving as they are. The size must be
at least `256k`.

---

## Success criteria

- `nginx -t` accepts the configuration.
- The limited client gets 429 on its second request.
- It still gets 429 after 1500 other distinct keys.
- `req_zone` is at least `256k`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `req_zone:64k forgot the limited client`: the zone filled up and NGINX evicted
  the oldest state, the limited client's. How many states does step 2 create,
  and how many fit in your zone?
- `held 1500 keys, but the exercise asks for at least 256k`: your size just
  works, but leaves no headroom. The requirement is 256k or more.
- `NGINX rejected the configuration: zone "zone=req_zone:..." is too small`: the
  size went the wrong way.
- `Expected HTTP 429 for the second request`: `limit_req` did not run. Is it
  still in `location /`, and is the location serving the file rather than
  using `return`?
- `Keep rate=1r/m on req_zone`: the check relies on this rate.

---

## Review questions

- Roughly how many states fit in a 256k zone on a 64-bit system, and how would
  you estimate the minimum size for 1500 keys?
- What does NGINX do first when a `limit_req` zone is full, and when does a
  request fail with an error instead?
- This lab's key contains `$arg_id`, which the client chooses. What could a real
  client do with such a key, and which key would you use to limit per client?
