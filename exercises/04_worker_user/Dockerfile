FROM nginx:1.28.3-alpine

# Keep this file as it is. It prepares the site the way a deployment tool
# might: a dedicated account owns the files and other accounts cannot read them.
RUN adduser -D -H -s /sbin/nologin webapp && \
    mkdir -p /srv/site && \
    printf 'worker-user-ok\n' > /srv/site/index.html && \
    chown -R webapp:webapp /srv/site && \
    chmod 0750 /srv/site && \
    chmod 0640 /srv/site/index.html

COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
