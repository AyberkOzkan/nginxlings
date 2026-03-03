FROM nginx:alpine

COPY nginx.conf /etc/nginx/nginx.conf
COPY sites/site1/ /usr/share/nginx/site1/
COPY sites/site2/ /usr/share/nginx/site2/

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]