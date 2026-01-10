# 05 – Install & Verify from Official Repo
#
# Goal:
#   - Use a Linux base image.
#   - Install NGINX from the *distribution’s official repository* (apt, dnf, yum…).
#   - Make sure `nginx -v` and `nginx -V` work inside the container.
#
# This is just a starting point. It deliberately does NOT install nginx yet.
# Your task is to:
#   - Choose a suitable base image,
#   - Install nginx from the official repo of that image,
#   - (Optionally) expose HTTP port and define a proper CMD.

FROM debian:stable-slim

# TODO:
#   - Update package index
#   - Install nginx from the official Debian repository
#   - Clean package cache
#
# Example pattern (you must adapt and complete it yourself):
#   apt-get update && \
#   apt-get install -y nginx && \
#   rm -rf /var/lib/apt/lists/*
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# TODO:
#   - Optionally expose HTTP port (e.g. 80) after nginx is installed.
#   - Optionally set a CMD such as:
#       ["nginx", "-g", "daemon off;"]
#
# For now this container only prints a message so that it builds successfully
# but `nginx` is not available yet – this is what you will fix.
CMD ["bash", "-c", "echo 'TODO: install nginx from official repo inside this image' && sleep 300"]

