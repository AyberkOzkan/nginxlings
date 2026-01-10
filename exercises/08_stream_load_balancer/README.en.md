# 08 – TCP/UDP Load Balancer (Layer 4, stream)

## Goal

In this exercise you will configure NGINX as a **Layer 4 (TCP) load balancer**
using the `stream {}` context.

You will:

- Define a `stream {}` block (not `http {}`).
- Create an `upstream` with two TCP backends:

      127.0.0.1:9001
      127.0.0.1:9002

- Configure a frontend listener on port 9000 and load balance connections using:

      proxy_pass tcp_backend;

- Observe that connections are distributed across both backends.


---

## Background – HTTP vs stream

NGINX has separate contexts for different protocols:

- `http {}` is for HTTP/HTTPS (Layer 7).
- `stream {}` is for TCP/UDP (Layer 4).

In a TCP load balancer, you do not use `location` blocks. Instead you use:

    stream {
        upstream tcp_backend {
            server 127.0.0.1:9001;
            server 127.0.0.1:9002;
        }

        server {
            listen 9000;
            proxy_pass tcp_backend;
        }
    }

The default balancing method is round-robin.

---

## What you change

You will edit:

    nginx.conf

The container startup script will start two simple TCP services internally:

- Backend A on 127.0.0.1:9001
- Backend B on 127.0.0.1:9002

Your job is to make NGINX accept TCP connections on 9000 and forward them
to the upstream.

---

## Success criteria

The automated validation is successful when:

- The Docker image builds.
- `nginx -t` succeeds inside the container.
- NGINX listens on TCP port 9000 (inside the container).
- Multiple TCP connections through the frontend return responses from both backends
  (so distribution is confirmed).

---

## Tips

- `stream {}` must be a top-level block in `nginx.conf` (not inside `http {}`).
- For stream load balancing you use:

      proxy_pass <upstream_name>;

  without `http://`.
- Avoid HTTP directives (`server_name`, `location`, `proxy_set_header`) in `stream {}`.
