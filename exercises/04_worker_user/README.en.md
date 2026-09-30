# 04 – Worker User & Permissions

## Goal

In this exercise you will make NGINX serve a site whose files only one account may
read. You will find out which account the workers run as, why that decides between
`200` and `403`, and how the `user` directive changes it.

You will practise:

- the `user` directive and the context it belongs to,
- reading owner, group and mode bits to predict who can open a file,
- finding the cause of a `403 Forbidden` in the error log.

---

## Background – which account opens the files

NGINX starts one **master process** and several **worker processes**. The master
reads the configuration and opens the listening sockets; the workers handle the
requests and open the files they serve. File access is therefore checked against
the **worker's** account, not the master's. (Exercise 17 looks at the process model
in more detail.)

The `user` directive, in the **main** context, names that account and optionally
its group:

    user www-data;
    user app appgroup;

Without a group, NGINX uses the group with the same name as the account. The account
must exist on the system (in `/etc/passwd`), otherwise `nginx -t` fails. Without a
`user` directive, NGINX uses the account it was built with.

To serve `/var/www/blog/index.html`, a worker needs **execute** (`x`) on every
directory in the path and **read** (`r`) on the file. Unix looks at the owner bits
if the process runs as the owner, otherwise at the group bits if the process is in
the file's group, otherwise at the "other" bits. A generic example:

    $ ls -ld /var/www/blog /var/www/blog/index.html
    drwxr-x---  2 blog blog 4096 Jan  1 10:00 /var/www/blog
    -rw-r-----  1 blog blog   12 Jan  1 10:00 /var/www/blog/index.html

Here the owner `blog` may enter the directory and read the file, members of the
group `blog` may do the same, and every other account can do neither.

Things worth knowing:

- When a worker may not open a file, NGINX answers `403 Forbidden` and writes the
  reason, including `(13: Permission denied)`, to the error log.
- Running the workers as `root` makes every permission problem disappear, which is
  exactly why it is not a fix: any bug in a worker would run with full privileges.
- `user` only has an effect when the master runs as root. Otherwise NGINX ignores
  it with a warning. That is why this lab runs in a container, where the master is
  root.

Reference: [user](https://nginx.org/en/docs/ngx_core_module.html#user),
[Beginner's guide](https://nginx.org/en/docs/beginners_guide.html).

---

## Lab environment

Everything runs inside one Docker container built from this directory:

- `Dockerfile` builds the image on `nginx:1.28.3-alpine`. It creates a dedicated
  account and the site **/srv/site** with one `index.html`, owned by that account and
  closed to others. **Keep the Dockerfile as it is**; the check verifies the owner
  and mode of both paths.
- `nginx.conf` is your file. It serves `/srv/site` on port 8080 and starts with a
  `user` line copied from a Debian server.

The check (`c` or Enter in Nginxlings, or `./check.sh`) builds the image
(`nginxlings/04_worker_user:check`), runs `nginx -t`, compares the owner and mode
of `/srv/site` and `/srv/site/index.html` with the Dockerfile's, publishes container
port 8080 on `http://127.0.0.1:18040/`, lists the worker processes with
`ps -o user,args` and requests `/`.

---

## Task

Edit only `nginx.conf`:

1. Make the workers run as an account that exists in this image and can read
   `/srv/site/index.html`.
2. The workers must not run as `root`.
3. Keep the server block: port 8080, `root /srv/site`.

---

## Success criteria

- The image builds and `nginx -t` accepts the configuration.
- `/srv/site` and `/srv/site/index.html` keep the owner, group and mode that the
  Dockerfile sets.
- At least one worker process runs, and none of them runs as `root`.
- `/` answers `200` with the page from `/srv/site/index.html`.

---

## If you are stuck

Work through these one at a time, re-running the check in between:

- `getpwnam("www-data") failed`: this is the starting state. `www-data` is the web
  server account on Debian and Ubuntu. Which accounts exist in this Alpine image?
- `"user" directive is not allowed here`: the directive is inside a block. Which
  context is left?
- `403 Forbidden: the workers cannot read /srv/site` and the error log shows
  `is forbidden (13: Permission denied)`: compare the worker account that the
  check prints with the owner and group of the files. The `chown` and `chmod`
  lines of the Dockerfile show them. To look yourself, build an image
  (`docker build -t ex04-debug .`) and run
  `docker run --rm ex04-debug ls -ld /srv/site /srv/site/index.html`.
- `The workers run as root`: root can read anything, so this proves nothing about
  the permissions. Choose an unprivileged account.
- `The owner or mode of /srv/site changed`: the Dockerfile was edited. Restore it
  and solve the task in `nginx.conf`.

---

## Review questions

- More than one `user` line passes this check. Find one that relies on the owner
  bits and one that relies on the group bits. Which would you choose, and why?
- Why does a directory need `x` and not only `r`? What would a worker in the
  file's group get if `/srv/site` had mode `0740`?
- What happens to the `user` directive if the container is started with
  `docker run --user 101 ...`? What else fails in that case?
- Without touching `nginx.conf` or the file modes, how could a deployment give the
  default `nginx` account read access to `/srv/site`?
