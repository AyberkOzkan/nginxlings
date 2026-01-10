FROM nginx:stable-alpine

# socat is used to implement simple TCP backends inside the container
RUN apk add --no-cache socat

COPY nginx.conf /etc/nginx/nginx.conf
COPY start.sh /start.sh
RUN chmod +x /start.sh

CMD ["/start.sh"]
