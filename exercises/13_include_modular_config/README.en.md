# 13 – Include & Modular Config Design

## Goal

In this exercise, you will learn the **actual behavior** of the `include` directive.

Objectives:
- Keep `nginx.conf` simple by splitting the configuration into separate files
- See that included file contents are processed **as if they were inserted at the include line**
- Practice context rules (`http`/`server`/`location`)

---

## Topic

- The `include` directive
- Modular configuration design
- Context behavior: included content is parsed **where the `include` appears**

---

## Task

The exercise uses this structure:

- `nginx.conf` contains an `http { ... }` block
- `conf.d/*.conf` is included inside `http` to load **server** files
- `conf.d/10-server.conf` contains a `server { ... }` block
- `server.d/*.conf` is included inside `server` to load **location** snippets

Complete these steps:

1. In `nginx.conf`, add this include inside `http {}`:
   - `/etc/nginx/conf.d/*.conf`

2. In `conf.d/10-server.conf`, add this include inside `server {}`:
   - `/etc/nginx/server.d/*.conf`

3. In `server.d/20-modular.conf`, create the `/modular` endpoint:
   - `GET /modular` response: `include-ok\n`

4. In `server.d/30-modular-extra.conf`, create the `/modular2` endpoint:
   - `GET /modular2` response: `include2-ok\n`

---

## Expected results

These endpoints must work:

- `GET /`        → `base-ok\n`
- `GET /modular` → `include-ok\n`
- `GET /modular2`→ `include2-ok\n`

---

## Success criteria

- The configuration is valid (`nginx -t` succeeds)
- `/`, `/modular`, and `/modular2` return the expected output
- `include /etc/nginx/conf.d/*.conf;` is inside `http {}`
- `include /etc/nginx/server.d/*.conf;` is inside `server {}`
- `location` blocks are not placed outside the `server` context

---

## Tip

`include` is not a shortcut; its contents are parsed as if they appeared at that exact point.

Therefore:
- `location {}` is valid only inside `server {}`
- `server {}` is valid only inside `http {}`
- Including a file in the wrong place produces an error such as “directive is not allowed here”
