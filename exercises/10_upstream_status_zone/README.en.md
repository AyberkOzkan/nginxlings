# 10 – Upstream Status in Shared Memory (zone up_mem)

## Goal

In this exercise you will configure an HTTP upstream that stores upstream state in a
shared memory zone using:

    zone up_mem 64k;

You will:

- Define an `upstream` with two backends:

      ex10_backend1:8081
      ex10_backend2:8082

- Enable upstream shared memory with:

      zone up_mem 64k;

- Proxy requests to that upstream.
- Expose which backend handled a request via a response header:

      X-Upstream: <ip:port>

During validation, one backend will be stopped on purpose. After that, requests should
stop being routed to the down backend.

---

## What you edit

You will edit:

    nginx.conf

The validator starts NGINX and two simple HTTP backends as separate containers on
the same Docker network:

- Backend A: `ex10_backend1:8081`
- Backend B: `ex10_backend2:8082`

Your job is to configure NGINX to load balance them and expose upstream selection
information.

Do not use `127.0.0.1` for these endpoints: inside the NGINX container, that
address points back to NGINX rather than to either backend.

The validator checks the configuration before the backend containers exist. Use
Docker's embedded DNS at `127.0.0.11` in the `http` context and enable runtime name
resolution with the `resolve` parameter on each upstream server.

---

## Requirements

Your `nginx.conf` must include:

1. A Docker DNS `resolver` in the `http` context.
2. An upstream named `backend` containing `ex10_backend1:8081` and
   `ex10_backend2:8082`; both server entries must use runtime `resolve`.
3. The `zone up_mem 64k;` directive inside that upstream.
4. A location that proxies requests to the `backend` upstream.
5. An `X-Upstream` response header whose value is `$upstream_addr`.

---

## Success criteria

The automated validation is successful when:

- The image builds and `nginx -t` succeeds.
- Both backends are reached before one is stopped (distribution observed).
- After backend 8082 is stopped:

    - Requests must continue being served by backend 8081.
    - The response header may show a retry chain like:

      X-Upstream: <addr>:8082, <addr>:8081

  This is acceptable and means NGINX tried 8082 first, then retried 8081.
    - It must never show an upstream value that contains only backend 8082 without backend 8081.

---

## Tips

- The upstream `zone` directive must be inside the `upstream` block.
- The Docker resolver belongs in the `http` context, not in the `upstream` block.
- Add the `resolve` parameter to both upstream `server` entries so names can be
  resolved after the backend containers start.
- The header must be added after proxying (use it inside the same `location`).
- `max_fails` / `fail_timeout` can make failover behavior more obvious.
