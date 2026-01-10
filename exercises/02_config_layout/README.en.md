# 02 – Config Layout & Default Paths

## Goal

In this exercise you will learn the typical layout of NGINX configuration files
and how to inspect the active configuration using `nginx -t` and `nginx -T`.

On a typical Linux system:

- Main configuration file: `/etc/nginx/nginx.conf`
- Additional virtual host / server definitions: `/etc/nginx/conf.d/*.conf`
  or `sites-enabled/`

In this lab we simulate the same structure **inside this directory**:

- `nginx.conf` in this folder acts as the main config file.
- Files under `conf.d/` are loaded via an `include` directive.

When you select this exercise in Nginxlings and press `c` or Enter,
a script will validate your configuration.

---

## Topic Overview

This exercise helps you practice:

- The default NGINX directory layout (similar to `/etc/nginx/`)
- Main config file: `nginx.conf`
- Extra config files: `conf.d/*.conf`
- How the `include` directive works
- Inspecting configuration with `nginx -t` and `nginx -T`

In everyday usage:

- `nginx -t` validates the syntax and basic correctness.
- `nginx -T` prints the full effective configuration with all included files.

Here you will try these commands against local config files.

---

## What You Need to Do

While working on this exercise, stay in the `02_config_layout` directory.

1. Inspect the main config file (`nginx.conf`)

   - Open `nginx.conf` in this directory.
   - It should contain at least these basic blocks:

        events { ... }
        http   { ... }

   - Inside the `http { ... }` block, make sure there is an **active** include line:

        include conf.d/*.conf;

     (No `#` in front of it. This line loads `.conf` files from `conf.d/`.)

2. Create the `conf.d` directory and a server file

   - If `conf.d/` does not exist yet, create it:

        mkdir -p conf.d

   - Create a file `conf.d/default.conf` and add a simple `server` block, for example:

        server {
            listen 8081;
            server_name localhost;

            root .;

            location / {
                return 200 "config-layout-ok\n";
            }
        }

   - The idea:
     - The `server` block lives in `conf.d/default.conf`, not in the main file.
     - The main file (`nginx.conf`) holds global settings and the
       `include conf.d/*.conf;` line.

3. Test the configuration (optional, for learning)

   From this directory you can run:

        nginx -t -c nginx.conf -p .

   - `-c nginx.conf` → tells NGINX which config file to use.
   - `-p .`          → uses this folder as the NGINX prefix (logs, pid, etc.).

   To inspect the full effective configuration:

        nginx -T -c nginx.conf -p .

   In the output you should see the `include conf.d/*.conf;` line
   and the contents of `conf.d/default.conf` inlined.

4. Verify the exercise via Nginxlings

   - Select this exercise in the Nginxlings TUI.
   - Press `c` or Enter.
   - The `check.sh` script will:
     - run `nginx -t` against your config,
     - inspect `nginx -T` output and look for the `include conf.d/*.conf;` line.

   If everything is correct, the exercise will be marked as completed (✓).

---

## Success Criteria

The exercise is considered **successful** when:

- `nginx -t -c nginx.conf -p .` exits with code 0 (no errors).
- The output of `nginx -T -c nginx.conf -p .` contains the line
  `include conf.d/*.conf;`.
- There is at least one `server { ... }` block in `conf.d/default.conf`
  listening on a port (for example 8081).

---

## Tips

- On many systems the main configuration file is:

      /etc/nginx/nginx.conf

- A common directory for additional virtual hosts is:

      /etc/nginx/conf.d/

  The `nginx.conf` and `conf.d/` structure in this lab is a small copy of that layout.

- The `nginx -T` command is very handy when troubleshooting:
  it shows all configuration files in one place. Remember the capital `T`.

