# 01 – Basic Static NGINX Server

## Goal

In this exercise you will run NGINX as a simple **static web server**.

When you select this exercise in Nginxlings and press `c` or Enter, the checker will
request `http://127.0.0.1:8080/` and expect the response body to contain:

> Merhaba Nginxlings

---

## What You Need to Do

1. **Configure `nginx.conf`**

   Adjust the configuration so that:

   - There is an `http { ... }` block with at least one `server { ... }`.
   - That server listens on port **8080**:
     ```nginx
     listen 8080;
     ```
   - The `root` points to this exercise directory.
     (Nginxlings runs NGINX with this directory as the prefix.)
   - The default index file is `index.html`:
     ```nginx
     index index.html;
     ```

2. **Edit `index.html`**

   - Create a valid HTML document.
   - Make sure the `<body>` contains the text **“Merhaba Nginxlings”**.
     (The checker searches for this exact string.)

3. **Validate the exercise from Nginxlings**

   - In the Nginxlings UI, select this exercise.
   - Press `c` or Enter.
   - Behind the scenes, `check.sh` will:
     - Run `nginx -t` to validate your config,
     - Start NGINX on port 8080,
     - Use `curl` to request the page and look for the required text,
     - Stop NGINX when it is done.

---

## Success Criteria

The exercise is considered **completed** when:

- `nginx -t` succeeds without errors,
- A request to `http://127.0.0.1:8080/` returns **HTTP 200 OK**,
- The response body contains **“Merhaba Nginxlings”**.

When all conditions are met, Nginxlings marks this exercise as **completed (✓)**.

---

## Tips

- Typical pattern for running NGINX with a custom config:
  ```bash
  nginx -c /path/to/nginx.conf -p /path/to/
  ```
- In this lab you do not need to run it yourself; Nginxlings does it for you.

- Because the server listens on port 8080, you do not need root privileges.
