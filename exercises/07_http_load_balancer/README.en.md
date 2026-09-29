# 07 – HTTP Load Balancer (Layer 7)

## Goal

In this exercise you will turn NGINX into an **HTTP load balancer** that spreads
requests across two backend servers, and you will prove from the access log which
backend answered each request.

You will practise:

- grouping backend servers in an `upstream` block,
- handing requests to that group with `proxy_pass`,
- reading `$upstream_addr` to see the load balancing decision.

---

## Background – upstream and round-robin

An `upstream` block, inside the `http` context, names a group of servers. A
`proxy_pass` whose URL uses that name as its host sends each request to one member
of the group instead of a single address.

A generic example (different names and addresses from this lab):

    upstream app_pool {
        server 10.0.0.11:3000;
        server 10.0.0.12:3000;
    }

    server {
        listen 80;

        location /api/ {
            proxy_pass http://app_pool;
        }
    }

Things worth knowing:

- The default method is **round-robin**: requests go to the servers in turn.
  `weight=` on a `server` changes the share; `least_conn` and `ip_hash` inside the
  `upstream` block change the method.
- `$upstream_addr` holds the address of the server that answered. If NGINX had to
  try more than one server, it lists all of them, separated by commas.
- `return` ends the request in the rewrite phase, before any proxying happens. A
  location that still has a `return` never reaches the upstream.

Reference: [ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html),
[proxy_pass](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass).

---

## Lab environment

Everything runs inside one Docker container built from this directory:

- `start.sh` starts two small HTTP backends on **127.0.0.1:8081** and
  **127.0.0.1:8082** inside the container, then starts NGINX in the foreground.
- `nginx.conf` is your file. It already defines the `lb_main` log format with
  `upstream=$upstream_addr` and sends the access log to stdout.
- `Dockerfile` builds the image on `nginx:1.28.3-alpine`.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/07_http_load_balancer:check`), runs `nginx -t`, publishes container
port 8080 on `http://127.0.0.1:18080/`, sends several requests, and counts the
`upstream=` values in the container logs.

---

## Task

Edit `nginx.conf`:

1. In the `upstream backend` block, replace the placeholder server with the two
   backends that `start.sh` runs.
2. In `location /`, replace the placeholder `return` so that requests are handed
   to the `backend` upstream group.
3. Keep the `log_format lb_main` and `access_log` lines; the check reads the
   `upstream=` field they produce.

---

## Success criteria

- The image builds and `nginx -t` accepts the configuration.
- Requests to port 8080 are answered through the upstream group.
- The access log shows at least one request answered by `127.0.0.1:8081` and at
  least one answered by `127.0.0.1:8082`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `upstream=127.0.0.1:65535` in the log output: the placeholder server is still in
  the upstream block.
- `upstream=-`: the request never reached an upstream. Is the `return` still in
  `location /`?
- Only one backend appears: count the `server` lines in the upstream block, and
  check that both ports match what `start.sh` starts.
- `nginx -t` reports `invalid URL prefix`: `proxy_pass` expects a URL with a
  scheme, and the host part of that URL is the upstream name.

---

## Review questions

- What would the distribution look like with `weight=3` on the first server?
- Which directive would you add to the `upstream` block to always send a client to
  the same backend, and what does it use as the key?
- What does `$upstream_addr` contain when the first backend refuses the connection
  and NGINX retries the second one? (Exercise 10 relies on this.)
