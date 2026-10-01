# 08 – TCP Load Balancer (Layer 4, stream)

## Goal

In this exercise you will turn NGINX into a **Layer 4 (TCP) load balancer** that
spreads raw TCP connections across two backends, using the `stream` context
instead of `http`.

You will practise:

- placing a `stream` block at the right level of `nginx.conf`,
- grouping TCP backends in an `upstream` block inside `stream`,
- handing whole connections to that group with the stream flavour of `proxy_pass`.

---

## Background – http and stream

NGINX handles protocols in separate top-level contexts:

- `http` understands HTTP requests (Layer 7): methods, headers, URIs, `location`.
- `stream` only sees TCP or UDP connections (Layer 4): bytes in, bytes out.

`stream` sits next to `http` at the top level of `nginx.conf`, never inside it.
Each context has its own `upstream` and `server` blocks.

A generic example (different names and addresses from this lab) that balances
Redis connections:

    stream {
        upstream redis_pool {
            server 10.0.0.31:6379;
            server 10.0.0.32:6379;
        }

        server {
            listen 16379;
            proxy_pass redis_pool;
        }
    }

Things worth knowing:

- A stream `server` has no `location` blocks. `proxy_pass` goes directly in the
  `server` and takes an address or an upstream name, **without** a scheme such as
  `http://`.
- Directives that work on HTTP requests (`location`, `proxy_set_header`,
  `add_header`, ...) do not exist in `stream`: there is no request to act on.
- `server_name` does exist in `stream` since 1.25.5, but it matches the TLS
  Server Name Indication (SNI), not an HTTP `Host` header.
- The default method is round-robin, like in `http`. Inside a stream `upstream`,
  `weight=`, `least_conn`, `hash` and `random` change it.
- `stream` can also proxy UDP (`listen ... udp`); this lab uses TCP only.

Reference: [ngx_stream_core_module](https://nginx.org/en/docs/stream/ngx_stream_core_module.html),
[ngx_stream_proxy_module](https://nginx.org/en/docs/stream/ngx_stream_proxy_module.html),
[ngx_stream_upstream_module](https://nginx.org/en/docs/stream/ngx_stream_upstream_module.html).

---

## Lab environment

Everything runs inside one Docker container built from this directory:

- `start.sh` starts two TCP backends with `socat` inside the container, then
  starts NGINX in the foreground. **127.0.0.1:9001** answers every connection with
  the line `backend=9001`, **127.0.0.1:9002** with `backend=9002`.
- `nginx.conf` is your file. It starts with an `upstream tcp_backend` block and a
  `server` on port 9000, both inside `http`, plus an optional log format.
- `Dockerfile` builds the image on `nginx:1.28.3-alpine` and installs `socat`.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/08_stream_load_balancer:check`), runs `nginx -t` and `nginx -T`,
publishes container port 9000 on `127.0.0.1:19000`, opens 10 TCP connections
with Python 3 and reads the first line each one returns.

---

## Task

Edit `nginx.conf`:

1. Fill the `tcp_backend` upstream with the two backends that `start.sh` runs.
2. Move the proxying to Layer 4: the upstream and the server listening on
   **9000** belong in a top-level `stream` block. Convert the starter's `http`
   block instead of adding a second block next to it.
3. In that server, forward every connection to the upstream group.
4. The `log_format` and `access_log` lines are optional. If you keep them, they
   also work inside `stream`.

---

## Success criteria

- The image builds and `nginx -t` accepts the configuration.
- `nginx -T` shows a top-level `stream` block and an active `proxy_pass`.
- Port 9000 returns backend data within 20 seconds of the container starting.
- Of the 10 connections, at least one returns `backend=9001` and at least one
  returns `backend=9002`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `no servers are inside upstream`: the upstream block is still empty.
- `"proxy_pass" directive is not allowed here`: the server is still inside `http`,
  where `proxy_pass` belongs in a `location`. Layer 4 proxying needs `stream`.
- `"stream" directive is not allowed here`: `stream` was nested inside `http`. It
  belongs at the top level of the file, like `events` and `http`.
- `"location" directive is not allowed here` or
  `"proxy_set_header" directive is not allowed here`: these are HTTP directives.
  In `stream`, the `server` itself forwards the connection.
- `invalid host in upstream "http://..."`: stream `proxy_pass` takes a bare
  address or upstream name, without a scheme.
- `invalid URL prefix`: `proxy_pass` is inside an `http` location, which is Layer 7.
- `The container exited during startup` and the logs show
  `bind() to 0.0.0.0:9000 failed (98: Address in use)`: two blocks listen on port
  9000. Is the old `http` server still there next to your `stream` block?
- `Got backend=9001 count=10, backend=9002 count=0` (or the other way round): the
  upstream reaches only one backend. Count its `server` lines and compare the
  ports with `start.sh`.

---

## Review questions

- A configuration where both `http` and `stream` listen on port 9000 passes
  `nginx -t`, yet NGINX cannot start. Why does the test not catch it?
- The starter log format prints `status=$status`, but `stream` sends no HTTP
  responses. What does `$status` mean there, and what would the log show for a
  connection whose backend is down?
- Does the directive you would use in exercise 07 to keep a client on the same
  backend work in a stream `upstream`? If not, what would you use instead?
- What would two stream `server` blocks on the same port with different
  `server_name` values need in order to send connections to different backends?
  Why would that not help with the plain TCP backends in this lab?
