FROM nginx:1.28.3-alpine

# Install python3 to run simple HTTP backends inside the container
RUN apk add --no-cache python3

# Copy lab nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Copy startup script
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Keep the original entrypoint from the official image.
# CMD is overridden so that, by default, the container runs /start.sh.
# When we run "docker run IMAGE nginx -t", the CMD is replaced with "nginx -t"
# and the original entrypoint will handle it correctly.
CMD ["/start.sh"]

