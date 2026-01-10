# 10 – Upstream Status in Shared Memory (zone up_mem)

## Goal

In this exercise you will configure an HTTP upstream that stores upstream state in a
shared memory zone using:

    zone up_mem 64k;

You will:

- Define an `upstream` with two backends:

      127.0.0.1:8081
      127.0.0.1:8082

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

The validator starts two simple HTTP backends as separate containers in the same Docker network:

- Backend A: 127.0.0.1:8081
- Backend B: 127.0.0.1:8082

Your job is to configure NGINX to load balance them and expose upstream selection
information.

---

## Requirements

Your `nginx.conf` must include:

1) An upstream with shared memory:

      upstream backend {
          zone up_mem 64k;
          server 127.0.0.1:8081;
          server 127.0.0.1:8082;
      }

2) A server that proxies to the upstream:

      location / {
          proxy_pass http://backend;
      }

3) A response header that shows upstream selection:

      add_header X-Upstream $upstream_addr always;

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
- The header must be added after proxying (use it inside the same `location`).
- `max_fails` / `fail_timeout` can make failover behavior more obvious.
