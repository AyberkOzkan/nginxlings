# 07 – HTTP Load Balancer (Layer 7)

## Goal

In this exercise you will configure NGINX as a **HTTP load balancer**
running entirely inside a Docker container.

The goals are:

- Run two backend HTTP servers inside the same container:
  - 127.0.0.1:8081
  - 127.0.0.1:8082
- Use an `upstream backend` block that balances requests between both servers.
- Expose a frontend `server` that listens on port 8080.
- Proxy all `/` requests with `proxy_pass http://backend;`.
- Use `$upstream_addr` in the log format to see which backend handled each request.

All of this will be tested via Docker; the host NGINX installation (if any) is not touched.

---

## Background – upstream and round-robin

A typical NGINX HTTP load balancer looks like this:

    upstream backend {
        server 127.0.0.1:8081;
        server 127.0.0.1:8082;
    }

    server {
        listen 8080;
        server_name localhost;

        location / {
            proxy_pass http://backend;
        }
    }

By default, NGINX uses a **round-robin** algorithm: each new request goes to the next server in the upstream list.

If you add `$upstream_addr` to your log format, you can see which backend served each request:

    log_format lb_main
        '$remote_addr - $remote_user [$time_local] '
        '"$request" $status $body_bytes_sent '
        '"$http_referer" "$http_user_agent" '
        'upstream=$upstream_addr';

    access_log /dev/stdout lb_main;

---

## What you do in this exercise

You will work in this directory:

    07_http_load_balancer/

Files:

- `nginx.conf`  
  NGINX configuration skeleton. Contains TODO comments for the `upstream backend`
  and the frontend `server` block.
- `Dockerfile`  
  Builds an image based on `nginx:alpine`, copies `nginx.conf` and `start.sh`
  into the image.
- `start.sh`  
  Starts two backend HTTP servers (127.0.0.1:8081 and 127.0.0.1:8082) inside the
  container, then starts NGINX in the foreground.
- `check.sh`  
  Automatically tests this exercise:
  - Builds the Docker image,
  - Starts a container,
  - Sends several HTTP requests,
  - Parses the logs and checks that requests are distributed across both backends.

The main file you will edit is: `nginx.conf`.

---

## Steps

1. Go to the directory

    cd exercises/07_http_load_balancer

2. Inspect `nginx.conf`

3. Complete the `upstream backend` block

   You should define two backend servers:

    upstream backend {
        server 127.0.0.1:8081;
        server 127.0.0.1:8082;
    }

4. Configure `location /` with `proxy_pass`

   The frontend server listening on 8080 should send all `/` requests to the upstream:

    server {
        listen 8080;
        server_name localhost;

        location / {
            proxy_pass http://backend;
        }
    }

   You may add extra `proxy_set_header` directives if you like (not required).

5. Run the automatic check

    ./check.sh

   The script will:

   - Build a Docker image from the current directory (for example `nginxlings_http_lb_07`),
   - Start a container from this image,
   - Inside the container:
     - Start two backend HTTP servers on 127.0.0.1:8081 and 127.0.0.1:8082,
     - Start NGINX listening on port 8080,
   - From the host, send several HTTP requests to `http://127.0.0.1:18080/`,
   - Collect the container logs and check that `$upstream_addr` includes both
     127.0.0.1:8081 and 127.0.0.1:8082.

If something is wrong, `check.sh` will show the relevant error messages and a snippet of the logs.

---

## Success criteria

This exercise is considered successful when:

- The `docker` command is available and working on the host.
- `Dockerfile`, `start.sh` and `nginx.conf` exist and the image builds successfully.
- Inside the container, `nginx -t` reports a valid configuration.
- The `upstream backend` block contains **at least two servers**:
  - 127.0.0.1:8081
  - 127.0.0.1:8082
- The frontend `server` block:
  - Listens on port 8080,
  - Has a `location /` that proxies to `http://backend`.
- The log format includes `$upstream_addr` and the `access_log` uses this format.
- According to `check.sh`:
  - At least one request is handled by 127.0.0.1:8081,
  - At least one request is handled by 127.0.0.1:8082.

---

## Tips

- Any syntax error in `nginx.conf` will prevent the container from starting;
  `check.sh` will show the `nginx -t` output in that case.
- If `$upstream_addr` is missing from the log format, it becomes hard to see
  which backend handled which request.
- When the check fails, carefully read the sample of logs printed by `check.sh`;
  you should be able to see which upstream addresses are used.

