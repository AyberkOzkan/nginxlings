# 13 – Include & Modular Config Design

## Goal

In this exercise you will split one NGINX configuration across several files
with `include`, and see what `include` really does: the included text is parsed
**as if it were written at the `include` line**, in that line's context.

---

## Background – how include works

- `include <path>;` inserts the contents of the file (or of every file matched by
  a glob such as `*.conf`) at that point. Nothing about the included file tells
  NGINX its context: a file with `server {}` blocks only works where `server` is
  allowed (inside `http {}`), a file with `location {}` blocks only inside
  `server {}`.
- Files matched by a glob are loaded in alphabetical order, which is why modular
  setups often start file names with numbers.
- A relative path is resolved against the configuration prefix (`/etc/nginx` in
  this lab's image).
- `nginx -T` tests the configuration and prints the whole effective result, with
  a `# configuration file ...:` header before each file that was loaded. It is
  the quickest way to see what an `include` actually pulled in.

A generic example (different directories and names from this lab):

    http {
        include /etc/nginx/sites/*.conf;          # each file holds one server {}
    }

    # /etc/nginx/sites/shop.conf
    server {
        listen 80;
        server_name shop.example;
        include /etc/nginx/snippets/headers.conf; # add_header lines
    }

Reference: [include](https://nginx.org/en/docs/ngx_core_module.html#include).

---

## Lab environment

- `Dockerfile` copies `nginx.conf` to `/etc/nginx/nginx.conf`, the `conf.d/`
  files to `/etc/nginx/conf.d/` and the `server.d/` files to
  `/etc/nginx/server.d/` in an `nginx:1.28.3-alpine` image. That image already
  has its own `/etc/nginx/conf.d/default.conf`.
- `conf.d/10-server.conf` holds a `server` on port 8080 whose `location /`
  answers `base-ok`.
- `server.d/20-modular.conf` and `server.d/30-modular-extra.conf` only contain
  TODO comments.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image, reads
the effective configuration with `nginx -T` (comment lines are ignored),
confirms that both directories are included, starts the container (host port
18130), and requests `/`, `/modular` and `/modular2`. Each body must match
exactly; a trailing newline is ignored.

---

## Task

1. In `nginx.conf`, inside `http {}`, load every `.conf` file in
   `/etc/nginx/conf.d/`.
2. In `conf.d/10-server.conf`, inside `server {}`, load every `.conf` file in
   `/etc/nginx/server.d/`.
3. In `server.d/20-modular.conf`, make `/modular` answer `include-ok` followed by
   a newline.
4. In `server.d/30-modular-extra.conf`, make `/modular2` answer `include2-ok`
   followed by a newline.

---

## Success criteria

- `nginx -t` accepts the configuration.
- The effective configuration includes `/etc/nginx/conf.d/*.conf` and
  `/etc/nginx/server.d/*.conf`.
- `GET /` returns `base-ok`, `GET /modular` returns `include-ok` and
  `GET /modular2` returns `include2-ok`, each with HTTP 200.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `nginx.conf does not load the .conf files in /etc/nginx/conf.d/ yet`: the first
  include is missing, or it is still a comment.
- `NGINX rejected the configuration: "server" directive is not allowed here in
  /etc/nginx/conf.d/10-server.conf`: the `conf.d` include sits outside
  `http {}`. The error points at the included file, but the fix is where the
  `include` line is.
- `"location" directive is not allowed here in /etc/nginx/server.d/...`: the
  `server.d` include is in `http {}` instead of `server {}`.
- `Expected GET /modular to return ... Is location /modular defined`: the
  request was answered by `location /` (`base-ok`), so no location for
  `/modular` was loaded.
- `duplicate location`: two included files define the same location.

---

## Review questions

- In which order are the files matched by `server.d/*.conf` loaded, and why do
  many setups start file names with numbers?
- Your `conf.d` include loads one more `server` block that you did not write.
  Which one, and why does it not clash with yours?
- How can you see the complete configuration NGINX loaded, file by file?
- What happens if `20-modular.conf` and `30-modular-extra.conf` both define a
  location for `/modular`?
