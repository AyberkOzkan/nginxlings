# 12 – Logs & Variables

## Goal

In this exercise you will define your own access log format, fill it with NGINX
variables and send the log to the container's standard output.

You will practise:

- `log_format` and the context it belongs to,
- `access_log` with a named format,
- common request variables and where their values come from.

---

## Background – log formats and variables

`log_format <name> '<string>';` defines a named format in the `http` context. The
string mixes plain text with variables; each request writes one line in which every
variable is replaced by its value. `access_log <path> <format>;` switches logging
on with that format.

A generic example (different names, variables and file from this lab):

    log_format timing '$remote_addr [$time_local] "$request" rt=$request_time';

    server {
        access_log /var/log/nginx/timing.log timing;
    }

Things worth knowing:

- A format has to be defined before the `access_log` that uses it; the name is
  looked up while the configuration is read.
- `access_log` without a format name uses the predefined `combined` format. Even
  with no `access_log` at all, NGINX logs in `combined` to the path it was built
  with.
- `access_log off;` in an inner context switches logging off there.
- A relative log path is resolved against the NGINX prefix, just like `root` in
  exercise 01.
- Some variables: `$remote_addr` is the client address of the connection,
  `$status` the response status, `$request` the full request line (method, URI,
  protocol). `$http_<name>` reads any request header: the name in lower case with
  dashes replaced by underscores.
- In a container, logs written to `/dev/stdout` show up in `docker logs`.

Reference: [ngx_http_log_module](https://nginx.org/en/docs/http/ngx_http_log_module.html),
[variables](https://nginx.org/en/docs/varindex.html).

---

## Lab environment

Everything runs inside one Docker container built from this directory:

- `Dockerfile` builds the image on `nginx:1.28.3-alpine` with your `nginx.conf`.
- `nginx.conf` is your file. Its server answers `200` for `/` on port 8080; the
  TODO comments list what the format and the access log need.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/12_logs_and_variables:check`), reads the active configuration with
`nginx -T`, publishes container port 8080 on `http://127.0.0.1:18120/`, sends five
requests with the User-Agent `nginxlings-test-agent/1.0` and searches
`docker logs` for them.

---

## Task

Edit `nginx.conf`:

1. In `http`, define a log format named **main** that contains at least
   `$remote_addr`, `$status`, `$request` and `$http_user_agent`.
2. Write the access log in the `main` format to **/dev/stdout**.
3. Keep the server on port 8080 that answers `200` for `/`.

---

## Success criteria

- `nginx -T` shows an active `log_format main` with the four variables and an
  active `access_log` that uses `main`.
- After the test requests, `docker logs` contains lines with the test
  User-Agent, the request `GET /`, the status `200` and an IP address.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `No active 'log_format main ...;' found`: this is the starting state.
- `log_format main does not include $...`: the format lacks the variable named in
  the message.
- `No active access_log uses the main format yet`: an `access_log` without a format
  name writes `combined`, not `main`.
- `unknown log format "main"`: the name is misspelled in one place, the format is
  defined after the `access_log` that uses it, or it is not in `http`.
- `open() "/etc/nginx/logs/access.log" failed (2: No such file or directory)`: a
  relative path was resolved against the prefix `/etc/nginx`.
- `No access log line contains the test User-Agent`: is the log written to
  `/dev/stdout`? Is logging switched off in the server?

---

## Review questions

- In this image, `access_log /var/log/nginx/access.log main;` also passes the
  check. Why do those lines reach `docker logs`?
- For a request `GET /?a=1` that is answered through `index.html`, what do
  `$request`, `$request_uri` and `$uri` contain?
- Behind a load balancer, what would `$remote_addr` show, and which module lets
  NGINX log the original client address instead?
