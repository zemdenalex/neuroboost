#!/bin/bash
# One-time setup for dev environment on the server
# Run as root on 62.76.228.106
set -euo pipefail

DEV_DIR="/root/neuroboost-dev"
REPO="https://github.com/$(cd /root/neuroboost && git remote get-url origin | sed 's|.*github.com[:/]||; s|\.git$||')"

echo "=== Setting up NeuroBoost dev environment ==="

# 1. Clone repo for dev
if [ -d "$DEV_DIR" ]; then
  echo "Dev directory exists, pulling latest..."
  cd "$DEV_DIR" && git fetch origin && git checkout develop && git pull origin develop
else
  echo "Cloning repo..."
  git clone "$REPO" "$DEV_DIR"
  cd "$DEV_DIR" && git checkout develop
fi

# 2. Copy env from production, adjust for dev
if [ ! -f "$DEV_DIR/.env" ]; then
  cp /root/neuroboost/.env "$DEV_DIR/.env"
  # Adjust env for dev
  sed -i 's/POSTGRES_DB=neuroboost$/POSTGRES_DB=neuroboost_dev/' "$DEV_DIR/.env"
  sed -i 's|FRONTEND_URL=https://neuroboost.website|FRONTEND_URL=https://dev.neuroboost.website|' "$DEV_DIR/.env"
  sed -i 's/NODE_ENV=production/NODE_ENV=development/' "$DEV_DIR/.env"
  echo "Created .env for dev (review it: $DEV_DIR/.env)"
fi

# 3. Setup nginx for dev subdomain
cat > /etc/nginx/sites-available/dev.neuroboost.website << 'NGINX'
server {
    listen 80;
    server_name dev.neuroboost.website;

    location /api/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://127.0.0.1:5174;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
NGINX

# Enable site
ln -sf /etc/nginx/sites-available/dev.neuroboost.website /etc/nginx/sites-enabled/

# Test and reload nginx
nginx -t && systemctl reload nginx
echo "Nginx configured for dev.neuroboost.website"

# 4. Get SSL cert
echo "Getting SSL certificate..."
certbot --nginx -d dev.neuroboost.website --non-interactive --agree-tos --redirect \
  || echo "Certbot failed — run manually: certbot --nginx -d dev.neuroboost.website"

# 5. Start dev stack
echo "Starting dev stack..."
cd "$DEV_DIR"
docker-compose -f docker-compose.dev.yml build --pull
docker-compose -f docker-compose.dev.yml up -d

# Wait for health
echo "Waiting for API..."
sleep 15
curl -fsS http://127.0.0.1:8081/api/health && echo "Dev API healthy!" || echo "API not ready yet, check logs"

echo ""
echo "=== Dev environment ready ==="
echo "URL: https://dev.neuroboost.website"
echo "Dir: $DEV_DIR"
echo "Logs: docker-compose -f $DEV_DIR/docker-compose.dev.yml logs -f"
