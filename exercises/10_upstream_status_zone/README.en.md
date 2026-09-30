# 10 – Upstream State in Shared Memory (zone up_mem)

## Goal

In this exercise you will build an HTTP upstream whose members are Docker
containers found by name at run time, keep its state in a shared memory zone, show
which backend answered each request, and watch NGINX fail over when one backend
stops.

You will practise:

- the upstream `zone` directive,
- run-time name resolution with `resolve` and Docker's DNS server,
- exposing `$upstream_addr` in a response header,
- how a refused connection makes NGINX retry the next server.

---

## Background – shared state, run-time DNS and failover

Without a zone, every worker process keeps its own copy of an upstream's state:
the round-robin position and the failure counters. `zone <name> <size>;` inside an
`upstream` block moves the group's configuration and state into shared memory, so
all workers see the same state.

Names in an upstream are normally resolved once, when the configuration is loaded.
The `resolve` parameter on a `server` line re-resolves the name while NGINX runs,
using the `resolver` from the `http` context. The addresses it finds are stored in
the zone, so `resolve` only works in an upstream that has one. Open-source NGINX
supports `resolve` since 1.27.3; this image uses 1.28.3.

A generic example (different names, addresses and sizes from this lab):

    http {
        resolver 10.0.0.2 valid=30s;

        upstream api_pool {
            zone api_state 128k;
            server api-a.internal:9000 resolve;
            server api-b.internal:9000 resolve;
        }

        server {
            listen 80;

            location /api/ {
                proxy_pass http://api_pool;
            }
        }
    }

Things worth knowing:

- Inside a container on a user-defined Docker network, Docker's embedded DNS
  server answers on `127.0.0.11`. `127.0.0.1` inside the NGINX container is NGINX
  itself, never one of the backends.
- `$upstream_addr` holds the address of the server that answered. After a retry it
  lists every address tried, separated by commas.
- `add_header` adds a response header. Without the `always` parameter it is only
  added to successful and redirect responses (200, 201, 204, 206, 301, 302, 303,
  304, 307, 308).
- When a connection is refused, the default `proxy_next_upstream error timeout;`
  passes the request to the next server in the group.
- Open-source NGINX does not report upstream state through an API or a status page
  (that is an NGINX Plus feature), so this lab makes the chosen backend visible
  with a response header instead.

Reference: [ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html)
(`zone`, `server ... resolve`), [resolver](https://nginx.org/en/docs/http/ngx_http_core_module.html#resolver),
[add_header](https://nginx.org/en/docs/http/ngx_http_headers_module.html#add_header).

---

## Lab environment

The check starts three containers on one Docker network:

- **ex10_backend1:8081** and **ex10_backend2:8082**: two small BusyBox HTTP servers
  that answer `backend=8081` and `backend=8082`.
- The NGINX container, built by `Dockerfile` on `nginx:1.28.3-alpine` with your
  `nginx.conf`, published on `http://127.0.0.1:19100/`.

`nginx.conf` is your file. The Docker DNS `resolver 127.0.0.11` line is already in
`http`; the `upstream backend` block has a placeholder server; `location /` still
returns placeholder text.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/10_upstream_status_zone:check`) and runs `nginx -t` **before the
backend containers exist**. It then starts everything, sends 10 requests and reads
the `X-Upstream` header, stops the HTTP server inside `ex10_backend2` (its container
keeps running, so connections to 8082 are refused) and sends 12 more requests.

---

## Task

Edit `nginx.conf`:

1. In `upstream backend`, add a shared memory zone named **up_mem** with a size of
   **64k**.
2. Replace the placeholder server with `ex10_backend1:8081` and
   `ex10_backend2:8082`, both resolved at run time.
3. Make `location /` proxy requests to the `backend` upstream instead of returning
   text.
4. Add a response header **X-Upstream** whose value is `$upstream_addr`.
5. Keep the `resolver` line.

---

## Success criteria

- The image builds and `nginx -t` accepts the configuration.
- The active configuration contains the `up_mem` zone of 64k and the
  `X-Upstream` header set to `$upstream_addr`.
- Before the stop, the `X-Upstream` values of the 10 requests include both 8081
  and 8082.
- After the stop, all 12 requests answer `200` through 8081. A retry chain such as
  `<addr>:8082, <addr>:8081` is fine.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `No shared memory zone named up_mem with a size of 64k`: this is the starting
  state. The name and the size must match exactly.
- `resolving names at run time requires upstream "backend" ... to be in shared
  memory`: `resolve` is there, but the zone is not.
- `host not found in upstream "ex10_backend1:8081"`: the name was resolved when
  the configuration was loaded, and the backend containers do not exist yet at
  that moment. Which `server` parameter postpones the lookup?
- `host not found in upstream "ex10_backend1"`: `proxy_pass` names a container
  directly instead of the upstream group.
- `no resolver defined to resolve names at run time`: the `resolver` line is gone.
- `No active X-Upstream response header`: the header is missing, commented out,
  or set to another value.
- `Before stopping, expected both upstream ports in X-Upstream. Got 8081=0,
  8082=0`: the responses carry no upstream address. Is `location /` still
  returning text instead of proxying?
- `After stopping backend2, 1 of 12 requests did not succeed through 8081`: the
  first request after the stop got a `502` instead of being retried. Has retrying
  the next server been switched off?

---

## Review questions

- After the first refused connection to 8082, the following requests go straight
  to 8081. Which `server` parameters decide that, and for how long?
- The check stops the HTTP server but keeps the backend container. What would a
  client notice if the container were removed instead?
- Without `always` on `add_header`, what does the `X-Upstream` header show on a
  `502` response?
- A backend container is recreated and gets a new IP address. What happens to the
  traffic with `resolve`, and what would happen without it?
