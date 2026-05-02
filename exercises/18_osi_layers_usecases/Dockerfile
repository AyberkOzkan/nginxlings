FROM alpine:3.20

RUN apk add --no-cache nginx nginx-mod-stream python3 socat

COPY nginx.conf /etc/nginx/nginx.conf
COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 8080 9000

CMD ["/start.sh"]