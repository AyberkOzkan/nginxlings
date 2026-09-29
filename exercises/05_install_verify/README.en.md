# 05 – Install & Verify from Official Repo

## Goal

In this exercise you will:

- Install NGINX **from the official OS package repository** (apt) inside a container.
- Build an image using the provided `Dockerfile`.
- Ensure that NGINX is correctly installed and can be verified with:
  - `nginx -v`
  - `nginx -V`
- Make sure the container can start NGINX and serve HTTP on port 80.

---

## Background – official package repository

On Debian/Ubuntu-based systems, NGINX is usually installed from the official
distribution repositories, for example:

    apt-get update
    apt-get install -y nginx

This provides:

- A packaged NGINX binary (`/usr/sbin/nginx` or similar),
- Default configuration files under `/etc/nginx/`,
- System integration (logs, default configs, etc.).

"Official repository" here means the Debian package archive that the base image
already uses. nginx.org also publishes its own package repository with newer
NGINX versions; adding it needs an extra apt source and signing key, and this
exercise does not use it. The check accepts any `nginx` package that `dpkg`
reports as installed.

In this exercise, you will simulate that installation **inside a container**
defined by the `Dockerfile` in this directory.

---

## What you do in this exercise

You work in:

    05_install_verify/

The key file is:

    Dockerfile

Your tasks:

1. Open the `Dockerfile` in this directory.

2. Under the `TODO` comment, add one or more `RUN` commands that:

    - Update the package index,
    - Install `nginx` from the official repository using `apt-get`,
    - Optionally clean the package cache to keep the image smaller.

   For example, the final shape of the block should conceptually:

    - Use `apt-get update`,
    - Install `nginx` with `DEBIAN_FRONTEND=noninteractive apt-get install -y nginx`,
    - Remove `/var/lib/apt/lists/*` when you are done.

3. Make sure the installed NGINX is available in `PATH` so that commands like:

       nginx -v
       nginx -V

   work inside a container created from this image.

4. Ensure that the image can start NGINX in a way that allows HTTP traffic
   on port 80. The provided `CMD` line is prepared for that; you only need
   to make sure NGINX is correctly installed.

---

## Success criteria

This exercise is considered successful when:

- The image defined by this `Dockerfile` builds successfully with:

      docker build -t <some-name> .

- `nginx` is installed from the OS repository inside the image
  (no missing binary, no package errors).

- Running the following inside a container based on this image works without errors:

      nginx -v
      nginx -V

- NGINX can start inside the container and listen on port 80, serving HTTP
  responses for requests to `/`.

---

## Tips

- Use `DEBIAN_FRONTEND=noninteractive` in front of `apt-get install` to avoid
  interactive prompts during installation.

- Always run `apt-get update` before the first `apt-get install` in a fresh image.

- Clean the apt cache at the end of the `RUN` chain to avoid unnecessary image size:

      rm -rf /var/lib/apt/lists/*

- If the build or NGINX commands fail, read the full error message carefully;
  it usually tells you which package or step is missing.


