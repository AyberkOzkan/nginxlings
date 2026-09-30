# 14 – Multiple Domains, Static Sites

## Goal

In this exercise you will serve two static sites from one NGINX instance on one
port, and NGINX will choose the site from the `Host` header of each request.

You will practise:

- several `server` blocks listening on the same port,
- choosing a server with `server_name`,
- giving each site its own `root`,
- what happens to a request whose host matches no server.

---

## Background – name-based virtual hosts

When several `server` blocks listen on the same address and port, NGINX compares the
request's `Host` header with their `server_name` values and hands the request to the
server that matches. The search order is: the exact name, then the longest wildcard
starting with `*` (`*.example.com`), then the longest wildcard ending with `*`
(`mail.*`), then the first matching regular expression.

A generic example (different names, port and paths from this lab):

    server {
        listen 80;
        server_name shop.example www.shop.example;
        root /srv/shop;
    }

    server {
        listen 80;
        server_name blog.example;
        root /srv/blog;
    }

    # curl -H 'Host: blog.example' http://<address>/  ->  /srv/blog/index.html

Things worth knowing:

- If no `server_name` matches, the request goes to the **default server** of that
  address and port: the first server listed for it, unless another one says
  `listen ... default_server`.
- A `server` block only receives requests for the ports it listens on. A block
  without a `listen` line listens on the default port (80 when NGINX runs as root,
  see exercise 03).
- Each `server` can have its own `root`, so the same URI `/` maps to a different
  directory per site.
- `curl -H 'Host: name' http://127.0.0.1:<port>/` lets you test a name without DNS.

Reference: [How nginx processes a request](https://nginx.org/en/docs/http/request_processing.html),
[Server names](https://nginx.org/en/docs/http/server_names.html).

---

## Lab environment

Everything runs inside one Docker container built from this directory:

- `Dockerfile` builds the image on `nginx:1.28.3-alpine` and copies
  `sites/site1/` to **/usr/share/nginx/site1/** and `sites/site2/` to
  **/usr/share/nginx/site2/**. Each `index.html` contains a marker:
  `site1-static-ok` or `site2-static-ok`.
- `nginx.conf` is your file. Its `http` block has no server yet; the TODO comments
  list the requirements.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/14_multi_domain_static:check`), reads the active configuration with
`nginx -T`, publishes container port 8080 on `http://127.0.0.1:18140/` and requests
`/` twice: once with `Host: site1.local` and once with `Host: site2.local`. It prints
each body.

---

## Task

Edit `nginx.conf` and add two `server` blocks inside `http`:

1. Both listen on port **8080**.
2. One answers for **site1.local** and serves files from `/usr/share/nginx/site1`.
3. The other answers for **site2.local** and serves files from
   `/usr/share/nginx/site2`.
4. `/` on each site returns that site's `index.html`.

---

## Success criteria

- `nginx -T` shows active `server_name` entries for `site1.local` and
  `site2.local`.
- `GET /` with `Host: site1.local` returns the page containing `site1-static-ok`.
- `GET /` with `Host: site2.local` returns the page containing `site2-static-ok`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `No active server block with 'server_name site1.local;' yet`: this is the
  starting state, or the name is misspelled.
- `site2.local did not return its own page` and the body shows `site1-static-ok`:
  the request landed in the other server. Does a server with
  `server_name site2.local` listen on 8080? If no name matches on a port, the
  default server answers.
- `site1.local did not return its own page` and the body shows `site2-static-ok`:
  the two `root` paths are swapped.
- The body is a `404 Not Found` page: `root` points to a directory without that
  site's `index.html`. Compare it with the `COPY` lines of the Dockerfile.
- `Nothing answered on port 8080`: no server listens on 8080 yet.

---

## Review questions

- A request arrives with `Host: other.local`, or with the bare IP address. Which
  site answers? How would you make such requests get no answer at all?
- Remove `index index.html;` from both servers, if you wrote it. Does the check
  still pass? Why?
- `server_name *.site1.local;` in one server and
  `server_name ~^www\.site1\.local$;` in another: which one gets
  `www.site1.local`, and what would beat both?
