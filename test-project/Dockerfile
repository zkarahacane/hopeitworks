FROM node:20-alpine AS backend

WORKDIR /app/backend
COPY backend/package.json backend/package-lock.json* ./
RUN npm install --production
COPY backend/ .

FROM nginx:alpine AS frontend

COPY frontend/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf

FROM node:20-alpine

RUN apk add --no-cache nginx

# Copy backend
WORKDIR /app/backend
COPY --from=backend /app/backend .

# Copy frontend + nginx config
COPY --from=frontend /usr/share/nginx/html/ /usr/share/nginx/html/
COPY --from=frontend /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 80 3000

CMD ["/entrypoint.sh"]
