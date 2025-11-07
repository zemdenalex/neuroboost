# --- Build stage ---
FROM node:20-alpine AS build
WORKDIR /app

# Supply API URL at build time (compose ARG gets consumed here)
ARG VITE_API_URL=/api
ENV VITE_API_URL=$VITE_API_URL

# Install deps
COPY package*.json ./
# If you use pnpm, swap to corepack enable && pnpm install --frozen-lockfile
RUN npm ci

# Build
COPY . .
# Ensure Vite picks the URL: write an env file for production build
RUN printf "VITE_API_URL=%s\n" "$VITE_API_URL" > .env.production && npm run build

# --- Runtime stage (static) ---
FROM nginx:stable-alpine
# Harden a bit; serve SPA; expose /healthz for container healthcheck
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
