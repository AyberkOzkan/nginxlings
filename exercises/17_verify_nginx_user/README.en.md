# 17 – Verify NGINX User & Processes

## Goal

In this exercise you will learn the NGINX process model and what the `user` directive actually controls.

Focus:

- The `user` directive defines the worker process user
- The difference between master and worker processes
- Interpreting process output correctly

## Task

In `nginx.conf`:

- Add `user nginx;` in the main context

With this change:

- The master process typically starts as root
- Worker processes run as the `nginx` user

## Success criteria

- Configuration is valid
- Effective config contains `user nginx;`
- Process list shows:
  - `nginx: master process` (root)
  - `nginx: worker process` (nginx)