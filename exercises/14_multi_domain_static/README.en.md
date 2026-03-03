# 14 – Multiple Domains, Static Sites

## Goal

In this exercise, you will learn how to serve static content for multiple domains on the same NGINX server.

The main focus is:

- Using multiple `server` blocks
- Matching requests by `server_name`
- Assigning a different `root` directory for each domain
- Using NGINX as a static web server

---

## Background

NGINX can route requests to different virtual hosts on the same port by checking the `Host` header.

The common structure looks like this:

    http {
        server {
            listen 8080;
            server_name site1.local;
            root /some/path;
        }

        server {
            listen 8080;
            server_name site2.local;
            root /another/path;
        }
    }

The core idea is:

- If the request is for `site1.local`, the first `server` block should handle it
- If the request is for `site2.local`, the second `server` block should handle it

This allows one NGINX instance to serve multiple static sites.

---

## What are you doing in this exercise?

You will edit the `nginx.conf` file.

Your task is to:

- Create two separate `server` blocks listening on the same port
- Set the first `server_name` to `site1.local`
- Set the second `server_name` to `site2.local`
- Set the first `root` to `/usr/share/nginx/site1`
- Set the second `root` to `/usr/share/nginx/site2`
- Use `index index.html;` in both server blocks
- Make sure the correct static content is served for each domain

The goal is not to use one generic server block. The goal is to define two distinct virtual hosts.

---

## Success criteria

This exercise is considered successful when:

- The configuration syntax is valid
- A request for `site1.local` returns the first static site content
- A request for `site2.local` returns the second static site content
- Both domains are correctly separated on the same port
- Nginxlings marks the exercise as completed

---

## Tips

- Domain-based matching is done with `server_name`
- For static content, `root` and `index` should be configured together
- It is normal to have multiple `server` blocks on the same port
- The main concept here is that NGINX chooses the correct site based on the `Host` header