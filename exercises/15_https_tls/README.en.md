# 15 – HTTPS (TLS) Quick Lab

## Goal

In this exercise, you will set up basic TLS termination with NGINX.

The main focus is:

- Generating a self-signed certificate
- Using `listen 443 ssl;` for HTTPS
- Defining `ssl_certificate` and `ssl_certificate_key`
- Redirecting HTTP requests to HTTPS

---

## Background

NGINX can terminate TLS connections and handle HTTPS traffic for clients.

This setup usually has two parts:

1. An HTTP server  
   It redirects incoming requests on port 80 to HTTPS.

2. An HTTPS server  
   It listens on port 443 with SSL/TLS enabled and uses a certificate.

The basic pattern looks like this:

    server {
        listen 80;
        return 301 https://$host$request_uri;
    }

    server {
        listen 443 ssl;
        ssl_certificate ...;
        ssl_certificate_key ...;
    }

The goal of this exercise is to build this basic structure and make NGINX serve content over HTTPS.

---

## What are you doing in this exercise?

Your task is to:

- Generate a self-signed certificate during the Docker image build
- Create the certificate files under `/etc/nginx/certs/`
- Define one `server` block for HTTP
- Redirect incoming HTTP requests to HTTPS
- Define another `server` block for HTTPS
- Use `listen 443 ssl;` in that block
- Configure `ssl_certificate` and `ssl_certificate_key` with the correct file paths
- Return a simple response over HTTPS

The main concept here is learning how NGINX terminates TLS and redirects plain HTTP traffic to secure HTTPS.

---

## Success criteria

This exercise is considered successful when:

- The Docker image builds successfully
- The NGINX configuration is valid
- The HTTP endpoint returns an HTTPS redirect
- The HTTPS endpoint returns a valid response
- The certificate and key files are used from the correct paths
- Nginxlings marks the exercise as completed

---

## Tips

- You can generate a self-signed certificate with `openssl req -x509`
- The certificate and private key paths must exactly match the NGINX config
- A simple redirect usually looks like `return 301 https://$host$request_uri;`
- The HTTPS server must enable SSL directly on the `listen` directive