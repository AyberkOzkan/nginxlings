# 02 – Config Layout & Default Paths

## Goal

In this exercise you will learn how an NGINX configuration is usually split
between a main file and extra files, how to find out which files NGINX really
loaded, and how to test a configuration with `nginx -t` and `nginx -T`.

---

## Background – where the configuration lives

- The **main file** (`/etc/nginx/nginx.conf` on most Linux systems) holds the
  main context (`user`, `worker_processes`, `error_log`, `pid`), the `events {}`
  block and the `http {}` block with settings shared by every site.
- **Server definitions** usually live in separate files that the main file loads
  with `include` inside `http {}`. Two common layouts:
  - `/etc/nginx/conf.d/*.conf`: every matching file is loaded (nginx.org
    packages and the official Docker image).
  - `sites-available/` + `sites-enabled/`: all sites are kept in the first
    directory, and only the ones linked into the second are loaded (Debian and
    Ubuntu packages).
- `nginx -V` shows where an installation expects its main file
  (`--conf-path=`). `-c <file>` chooses another file and `-p <dir>` sets the
  prefix that relative paths such as `logs/error.log` are resolved against. A
  relative `include` is resolved against the directory of the main file.
- `nginx -t` tests the configuration. `nginx -T` tests it and prints everything
  NGINX loaded, with a `# configuration file ...:` header before each file.

A generic example (different directories and names from this lab):

    http {
        include /etc/nginx/vhosts/*.conf;
    }

    # /etc/nginx/vhosts/blog.conf
    server {
        listen 8000;
        server_name blog.local;
        ...
    }

Reference: [Beginner's Guide](https://nginx.org/en/docs/beginners_guide.html),
[Command-line parameters](https://nginx.org/en/docs/switches.html).

---

## Lab environment

This exercise runs NGINX directly on your machine, not in Docker, and imitates
the `/etc/nginx` layout inside this directory:

- `nginx.conf` is the main file. Its `http {}` block has a `TODO` where the
  extra files should be loaded.
- `conf.d/default.conf` is the extra file. It only contains a `TODO` comment.

You can test your work yourself from this directory:

    nginx -t -c nginx.conf -p .
    nginx -T -c nginx.conf -p .

The check (`c` or Enter in Nginxlings, or `./check.sh`) runs `nginx -t` with this
directory as the prefix, reads `nginx -T` (comment lines are ignored), confirms
that the include is active and that `conf.d/default.conf` was loaded with an
active `server` block, reads that block's `listen` port, starts NGINX, and
requests `http://127.0.0.1:<port>/`.

---

## Task

1. In `nginx.conf`, inside `http {}`, load every `.conf` file from the `conf.d/`
   directory.
2. In `conf.d/default.conf`, define a `server` block that listens on a free port
   of your choice (for example 8081), uses `localhost` as its server name, and
   answers every request with `config-layout-ok` followed by a newline.

---

## Success criteria

- `nginx -t` accepts the configuration.
- `nginx -T` shows an active include for `conf.d/*.conf` and the contents of
  `conf.d/default.conf` with an active `server` block and a `listen` port.
- A request to that port returns HTTP 200 with `config-layout-ok`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `nginx.conf does not load the .conf files in conf.d/ yet`: the include is
  missing, still a comment, or points somewhere else.
- `NGINX rejected the configuration: "server" directive is not allowed here in
  .../conf.d/default.conf`: the include is outside `http {}`. Look at the
  include line, not at `default.conf`.
- `No active 'server { ... }' block found in conf.d/default.conf`: the file is
  not loaded, or its `server` block is still a comment.
- `No active 'listen <port>;' directive found`: the `server` block needs a port.
- `Port ... is already in use by ...`: another program uses that port. Pick a
  different free port.
- `The response ... does not contain 'config-layout-ok'`: look at what
  `location /` answers.

---

## Review questions

- Where is the main configuration file of the NGINX installed on your machine,
  and how can NGINX itself tell you?
- What is the difference between the `conf.d/*.conf` layout and the
  `sites-available/` + `sites-enabled/` layout? Which one does your machine use?
- In this lab, what do `-c` and `-p` change, and which file does
  `logs/error.log` end up in?
- In the `nginx -T` output, how can you tell which files were included?
