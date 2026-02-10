# 13 – Include & Modular Config Design

## Goal
Practice the real behavior of `include` in NGINX by splitting config into multiple files.

## What you will build
- `/` returns `base-ok\n` from the main server file.
- `/modular` returns `include-ok\n` from a separate snippet file.
- `/modular2` returns `include2-ok\n` from another snippet file.

## Tasks
1. In `nginx.conf`, enable:
   include /etc/nginx/conf.d/*.conf;
   (must be inside `http {}`)

2. In `conf.d/10-server.conf`, enable:
   include /etc/nginx/server.d/*.conf;
   (must be inside `server {}`)

3. In `server.d/20-modular.conf`, implement:
   location /modular { return 200 "include-ok\n"; }

4. In `server.d/30-modular-extra.conf`, implement:
   location /modular2 { return 200 "include2-ok\n"; }

## Success criteria
- Config is valid (`nginx -t` passes)
- `/` returns `base-ok\n`
- `/modular` returns `include-ok\n`
- `/modular2` returns `include2-ok\n`

## Tip
If you place `location {}` in the wrong context, NGINX will complain with “directive is not allowed here”.
