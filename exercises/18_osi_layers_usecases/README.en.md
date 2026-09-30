# 18 – OSI Layers & Use-Case Mapping (Theory + Mini Lab)

## Goal

In this exercise you will name the role NGINX plays in a scenario and the layer it
works on, and build one Layer 7 path and one Layer 4 path in the same NGINX
instance.

You will practise:

- telling Layer 7 (HTTP) use cases from Layer 4 (TCP/UDP) ones,
- loading a dynamic module with `load_module`,
- running an `http` upstream and a `stream` upstream side by side.

---

## Background – roles and layers

The same NGINX binary plays different roles depending on its configuration:

- `root` / `index` serving files: **web server** (L7)
- `upstream` + `proxy_pass http://...` in `http`: **reverse proxy, L7 load balancer** (L7)
- `proxy_cache` storing upstream responses: **cache** (L7)
- `upstream` + `proxy_pass name` in `stream`: **L4 load balancer** (L4)

At **Layer 7** NGINX parses HTTP: method, URI, headers, host, cookies. It can route
by path, add headers and cache responses. That is why an upstream in `http {}` is
an L7 use case (exercise 07).

At **Layer 4** NGINX only forwards TCP or UDP connections. It does not interpret the
bytes, so there are no URIs or headers to act on. That is why an upstream in
`stream {}` is an L4 use case (exercise 08).

Things worth knowing:

- NGINX modules are either built into the binary or built as **dynamic modules**
  (`.so` files) that `load_module <path>;` loads. `load_module` belongs to the
  `main` context, before the `events` block.
- `nginx -V` shows how a module was built: `--with-stream` means built in,
  `--with-stream=dynamic` means it has to be loaded.
- `http` and `stream` are separate top-level contexts; each has its own `upstream`
  and `server` blocks.

Reference: [load_module](https://nginx.org/en/docs/ngx_core_module.html#load_module),
[ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html),
[ngx_stream_upstream_module](https://nginx.org/en/docs/stream/ngx_stream_upstream_module.html).

---

## Lab environment

Everything runs inside one Docker container built from this directory:

- `Dockerfile` builds the image on `alpine:3.20.10` and installs Alpine's `nginx`
  and `nginx-mod-stream` packages, plus `python3` and `socat` for the backends. The
  stream module file is **/usr/lib/nginx/modules/ngx_stream_module.so**.
- `start.sh` starts the backends, then NGINX:
  - HTTP: **127.0.0.1:8081** answers `l7-backend=8081`, **127.0.0.1:8082** answers
    `l7-backend=8082`.
  - TCP: **127.0.0.1:9001** answers `l4-backend=9001`, **127.0.0.1:9002** answers
    `l4-backend=9002`.
- `nginx.conf` is your file. Its server on port 8080 has a `location /l7` that still
  returns placeholder text; TODO comments mark the module and the stream part.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/18_osi_layers_usecases:check`), reads the active configuration with
`nginx -T`, publishes port 8080 on `127.0.0.1:18180` and port 9000 on
`127.0.0.1:19080`, requests `/l7` until both HTTP backends appear in the
`X-L7-Upstream` header, then opens TCP connections to port 9000 until both TCP
backends have answered.

---

## Task

Edit `nginx.conf`:

1. Load the stream module at the top of the file.
2. In `http`, define an upstream with the two HTTP backends and make `location /l7`
   proxy to it instead of returning text.
3. In `location /l7`, add the response header **X-L7-Upstream** with the address of
   the backend that answered.
4. Add a top-level `stream` block with an upstream of the two TCP backends and a
   server on port **9000** that forwards connections to it.

---

## Success criteria

- `nginx -T` shows the `load_module` line for `ngx_stream_module.so`, a `stream`
  block and the `X-L7-Upstream` header.
- `/l7` responses show both `:8081` and `:8082` in `X-L7-Upstream`.
- TCP connections to port 9000 reach both `l4-backend=9001` and `l4-backend=9002`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `The stream module is not loaded yet`: this is the starting state.
- `NGINX does not know the stream directive yet`: the `stream` block is written,
  but the module that provides it is not loaded.
- `"load_module" directive is specified too late`: it sits after the `events`
  block. Where does the `main` context begin?
- `"stream" directive is not allowed here`: `stream` is nested inside `http`
  (see exercise 08).
- `No X-L7-Upstream header yet`: the header is missing or commented out.
- `/l7 answered without an X-L7-Upstream value`: a `return` still answers `/l7`
  before any proxying, or the header is set in a place that does not apply to
  `/l7` (remember how `add_header` is inherited, exercise 03).
- `Got 8081=60, 8082=0` or `Got 9001=40, 9002=0`: that upstream reaches only one
  backend. Count its `server` lines and compare the ports with `start.sh`.

---

## Review questions

- Name the role and the layer for each scenario: NGINX terminates TLS and proxies to
  HTTP application servers; NGINX spreads PostgreSQL connections over two replicas;
  NGINX keeps backend responses with `proxy_cache`; NGINX sends TLS connections to
  different backends by SNI without decrypting them.
- The stream side has no `X-L7-Upstream` header. Where could you see which TCP
  backend served a connection?
- Exercise 08 used `stream` without any `load_module`. Why did that work there but
  not here?
- `load_module modules/ngx_stream_module.so;` with a relative path also works in
  this image. Relative to what?
