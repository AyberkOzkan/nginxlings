# 15 – HTTPS (TLS) Quick Lab

## Goal

In this exercise you will make NGINX **terminate TLS**: it serves a response over
HTTPS with a certificate built into the image, and it sends plain HTTP clients to
the HTTPS address.

You will practise:

- enabling TLS on a listening socket,
- pointing NGINX at a certificate and its private key,
- redirecting HTTP to HTTPS without losing the host name and path.

---

## Background – TLS termination and the HTTP redirect

With TLS termination, NGINX performs the TLS handshake with the client, decrypts
the traffic and handles the plain HTTP request inside. A typical site uses two
`server` blocks: one that accepts encrypted connections, and one on the plain HTTP
port that only tells clients to come back over HTTPS.

A generic example (different names, ports and files from this lab):

    server {
        listen 8080;
        server_name shop.example;
        return 308 https://$host:8443$request_uri;
    }

    server {
        listen 8443 ssl;
        server_name shop.example;

        ssl_certificate     /etc/ssl/shop/fullchain.pem;
        ssl_certificate_key /etc/ssl/shop/privkey.pem;

        location / {
            root /srv/shop;
        }
    }

Things worth knowing:

- TLS is switched on by the `ssl` parameter of `listen`. The separate `ssl on;`
  directive was removed in NGINX 1.25.1.
- `ssl_certificate` names the certificate (PEM, optionally followed by its chain)
  and `ssl_certificate_key` names the private key. Both are required, and
  `nginx -t` loads the files, so a wrong path fails the test, not the first
  request.
- `return code URL;` in a `server` block answers before any `location` is chosen.
  `$host` keeps the host name the client asked for; `$request_uri` keeps the
  original path and query string.
- A redirect target that starts with `/` is completed with the scheme and host of
  the current request, which gives `http://` on the plain HTTP server. A target
  without a scheme that does not start with `/` is sent unchanged.
- The certificate in this lab is self-signed: no certificate authority vouches for
  it, so clients have to be told to accept it (the check uses `curl -k`).

Reference: [ngx_http_ssl_module](https://nginx.org/en/docs/http/ngx_http_ssl_module.html),
[Configuring HTTPS servers](https://nginx.org/en/docs/http/configuring_https_servers.html),
[return](https://nginx.org/en/docs/http/ngx_http_rewrite_module.html#return).

---

## Lab environment

Everything runs inside one Docker container built from this directory:

- `Dockerfile` builds the image on `nginx:1.28.3-alpine`, installs `openssl` and
  generates a self-signed certificate for `CN=localhost` at build time:
  **/etc/nginx/certs/self.crt** (certificate) and **/etc/nginx/certs/self.key**
  (private key). You do not need to change it.
- `nginx.conf` is your file. Its `http` block is empty apart from a few defaults;
  the TODO comments list what each server block needs.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/15_https_tls:check`), runs `nginx -T`, publishes container port 80 on
`127.0.0.1:18150` and port 443 on `127.0.0.1:18450`, sends an HTTP request with
`Host: localhost` and reads its headers, then fetches `/` over HTTPS with
`curl -k`.

---

## Task

Edit `nginx.conf` and add two separate `server` blocks inside `http`:

1. An HTTP server on port **80** with `server_name localhost` that answers every
   request with a redirect to the `https://` URL of the same host and path.
2. An HTTPS server on port **443** with TLS enabled and `server_name localhost`
   that uses the certificate and key the `Dockerfile` generates and answers `/`
   with the body `tls-ok` followed by a newline.

---

## Success criteria

- The image builds and `nginx -t` accepts the configuration.
- The active configuration contains `listen 80` and `listen 443 ssl`.
- Port 80 answers with a redirect (301, 302, 303, 307 or 308) whose `Location`
  starts with `https://`.
- Port 443 answers over TLS with a body containing `tls-ok`.
- The certificate and key exist at the paths above inside the container.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `No active 'listen 80;' yet`: this is the starting state. Add the HTTP server
  block first.
- `No active 'listen 443 ssl;' yet`: the HTTPS block is missing, or its `listen`
  line does not carry the `ssl` parameter.
- `unknown directive "ssl"`: `ssl on;` no longer exists. Where does TLS get
  switched on instead?
- `no "ssl_certificate" is defined for the "listen ... ssl" directive`: a TLS
  server needs both the certificate and the key.
- `cannot load certificate "...": BIO_new_file() failed ... No such file`: the path
  does not match what the `Dockerfile` generates. Read its `openssl req` step.
- `cannot load certificate "...": PEM_read_bio_X509_AUX() failed`: the file given
  to `ssl_certificate` is not a certificate. Are the certificate and key swapped?
- `HTTP endpoint did not redirect to an https:// URL`: read the `Location` line the
  check prints. `http://...` or a value without any scheme means the redirect
  target does not start with `https://`.
- `HTTPS endpoint did not return the expected body: tls-ok`: look at the body the
  check prints. A `404 Not Found` page means the HTTPS server has nothing to answer
  with. A `301 Moved Permanently` page means the redirect also runs on port 443:
  are both `listen` lines in the same server block?

---

## Review questions

- Why does the check need `curl -k`? What would a browser show for this
  certificate, and what would it take to get one that browsers trust?
- A form sends a `POST` to the HTTP address. What can happen to the method and the
  body with a `301` redirect, and what changes with `308`?
- Replace `$host` with `$server_name` in your redirect and request the HTTP port
  with `Host: shop.example`. Where does the redirect point now, and why?
- Your configuration has no `ssl_protocols` line. Which TLS versions does this
  server accept, and how would you prove it from inside the container?
- How do you enable HTTP/2 on the HTTPS server in NGINX 1.28, and what does
  `nginx -t` say about `listen 443 ssl http2;`?
