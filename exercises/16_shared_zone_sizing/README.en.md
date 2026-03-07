# 16 – Shared Memory Zone Sizing & Limits

## Goal

In this exercise you will learn that the shared memory zone size is defined by configuration.

Focus:

- The `<size>` in `zone=<name>:<size>` is a real limit
- A small zone makes the system struggle under many unique keys
- Increasing the zone size makes the same scenario stable again

## Task

In `nginx.conf`, in the `limit_req_zone` line:

- Keep the zone name as `req_zone`
- Increase the zone size (e.g., 256k, 512k, 1m)

After increasing the zone size:

- Many unique keys can be stored
- No shared memory allocation errors appear
- Rate limiting still works

## Success criteria

- Zone size is at least 256k
- Configuration is valid
- Rate limiting is active (some requests can return 429)