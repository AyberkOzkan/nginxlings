# 04 – Worker User & Permissions

## Goal

In this exercise you will practice how the `user` directive affects which Unix
account NGINX worker processes run as, and why file ownership of the web root
matters.

We focus on:

- The `user` directive in the **main** (top-level) context.
- The difference between the NGINX master process and worker processes.
- Making sure the web root is owned by the same user that NGINX runs as.

Press `c` or `Enter` for this exercise in Nginxlings; an automated check will
validate your configuration and file permissions.

---

## Background – worker user

When NGINX is started as **root** in a typical Linux system:

- The *master process* keeps running as `root`.
- The master then starts worker processes and **drops privileges** for them
  using the `user` directive:

    user nginx;
    # or
    user www-data www-data;

- The workers then handle client traffic as that less-privileged user.

Some notes:

- The `user` directive is only valid in the **main** context (top level of
  `nginx.conf`), not inside `http`, `server` or `location`.
- Typical accounts are `nginx`, `www-data` or a custom account created for
  your web service.
- File permissions of the web root must allow that user to read (and sometimes
  write) the files it serves.

In unprivileged environments (when you start NGINX as a normal user), the
`user` directive may be ignored by NGINX, but the configuration syntax is the
same and still worth practicing.

---

## What you do in this exercise

You work only inside this directory:

    04_worker_user/

The idea is:

1. Configure NGINX so that:
   - There is a `user` directive in the **main** context that uses the same
     Unix user as the one running Nginxlings.
   - There is an HTTP server that serves files from an `html/` directory
     inside this folder.
2. Make sure the `html/` directory and its `index.html` file are **owned by
   the same user**.

The checker will determine the current Unix user with:

    id -un
    id -gn

and expect your `user` directive to match.

---

## What you need to prepare

In the `04_worker_user` directory:

1. **Main config file**

    Create or edit `nginx.conf` in this directory. It should have, at minimum:

    - A `user` directive in the **main** context, using your current user
      (the same user you see in `whoami`):

        user youruser;
        # or with a group:
        user youruser yourgroup;

      The checker will accept:

      - `user <current_user>;`
      - `user <current_user> <current_group>;`

    - A basic `events` block:

        events {
            worker_connections 1024;
        }

    - An `http` block with a simple server, for example:

        http {
            server {
                listen 8082;
                server_name localhost;

                # Web root inside this exercise folder
                root html;
                index index.html;

                location / {
                }
            }
        }

    You are free to add logging, gzip, etc., as long as the syntax remains
    valid and `nginx -t` passes.

2. **Web root directory and file**

    Create an `html/` directory in this folder and an `index.html` file inside
    it, for example:

        html/
          index.html

    The content of `index.html` is up to you; the checker only cares that it
    exists and that the ownership is correct.

3. **File ownership**

    Make sure that:

    - The `html/` directory is owned by the same user as the one running this
      exercise.
    - The `html/index.html` file is also owned by that user.

    You can inspect ownership with:

        ls -ld html
        ls -l html/index.html

    And, if needed, adjust it with `chown`, for example:

        chown youruser:yourgroup html html/index.html

    (In some environments you may need `sudo` for this.)

---

## What the checker validates

When you press `c` or `Enter` in Nginxlings, the script `check.sh` in this
directory will run and validate:

1. **Configuration syntax**

    - `nginx.conf` exists in this directory.
    - `nginx -t -c nginx.conf -p .` succeeds (syntax and basic layout are OK).

2. **`user` directive**

    - In `nginx.conf` there is a `user` directive in the main context that
      matches the current Unix user (and optionally its primary group).
    - Accepted forms:

        user <current_user>;
        user <current_user> <current_group>;

3. **Web root and ownership**

    - There is an `html/` directory in this folder.
    - There is an `html/index.html` file.
    - Both `html/` and `html/index.html` are owned by the current user.

If all these checks pass, the exercise will be marked as completed (✓) in the
Nginxlings TUI.

---

## Tips

- Remember that the `user` directive belongs at the **top level** of
  `nginx.conf`, not inside `http {}`.
- If the checker complains about ownership, verify with `ls -l` who owns the
  files and use `chown` if necessary.
- On systems where you start NGINX as a normal user, worker processes may
  already run as that user. The purpose here is to get used to the `user`
  directive and the relationship between worker user and file permissions.

