FROM nginx:1.28.3-alpine

COPY nginx.conf /etc/nginx/nginx.conf
COPY start.sh /start.sh
RUN chmod +x /start.sh

CMD ["/start.sh"]
