# 06 – HTTP Reverse Proxy Basics

## Goal

In this exercise you will configure NGINX as a simple **HTTP reverse proxy**.

You will:

- Keep a small backend HTTP app listening on `http://127.0.0.1:8080`,
- Configure NGINX to listen on port 8081 (the check connects to `127.0.0.1:8081`),
- Forward requests from NGINX to the backend using `proxy_pass`,
- Make sure the client only talks to NGINX, not directly to the backend.

An automatic check script in this directory will verify your configuration.

---

## Background – What is a reverse proxy?

A reverse proxy is a server that:

- Accepts client connections (for example on `127.0.0.1:8081`),
- Forwards the request to one or more backend servers,
- Returns the backend’s response to the client.

For NGINX, a minimal reverse proxy configuration usually looks like:

    http {
        server {
            listen 8081;
            server_name localhost;

            location / {
                proxy_pass http://127.0.0.1:8080;
            }
        }
    }

In this example:

- The **client** connects to NGINX on `127.0.0.1:8081`,
- NGINX forwards the request to `127.0.0.1:8080`,
- The backend responds,
- NGINX sends the response back to the client.

The client never talks directly to the backend.

---

## What you do in this exercise

You work in this directory:

    06_reverse_proxy/

The important files are:

- `nginx.conf` – your NGINX configuration,
- `backend_app.py` – a tiny HTTP backend that responds with a simple text body,
- `check.sh` – the automatic check script.

Your task:

1. Configure `nginx.conf` so that:
   - NGINX listens on port 8081 (HTTP),
   - There is a `server` block under `http`,
   - Inside `server` there is a `location /` block that:
     - Uses `proxy_pass` to send requests to `http://127.0.0.1:8080`.

2. Keep the basic structure/log settings consistent with other exercises:
   - Use a `logs/` directory for `error_log` and `access_log`,
   - Keep `events { ... }` and `http { ... }` contexts valid.

3. The backend app will be started automatically by the check script.
   You do **not** need to run it manually.

You do **not** have to touch `backend_app.py` or `check.sh` for this exercise.

---

## Success criteria

This exercise is considered successful when:

- `nginx.conf` is syntactically valid and passes `nginx -t` with:
    - `-c nginx.conf`
    - `-p` pointing to this directory
- NGINX can start with your configuration (no runtime errors),
- A request to `http://127.0.0.1:8081/` is forwarded to the backend on
  `http://127.0.0.1:8080/`,
- The response body that reaches the client clearly comes from the backend
  (the automatic check script verifies this),
- The automatic check script in this directory reports success.

---

## Tips

- Keep the context hierarchy correct:
  - `events` and `http` are at the main level,
  - `server` is inside `http`,
  - `location` is inside `server`.

- A simple pattern that usually works:

      user  www-data;
      worker_processes auto;

      error_log  logs/error.log;
      pid        logs/nginx.pid;

      events {
          worker_connections 1024;
      }

      http {
          access_log  logs/access.log;

          server {
              listen 8081;
              server_name localhost;

              location / {
                  proxy_pass http://127.0.0.1:8080;
              }
          }
      }

- If you want to follow common best practices, you can also add:

      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

  These are not strictly required for the basic functionality, but they
  are typical for production reverse proxies.





