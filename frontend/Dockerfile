# ─── Build stage ──────────────────────────────────────────────────────────────
FROM node:22-alpine AS build

WORKDIR /app

# Install deps from the committed npm lockfile (repo standard is npm, not pnpm).
COPY package.json package-lock.json ./
RUN npm ci

# Build the SPA (runs type-check + vite build). API base URL is the relative
# "/api/v1", so no build-time backend URL is needed — nginx proxies it at runtime.
COPY . .
RUN npm run build

# ─── Serve stage ──────────────────────────────────────────────────────────────
FROM nginx:alpine

COPY --from=build /app/dist /usr/share/nginx/html
# nginx.conf is an envsubst template: ${API_BACKEND} is filled from the env at
# container start (e.g. http://api:8080). Nginx-internal vars like $host are left
# untouched because the official image only substitutes defined env vars.
COPY nginx.conf /etc/nginx/templates/default.conf.template

EXPOSE 80
