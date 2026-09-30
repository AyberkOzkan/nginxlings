# 03 – Context Hierarchy (main / http / server / location)

## Goal

In this exercise you will repair an `nginx.conf` whose directives sit in the wrong
contexts, using the messages from `nginx -t` to find each one.

You will practise:

- the nesting `main` > `http` > `server` > `location`,
- reading `"..." directive is not allowed here` and the line number it names,
- deciding which context a directive belongs to.

---

## Background – contexts

An NGINX configuration is a tree of blocks called **contexts**. Each directive is
valid only in certain contexts, and a block directive opens a new context:

    # main: settings for the whole NGINX instance
    worker_processes 2;

    events {
        # connection processing
    }

    http {
        # settings for all HTTP traffic
        keepalive_timeout 30;

        server {
            # one virtual server: address, port, names
            listen 9090;
            server_name docs.example;

            location /api/ {
                # requests whose URI starts with /api/
            }
        }
    }

Things worth knowing:

- Every directive's documentation has a **Context:** line listing where it may
  appear. `nginx -t` enforces it and names the first offending line:
  `"listen" directive is not allowed here in .../nginx.conf:31`.
- `http` and `events` belong to `main`, `server` belongs to `http`, and `location`
  belongs to `server` (or to another `location`).
- A setting in an outer context is inherited by the inner ones unless they set it
  themselves: a `root` in `server` also applies to its locations.
- `nginx -t` stops at the first error. After each fix, the next misplaced
  directive shows up.

Reference: [Beginner's guide – configuration file structure](https://nginx.org/en/docs/beginners_guide.html#conf_structure),
[ngx_http_core_module](https://nginx.org/en/docs/http/ngx_http_core_module.html).

---

## Lab environment

This exercise uses the NGINX installed on your machine:

- `nginx.conf` is your file. Its top part (`main`, `events`, `http`) is valid; the
  section marked `INTENTIONALLY BROKEN CONTEXT SECTION` is not.
- Relative paths such as `logs/error.log` resolve against this directory.

The check (`c` or Enter in Nginxlings, or `./check.sh`) runs
`nginx -t -c nginx.conf -p <this directory>`, starts NGINX in the foreground,
requests `http://127.0.0.1:8080/` and expects a `200` whose body contains
`context-hierarchy-ok`. Port 8080 must be free.

---

## Task

Edit `nginx.conf` so that:

1. `listen 8080;` and `server_name localhost;` belong to the `server` block.
2. The `server` block, with its `location /`, lives inside `http`.
3. `location /` keeps its `return`, so `/` still answers `context-hierarchy-ok`.

---

## Success criteria

- `nginx -t` accepts the configuration.
- `http://127.0.0.1:8080/` answers `200`.
- The body contains `context-hierarchy-ok`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `"listen" directive is not allowed here in ...:31`: this is the starting state.
  Which block does `listen` describe?
- `"server" directive is not allowed here`: the `server` block is still outside
  `http`.
- `"location" directive is not allowed here`: a `location` sits directly in
  `http`. Which block has to surround it?
- `"sendfile" directive is not allowed here` or
  `"worker_processes" directive is not allowed here`: a directive was moved to a
  context it does not support. Check its **Context:** line in the documentation.
- `NGINX did not answer on http://127.0.0.1:8080/`: the configuration is valid,
  but no `server` listens on 8080. Did `listen 8080;` get lost on the way?
- `got 403`: the `return` is gone from `location /`, so NGINX tries to serve a file
  from `root`, and this directory has no `index.html`.

---

## Review questions

- A `server` without any `listen` directive still accepts connections. On which
  port, and why does the answer depend on whether NGINX starts as root?
- `sendfile` is allowed in `http`, `server` and `location`. If `http` says
  `sendfile on;` and one location says `sendfile off;`, what applies to the other
  locations?
- A `server` has `add_header X-A ...;` and one of its locations has
  `add_header X-B ...;`. Which headers does a request to that location get, and
  which does a request to another location get?
