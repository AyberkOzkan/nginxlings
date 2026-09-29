# 18 – OSI Layers & Use-Case Mapping (Theory + Mini Lab)

## Goal

In this exercise you will learn how to identify the role NGINX is playing in different scenarios.

Focus:

- The difference between L4 (TCP/UDP) and L7 (HTTP/HTTPS)
- Why an HTTP upstream is an L7 use case
- Why a stream upstream is an L4 use case
- How to distinguish reverse proxy, load balancer, web server, and cache roles

---

## Background

NGINX can play multiple roles depending on how it is configured.

Examples:

- Serving static files -> **web server**
- Forwarding HTTP requests to HTTP backends -> **L7 reverse proxy / L7 load balancer**
- Forwarding raw TCP streams to TCP backends -> **L4 load balancer**
- Storing responses for reuse -> **cache**

The key difference is:

### L7 example
NGINX understands the HTTP layer:
- request path
- headers
- host
- method
- cookies

That is why an upstream in `http {}` is an L7 use case.

### L4 example
NGINX only forwards TCP/UDP streams:
- it does not need to interpret HTTP headers
- it can route traffic without understanding application-level content

That is why an upstream in `stream {}` is an L4 use case.

---

## Mini lab

In this exercise you will build two paths:

1. **HTTP / L7**
   - requests to `/l7` go to an HTTP upstream group
   - NGINX acts as a reverse proxy / L7 load balancer

2. **TCP / L4**
   - raw TCP connections on port `9000` go to a stream upstream group
   - NGINX acts as an L4 load balancer

---

## Task

In `nginx.conf`:

- load the stream module
- define one `http` upstream
- define one `stream` upstream
- proxy `/l7` to the HTTP upstream
- forward port `9000` to the stream upstream

On the HTTP side:
- expose the address of the backend that answered (`$upstream_addr`) in the
  `X-L7-Upstream` response header

The container's `start.sh` starts the backends before NGINX:

- HTTP backends: `127.0.0.1:8081` and `127.0.0.1:8082`
- TCP backends: `127.0.0.1:9001` and `127.0.0.1:9002`

The image installs Alpine's `nginx-mod-stream` package. Its module file is
`/usr/lib/nginx/modules/ngx_stream_module.so`, and `load_module` belongs at the
top of the file, before the `events` block.

---

## Expected result

- `GET /l7` requests are distributed across two HTTP backends
- TCP connections to port `9000` are distributed across two TCP backends
- You should be able to identify:

  - `http upstream` -> L7
  - `stream upstream` -> L4
  - static file serving -> web server
  - cached responses -> cache role

---

## Success criteria

- Configuration is valid
- `/l7` is proxied to a working HTTP upstream
- `X-L7-Upstream` header is visible
- Both TCP backends are observable through port `9000`
- Both `http` and `stream` structures appear in the effective config