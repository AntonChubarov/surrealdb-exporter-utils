#!/bin/bash

echo "╔════════════════════════════════════════════════════════╗"
echo "║     SurrealDB Cluster Health Check                    ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

# SurrealDB Nodes
echo "━━━ SurrealDB Nodes ━━━"
for ip in 161 182 203; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://192.168.1.$ip:8000/health)
  if [ "$STATUS" = "200" ]; then
    echo "  ✓ 192.168.1.$ip:8000 - UP"
  else
    echo "  ✗ 192.168.1.$ip:8000 - DOWN"
  fi
done
echo ""

# TiKV Stores
echo "━━━ TiKV Stores ━━━"
curl -s http://192.168.1.161:2379/pd/api/v1/stores | \
  jq -r '.stores[] | "  \(.store.address): \(.store.state_name)"'
echo ""

# Load Balancer
echo "━━━ Load Balancer ━━━"
LB_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://192.168.1.161/health)
if [ "$LB_STATUS" = "200" ]; then
  echo "  ✓ Nginx LB - UP"
else
  echo "  ✗ Nginx LB - DOWN"
fi
