# 05 – Install & Verify from Official Repo

## Goal

In this exercise you will install NGINX into a container image from the
distribution's package repository, then verify the installation the way you
would on a new server: which version it is, how it was built, which package
it came from, and whether it serves HTTP.

---

## Background – packages and images

- Debian's package tool is `apt-get`. A fresh `debian:*-slim` image ships
  **without a package index**, so no package can be found until the index has
  been downloaded.
- Nothing can answer a question during `docker build`. `apt-get` has an option
  that answers "yes" to its own confirmation prompt, and the
  `DEBIAN_FRONTEND=noninteractive` environment variable tells debconf not to
  open configuration dialogs.
- Every `RUN` instruction becomes an image layer, and Docker caches layers.
  If refreshing the index and installing a package are separate `RUN`
  instructions, a later build can reuse an old cached index with a new install
  line. That is why the two are usually chained in one `RUN` with `&&`.
- The downloaded index under `/var/lib/apt/lists/` is not needed at run time.
  Removing it **in the same `RUN`** keeps the layer small; removing it in a
  later `RUN` does not, because the earlier layer still contains it.
- Verification commands:
  - `nginx -v` prints the version.
  - `nginx -V` also prints the compiler and the `configure arguments:` NGINX was
    built with, which tells you which modules exist and whether they are static
    or dynamic.
  - `dpkg -s nginx` shows the package status, version and dependencies.

"Official repository" here means the Debian package archive that the base image
already uses. nginx.org also publishes its own package repository with newer
NGINX versions; adding it needs an extra apt source and signing key, and this
exercise does not use it. The check accepts any `nginx` package that `dpkg`
reports as installed.

Reference: [Installing NGINX Open Source](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/).

---

## Lab environment

- `Dockerfile` starts from `debian:12.15-slim`. A `TODO` comment marks where the
  installation goes; `EXPOSE 80` and the `CMD` that runs NGINX in the foreground
  are already in place.
- The Debian package ships a default site that answers on port 80, so this
  exercise needs no NGINX configuration of your own.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/05_install_verify:check`), confirms that `nginx` is on the `PATH`,
runs `nginx -v` and `nginx -V` (it looks for `configure arguments:`), confirms
that `dpkg` reports the `nginx` package as installed, then starts a container
with port 80 published on a free local port and requests `/`.

---

## Task

Under the `TODO` in the `Dockerfile`, add a single `RUN` instruction that:

1. refreshes the package index,
2. installs the `nginx` package without asking any questions,
3. removes the downloaded package index.

Leave `FROM`, `EXPOSE` and `CMD` as they are.

---

## Success criteria

- The image builds.
- `nginx` is on the `PATH`; `nginx -v` and `nginx -V` work, and `nginx -V` shows
  its `configure arguments:`.
- `dpkg -s nginx` reports `Status: install ok installed`.
- A container started from the image answers `/` on port 80 with a successful
  HTTP response.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `E: Unable to locate package nginx`: the package index is empty. What has to
  run before the install?
- `Do you want to continue? [Y/n] Abort.`: the install is waiting for an answer
  that never comes during a build.
- `debconf: unable to initialize frontend: Dialog` in the build output: only a
  warning, the install still works. The environment variable from the
  Background section silences it.
- `The image has no nginx yet.`: the build succeeded but nothing installed
  NGINX. Is your `RUN` instruction below the `TODO` and not commented out?

---

## Review questions

- Where in the `nginx -V` output can you see whether the `stream` module is
  available in Debian's package, and is it built in statically or as a dynamic
  module? What else would you need before a `stream {}` block works?
- Why are refreshing the index and installing a package normally chained in one
  `RUN` instruction?
- Which NGINX version does Debian 12 ship, and what would you change to install
  the current version from nginx.org instead?
- Where does Debian's package keep its default site, and which line in its
  `nginx.conf` loads it? How does that differ from the official `nginx` Docker
  image used in the other exercises?
