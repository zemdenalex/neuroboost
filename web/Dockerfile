# neuroboost-web/Dockerfile

FROM node:22-alpine AS builder
WORKDIR /app
RUN npm i -g pnpm@9

COPY package.json ./
# COPY pnpm-lock.yaml ./   # if you have one

RUN /bin/sh -lc 'if [ -f pnpm-lock.yaml ]; then pnpm install --frozen-lockfile; else pnpm install; fi'

COPY index.html ./
COPY tsconfig.json vite.config.ts tailwind.config.js postcss.config.js ./
COPY .env.example ./
COPY public ./public
COPY src ./src
RUN pnpm build

FROM nginx:stable-alpine
# Add curl for healthcheck (smaller than wget)
RUN apk add --no-cache curl
RUN rm -rf /usr/share/nginx/html/*
COPY --from=builder /app/dist /usr/share/nginx/html
# Custom nginx config for SPA routing
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
# Healthcheck using curl with IPv4
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD curl -f http://127.0.0.1/ || exit 1
CMD ["nginx", "-g", "daemon off;"]