# 11 – Zone Synchronization & API (zone_sync)

## Goal

Expose a Zone Sync status endpoint that behaves like a small JSON API: one exact
URL that answers with a JSON document, served with the right content type.

---

## Background – exact matches, files and content types

**Choosing a location.** For each request NGINX first looks for an exact match
(`location = /path`); if one matches, it is used immediately. Otherwise it
remembers the longest matching prefix location, then tries regular-expression
locations (`~`, `~*`) in order, and a matching regex wins over the prefix. A
prefix location marked `^~` stops the regex search.

**Mapping a URL to a file.**

- `root <dir>` appends the whole request path to `<dir>`.
- `alias <path>` replaces the part of the path that the location matched with
  `<path>`.

**Content type.** For static files NGINX looks up the extension of the
**request path** (not of the file on disk) in its `types` map, usually loaded
with `include mime.types;`. If the path has no extension or the map has no
match, it uses `default_type`, which is `text/plain` unless you set it.

A generic example (different paths from this lab):

    location = /health {
        default_type text/plain;
        return 200 "ok\n";
    }

    location /downloads/ {
        # /downloads/app.zip is read from /srv/files/app.zip
        alias /srv/files/;
    }

Reference: [location](https://nginx.org/en/docs/http/ngx_http_core_module.html#location),
[alias](https://nginx.org/en/docs/http/ngx_http_core_module.html#alias),
[default_type](https://nginx.org/en/docs/http/ngx_http_core_module.html#default_type).

---

## Lab environment

- `Dockerfile` copies your `nginx.conf` and the provided `api/zone_sync.json`
  into an `nginx:1.28.3-alpine` image. The JSON file ends up at
  `/usr/share/nginx/html/api/zone_sync.json`. The image also has `curl` and `jq`
  for the check.
- `nginx.conf` has a `server` on port 8080 whose `location /` answers 404. It
  does not include `mime.types`.
- Do not change the JSON file; it already contains the fields the check needs.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image, runs
`nginx -t`, starts the container (host port 18110), requests
`/api/8/stream/zone_sync` inside the container, and checks the status, the
`Content-Type` header and the JSON fields with `jq`.

---

## Task

Edit `nginx.conf`:

1. Add a location that matches exactly `/api/8/stream/zone_sync`, not as a
   prefix.
2. Serve the provided JSON file for that path.
3. Make the response's `Content-Type` `application/json`.

---

## Success criteria

- `/api/8/stream/zone_sync` returns HTTP 200.
- The response has `Content-Type: application/json`.
- The body is valid JSON with `.zones.sync_zone.state`,
  `.zones.sync_zone.last_sync`, and `.zones.sync_zone.peers` holding exactly two
  items.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `Expected HTTP 200, got 404`: first ask which location answered. If it was
  `location /`, your location does not match the path. If it was yours, the path
  does not lead to the JSON file: what does `root` do with the request path, and
  what does `alias` do?
- `Expected Content-Type to include application/json, got text/plain`: the file
  was served, but the request path has no extension, so NGINX found no type
  for it. Which directive sets the fallback type?
- `Missing JSON field` or `Expected exactly 2 peers`: the body is not the
  provided file. Are you serving `zone_sync.json`?

---

## Review questions

- Which location answers `/api/8/stream/zone_sync/` (with a trailing slash),
  and why?
- Suppose the same server also had `location /api/ { return 403; }` and
  `location ~ ^/api/8/ { return 410; }`. Which one answers
  `/api/8/stream/zone_sync` if your location were a plain prefix, and what
  changes with `=` or with `^~`?
- If your location used `root /usr/share/nginx/html;`, which file would NGINX
  try to open?
- How does NGINX normally choose the `Content-Type` of a static file? Why would
  adding `include mime.types;` not fix the content type for this URL?
