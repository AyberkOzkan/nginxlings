FROM nginx:alpine

# Copy main config and modular pieces
COPY nginx.conf /etc/nginx/nginx.conf
COPY conf.d/ /etc/nginx/conf.d/
COPY server.d/ /etc/nginx/server.d/

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
