# 17 – Verify NGINX User & Processes

## Goal

In this exercise you will learn the NGINX process model, set the account that
the worker processes run as, and verify the result from the process list
instead of trusting the configuration.

---

## Background – master and workers

- NGINX starts one **master process** and one or more **worker processes**. The
  master reads the configuration, opens the listening sockets and log files, and
  starts, reloads and stops the workers. The workers handle the client
  connections.
- The master keeps the account that started NGINX (root inside this container).
  The `user` directive, in the **main** context, sets the account the workers
  switch to, so the processes that handle untrusted traffic have fewer rights.
  Its form is `user <account> [<group>];`; without a group, the group with the
  account's name is used.
- If the configuration has no `user` directive, NGINX uses the account it was
  built with; `nginx -V` shows it among the `configure arguments:`. Setting it
  explicitly makes the configuration say what actually happens.
- `user` only works when the master runs as root. Otherwise NGINX warns that the
  directive "makes sense only if the master process runs with super-user
  privileges" and ignores it.

A generic example (Debian's own `nginx.conf` uses a different account):

    user www-data;
    worker_processes auto;

Reference: [user](https://nginx.org/en/docs/ngx_core_module.html#user),
[Controlling NGINX](https://nginx.org/en/docs/control.html).

---

## Lab environment

- `Dockerfile` builds on `nginx:1.28.3-alpine` and adds `procps` so that
  `ps aux` works inside the container. The image already has an `nginx` account.
- `nginx.conf` has a `server` on port 8080 that answers `process-user-ok`, and a
  `TODO` at the top where the main context is.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image, runs
`nginx -t`, starts the container (host port 18170), reads the effective
configuration with `nginx -T` inside it, then reads `ps aux` and requests `/`.

---

## Task

In `nginx.conf`, set explicitly that the worker processes run as the `nginx`
account (with or without the `nginx` group). Put the directive in the context
where it belongs.

---

## Success criteria

- `nginx -t` accepts the configuration.
- The effective configuration has an active `user` directive for `nginx`.
- `ps aux` shows `nginx: master process` owned by `root` and at least one
  `nginx: worker process` owned by `nginx`.
- `/` answers `process-user-ok`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `No active user directive for the nginx account yet`: the directive is
  missing, still a comment, or names another account.
- `NGINX rejected the configuration: "user" directive is not allowed here`: it
  was placed inside `events {}`, `http {}` or `server {}`. Which context is left?
- `getpwnam("...") failed`: the account does not exist in this image. Check the
  spelling; the image's account is `nginx`.
- `Could not find an nginx worker process running as the nginx user`: look at
  the `ps aux` output that the check prints. Which account owns the workers?

---

## Review questions

- Which account would the workers run as in this image if you removed the
  `user` directive, and where can you see why?
- Why does the master process stay root while the workers do not?
- What happens to the `user` directive when NGINX is started by a normal user,
  as in exercise 04?
- How many worker processes do you see in `ps aux`, and which directive
  controls that number?
