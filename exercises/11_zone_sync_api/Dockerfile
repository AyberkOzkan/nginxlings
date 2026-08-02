FROM nginx:1.28.3-alpine

# Tools for validation inside the container (fast, small)
RUN apk add --no-cache curl jq

# Copy exercise config and JSON payload
COPY nginx.conf /etc/nginx/nginx.conf
COPY api/zone_sync.json /usr/share/nginx/html/api/zone_sync.json

EXPOSE 8080

# Use the standard nginx foreground command
CMD ["nginx", "-g", "daemon off;"]
