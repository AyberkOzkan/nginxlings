FROM nginx:1.28.3-alpine
RUN printf 'zone-sizing-ok\n' > /usr/share/nginx/html/index.html

COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
