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
# Add wget for healthcheck
RUN apk add --no-cache wget
RUN rm -rf /usr/share/nginx/html/*
COPY --from=builder /app/dist /usr/share/nginx/html
# Custom nginx config for SPA routing
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]