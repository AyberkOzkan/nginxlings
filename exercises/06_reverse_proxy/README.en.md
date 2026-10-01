# 06 – HTTP Reverse Proxy Basics

## Goal

In this exercise you will put NGINX in front of a small backend application as
an **HTTP reverse proxy**: clients talk only to NGINX, and NGINX fetches the
answer from the backend.

---

## Background – what a reverse proxy does

A reverse proxy accepts the client's connection, opens its own connection to a
backend server, forwards the request, and returns the backend's response. The
client never sees the backend's address.

In NGINX this is the job of `proxy_pass` inside a `location`. It takes a URL
with a scheme (`http://` or `https://`) and an address. A generic example
(different names and addresses from this lab):

    server {
        listen 80;

        location /shop/ {
            proxy_pass http://10.0.0.20:5000;
        }
    }

Things worth knowing:

- A `location` without a handler such as `proxy_pass` or `return` serves files
  from the `root` directory (by default `html/` under the NGINX prefix).
- Whether the `proxy_pass` URL ends with a path changes which path the backend
  receives. Without one, the request path is passed on as it is.
- By default NGINX does not pass the client's `Host` header or address on to the
  backend; `proxy_set_header` controls what the backend receives.

Reference: [proxy_pass](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass).

---

## Lab environment

This exercise runs NGINX directly on your machine, not in Docker:

- `backend_app.py` is a tiny HTTP backend. The check starts it on
  **127.0.0.1:8080**; you do not run it yourself.
- `nginx.conf` is your file. The `events`/`http` structure, the logs under
  `logs/`, and a `server` listening on **8081** are ready; `location /` is empty.

The check (`c` or Enter in Nginxlings, or `./check.sh`) starts the backend with
a random token that only the backend knows, runs `nginx -t` with this directory
as the prefix, starts NGINX, and requests `http://127.0.0.1:8081/`. The answer
must be HTTP 200 **and** contain the backend's token, so a response that NGINX
makes up itself does not pass. Ports 8080 and 8081 must be free; if they are
not, the check names the process that holds them.

---

## Task

Edit `nginx.conf` so that `location /` hands every request to the backend on
`127.0.0.1:8080`. Keep `listen 8081;` and the rest of the file as it is.

---

## Success criteria

- `nginx -t` accepts the configuration.
- `http://127.0.0.1:8081/` answers with HTTP 200.
- The response body is the backend's (it contains the backend's token).

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `got 404: location / has no handler`: the location is still empty, so NGINX
  looks for files instead of asking the backend.
- `got 502: NGINX tried to proxy, but nothing answered`: the request went to the
  wrong address or port. Which port does the backend use?
- `NGINX rejected the configuration: invalid URL prefix`: `proxy_pass` needs a
  URL that starts with a scheme.
- `The response via NGINX does not come from the backend`: NGINX answered by
  itself, for example with a `return`. Replace it with proxying.

---

## Review questions

- With `location /app/`, which path does the backend receive for a request to
  `/app/x` when `proxy_pass` has no path after the address, and which when it
  ends with `/`?
- Which `Host` header does the backend receive by default, and how would you
  pass on the `Host` the client sent?
- The backend only sees connections from NGINX. How can it still learn the
  client's IP address?
- What does the client get when the backend refuses the connection, and which
  directives limit how long NGINX waits to connect and to read the response?
