#!/usr/bin/env bash
# Allow port 5173 (e.g. Vite dev server) from local network.
# Run once: ./scripts/enable-local-network-access.sh

set -e
if command -v ufw &>/dev/null; then
  sudo ufw allow 5173/tcp
  echo "Port 5173 allowed (ufw). Status:"
  sudo ufw status | grep 5173 || true
elif command -v firewall-cmd &>/dev/null; then
  sudo firewall-cmd --add-port=5173/tcp --permanent
  sudo firewall-cmd --reload
  echo "Port 5173 allowed (firewalld)."
else
  echo "No ufw or firewalld found. If you use another firewall, allow TCP port 5173."
fi
echo ""
echo "From other devices use: http://$(hostname -I | awk '{print $1}'):5173"
