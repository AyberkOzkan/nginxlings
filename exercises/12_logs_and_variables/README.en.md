# 12 – Logs & Variables

## Goal

In this exercise you will create a custom access log format and verify that
common NGINX variables are written to the access log.

Targets:

- Define a custom `log_format` named `main`.
- Use `access_log` with that `main` format.
- The log lines must contain these variables (as values in the output):
  - `$remote_addr`
  - `$status`
  - `$request`
  - `$http_user_agent`

The verification will send a few HTTP requests and then check the access log
output for those fields.

---

## Task

Edit `nginx.conf` for this exercise and do the following:

1. Define a `log_format` named `main` inside the `http { ... }` context.
2. Configure `access_log` to use that format.
3. Ensure there is an HTTP server listening on port 8080 that returns HTTP 200
   for `/`.

The `log_format main` should include at least:

    $remote_addr
    $status
    $request
    $http_user_agent

---

## Success criteria

This exercise is successful when:

- Access logging is enabled with the `main` log format.
- After test requests, the access log contains entries that include:
  - an IP address from `$remote_addr`
  - the HTTP status code from `$status`
  - the request line from `$request`
  - a user agent string from `$http_user_agent`

---

## Tips

- `log_format` must be defined in the `http` context.
- `access_log` can be set in `http` or `server`. Either is fine for this lab.
- Use `/dev/stdout` as the access log target so the log is visible through
  container logs.
