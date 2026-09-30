# 01 – Basic Static NGINX Server

## Goal

In this exercise you will run NGINX as a **static web server**: a request for `/`
is answered with a file read from disk.

You will practise:

- mapping a request URI to a file with `root`,
- choosing the default file for a directory with `index`,
- how relative paths are resolved against the NGINX **prefix**.

---

## Background – from URI to file

For a static request, NGINX appends the request URI to the `root` directory and
serves that file. When the URI ends with `/`, it names a directory, so the `index`
directive lists the file names to try in that directory, in order; the first one
that exists is served.

A generic example (different names and ports from this lab):

    server {
        listen 9090;
        root /srv/docs;
        index start.html;
    }

    # GET /          -> /srv/docs/start.html
    # GET /css/a.css -> /srv/docs/css/a.css

Things worth knowing:

- A relative path in `root`, `error_log`, `pid` or `access_log` is resolved
  against the **prefix**, not against your current directory or the directory of
  `nginx.conf`. The prefix is set with `nginx -p <dir>`; without it, NGINX uses
  the prefix it was built with (`nginx -V` shows it).
- If the directory exists but none of the `index` files do, NGINX answers `403`
  (it does not list directories unless `autoindex on;` is set). If the path does
  not exist at all, it answers `404`.
- The error log names the exact path NGINX tried, which shows you where a relative
  `root` really points.
- `return 200 "text";` answers without reading any file. This exercise is about
  serving the file.

Reference: [root](https://nginx.org/en/docs/http/ngx_http_core_module.html#root),
[index](https://nginx.org/en/docs/http/ngx_http_index_module.html),
[Command-line parameters](https://nginx.org/en/docs/switches.html).

---

## Lab environment

This exercise uses the NGINX installed on your machine:

- `nginx.conf` is your file. It listens on port **8080** and still has
  placeholders for `root` and `index`. Its error log goes to `logs/error.log`
  inside this directory.
- `index.html` is the page to serve. It still contains a placeholder.

The check (`c` or Enter in Nginxlings, or `./check.sh`) runs
`nginx -t -c nginx.conf -p <this directory>`, starts NGINX in the foreground with
the same prefix, requests `http://127.0.0.1:8080/`, compares the response with
`index.html` byte for byte and stops NGINX again. Port 8080 must be free.

---

## Task

1. In `nginx.conf`, replace the two placeholders so that a request for `/` is
   answered with `index.html` from this exercise directory. Keep `listen 8080;`.
2. In `index.html`, replace the placeholder so that the page contains the exact
   text `Merhaba Nginxlings`.

---

## Success criteria

- `nginx -t` accepts the configuration.
- `http://127.0.0.1:8080/` answers `200`.
- The response is byte for byte the content of this directory's `index.html`.
- `index.html` contains `Merhaba Nginxlings`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `got 404`: NGINX did not find the directory or the file. Open `logs/error.log`:
  the `is not found` line shows the full path it tried. Which directory does the
  prefix point to, and what does your relative `root` add to it?
- `got 403`: the directory was found, but none of the names in `index` exists
  there (`directory index of "..." is forbidden` in the error log).
- `index.html does not contain 'Merhaba Nginxlings' yet`: NGINX works; now edit
  the page.
- `The response to / is not the content of index.html`: a `return` answers with
  text instead of the file.
- `bind() to 0.0.0.0:80 failed (13: Permission denied)`: ports below 1024 need
  root. Keep port 8080.
- `Port 8080 ... is already in use by ...`: another program holds the port. Stop it
  and run the check again.

---

## Review questions

- `nginx -V` shows a `--prefix=`. Where would `root html;` point if NGINX were
  started without `-p`?
- With `index a.html b.html index.html;` and no `a.html` on disk, which file is
  served for `/`?
- What changes for `GET /` if you add `autoindex on;` and `index` names a file
  that does not exist?
