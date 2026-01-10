# 03 – Context Hierarchy (main / http / server / location)

## Goal

In this exercise, you will practice understanding the **context hierarchy** in
an NGINX configuration file and learn how to move misplaced directives into
the correct level.

We focus on the following chain:

- `main` (top level of the file)
- `http`
- `server`
- `location`

Some directives in this lab will be **deliberately placed in the wrong
context**. Your task is to use the error/hint messages from the automatic
check and place these directives into the correct context.

---

## Background – Context hierarchy

In an NGINX configuration, the context defines where a directive is valid and
what scope it affects.

- **main context**  
  Top level of the file. Typically used for:

    user ...
    worker_processes ...
    error_log ...
    pid ...

- **http context**

    http {
        # HTTP-level settings
        sendfile on;
        keepalive_timeout 65;

        server {
            ...
        }
    }

- **server context**

    server {
        listen 8080;
        server_name localhost;
        root /var/www/html;

        location / {
            ...
        }
    }

- **location context**

    location /images/ {
        # Behavior for a specific URI prefix
    }

Some directives are only meaningful in certain contexts. For example:

- `user`, `worker_processes` → only make sense at the main level.
- `sendfile`, `keepalive_timeout` → used at the `http` level.
- `listen`, `server_name`, `root` → typically live inside a `server` block.
- URI/path-specific behavior → defined inside a `location` block.

In this exercise, some directives are intentionally placed in the wrong
context. The automatic check will generate errors or warnings to help you
identify and fix them.

---

## What are you doing in this exercise?

In this directory, the `nginx.conf` file:

- Contains a basic context hierarchy,
- But some directives have been deliberately placed in the wrong context.

Your task:

1. Run the check with `c` or Enter.
2. Look at the hints shown in the status line (and extra output if provided):
   - Messages like “directive X is not allowed here”,
   - These point to directives that are in the wrong context.
3. Adjust the block structure and directive locations in `nginx.conf` so that
   the context hierarchy is correct and consistent.

The important points:

- The `http` block must live at the **main** level.
- `server` blocks must live inside the `http` block.
- `location` blocks must live inside a `server` block.
- Directives that do not belong to a given context should be moved down or up
  to the level where they logically belong.

---

## Success criteria

This exercise is considered **successful** when:

- The configuration is syntactically valid.
- The context hierarchy is correct.
- Directives that were placed in the wrong context are moved to a valid level.
- `server` and `location` blocks are inside the correct parent contexts.
- Context-related error messages (for example, “is not allowed here”) no longer appear.
- In the Nginxlings exercise list, this step is marked as completed (✓).

---

## Tips

- Error/warning messages usually mention which directive on which line is in
  the wrong context.
- If you are unsure where a directive belongs:
  - Think about the scope: does it affect all of NGINX, all HTTP traffic,
    a single virtual host, or only a specific path?
  - As a rough rule:
    - Entire NGINX instance → `main`
    - HTTP layer → `http`
    - Specific host/port/virtual server → `server`
    - URI/path-specific behavior → `location`
- If you see the same type of directive (for example `listen`) used in
  different contexts, one of those places is probably wrong.

