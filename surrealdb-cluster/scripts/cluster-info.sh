#!/bin/bash

# SurrealDB Cluster Information Script
# Показує детальну інформацію про стан кластера

# Кольори
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# IP адреси нод
NODE1_IP="192.168.1.161"
NODE2_IP="192.168.1.182"
NODE3_IP="192.168.1.203"

# Функція для форматування розміру
format_size() {
    local size=$1
    echo "$size"
}

# Функція для перевірки доступності
check_endpoint() {
    local url=$1
    local timeout=2

    if curl -s -o /dev/null -m $timeout "$url"; then
        return 0
    else
        return 1
    fi
}

# Заголовок
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         SurrealDB Cluster Information & Status Report             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════════╝${NC}"
echo ""
date
echo ""

# ══════════════════════════════════════════════════════════════════════
# 1. Placement Driver (PD) Cluster Information
# ══════════════════════════════════════════════════════════════════════

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}1. Placement Driver (PD) Cluster${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

PD_MEMBERS=$(curl -s http://$NODE1_IP:2379/pd/api/v1/members 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$PD_MEMBERS" ]; then
    LEADER=$(echo "$PD_MEMBERS" | jq -r '.leader.name' 2>/dev/null)
    LEADER_URL=$(echo "$PD_MEMBERS" | jq -r '.leader.client_urls[0]' 2>/dev/null)

    echo -e "  ${GREEN}Leader:${NC} $LEADER ($LEADER_URL)"
    echo ""
    echo -e "  ${YELLOW}Members:${NC}"

    echo "$PD_MEMBERS" | jq -r '.members[] |
        "    • \(.name)
      Member ID: \(.member_id)
      Client URLs: \(.client_urls | join(", "))
      Peer URLs: \(.peer_urls | join(", "))"' 2>/dev/null

    echo ""
    echo -e "  ${YELLOW}Health Status:${NC}"

    for ip in $NODE1_IP $NODE2_IP $NODE3_IP; do
        PD_HEALTH=$(curl -s http://$ip:2379/pd/api/v1/health 2>/dev/null)
        if [ $? -eq 0 ]; then
            echo "$PD_HEALTH" | jq -r '.[] | "    ✓ \(.name): \(.health)"' 2>/dev/null
        fi
    done
else
    echo -e "  ${RED}✗ PD Cluster not available${NC}"
fi

echo ""

# ══════════════════════════════════════════════════════════════════════
# 2. TiKV Storage Cluster
# ══════════════════════════════════════════════════════════════════════

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}2. TiKV Storage Cluster${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

TIKV_STORES=$(curl -s http://$NODE1_IP:2379/pd/api/v1/stores 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$TIKV_STORES" ]; then
    STORE_COUNT=$(echo "$TIKV_STORES" | jq '.count' 2>/dev/null)
    echo -e "  ${YELLOW}Total Stores:${NC} $STORE_COUNT"
    echo ""

    echo "$TIKV_STORES" | jq -r '.stores[] |
        "  ${GREEN}Store ID:${NC} \(.store.id)
    Address: \(.store.address)
    State: \(.store.state_name)
    Version: \(.store.version)
    Uptime: \(.status.uptime)

    Storage:
      • Capacity: \(.status.capacity)
      • Available: \(.status.available)
      • Used: \(.status.used_size)

    Regions:
      • Region Count: \(.status.region_count)
      • Leader Count: \(.status.leader_count)
      • Leader Size: \(.status.leader_size)

    Performance:
      • Slow Score: \(.status.slow_score)
    "' 2>/dev/null | sed "s/\${GREEN}/${GREEN}/g; s/\${NC}/${NC}/g; s/\${YELLOW}/${YELLOW}/g"

    echo ""

    # Регіони та репліки
    echo -e "  ${YELLOW}Region Distribution:${NC}"
    REGIONS=$(curl -s http://$NODE1_IP:2379/pd/api/v1/stats/region 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$REGIONS" ]; then
        echo "$REGIONS" | jq -r '
            "    Total Regions: \(.count)
    Empty Regions: \(.empty_count)
    Storage Size: \(.storage_size)
    Storage Keys: \(.storage_keys)"' 2>/dev/null
    fi
else
    echo -e "  ${RED}✗ TiKV Cluster not available${NC}"
fi

echo ""

# ══════════════════════════════════════════════════════════════════════
# 3. SurrealDB Nodes
# ══════════════════════════════════════════════════════════════════════

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}3. SurrealDB Nodes${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

SURREAL_COUNT=0
SURREAL_UP=0

for i in 1 2 3; do
    if [ $i -eq 1 ]; then IP=$NODE1_IP; fi
    if [ $i -eq 2 ]; then IP=$NODE2_IP; fi
    if [ $i -eq 3 ]; then IP=$NODE3_IP; fi

    echo -e "  ${YELLOW}Node $i ($IP):${NC}"

    # Health check
    HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://$IP:8000/health 2>/dev/null)

    if [ "$HEALTH_CODE" = "200" ]; then
        echo -e "    Status: ${GREEN}✓ UP${NC}"
        SURREAL_UP=$((SURREAL_UP + 1))

        # Version
        VERSION=$(curl -s http://$IP:8000/version 2>/dev/null)
        if [ -n "$VERSION" ]; then
            echo -e "    Version: $VERSION"
        fi

        # Status endpoint
        STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://$IP:8000/status 2>/dev/null)
        if [ "$STATUS_CODE" = "200" ]; then
            echo -e "    Server: ${GREEN}Running${NC}"
        fi
    else
        echo -e "    Status: ${RED}✗ DOWN${NC} (HTTP $HEALTH_CODE)"
    fi

    SURREAL_COUNT=$((SURREAL_COUNT + 1))
    echo ""
done

echo -e "  ${YELLOW}Summary:${NC} $SURREAL_UP/$SURREAL_COUNT nodes operational"
echo ""

# ══════════════════════════════════════════════════════════════════════
# 4. Nginx Load Balancer
# ══════════════════════════════════════════════════════════════════════

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}4. Nginx Load Balancer${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

LB_HEALTH=$(curl -s -o /dev/null -w "%{http_code}" http://$NODE1_IP/health 2>/dev/null)

if [ "$LB_HEALTH" = "200" ]; then
    echo -e "  ${GREEN}✓ Load Balancer is UP${NC}"
    echo -e "  Endpoint: http://$NODE1_IP"
    echo ""

    # Nginx status
    NGINX_STATUS=$(curl -s http://$NODE1_IP/nginx_status 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$NGINX_STATUS" ]; then
        echo -e "  ${YELLOW}Nginx Statistics:${NC}"
        echo "$NGINX_STATUS" | while IFS= read -r line; do
            echo "    $line"
        done
        echo ""
    fi

    # Request distribution (якщо логи доступні)
    LOG_FILE="$HOME/surrealdb-cluster/node-1/logs/access.log"
    if [ -f "$LOG_FILE" ]; then
        echo -e "  ${YELLOW}Request Distribution (last 100 requests):${NC}"
        DISTRIBUTION=$(tail -100 "$LOG_FILE" 2>/dev/null | grep -oP 'upstream: \K[^:]+' | sort | uniq -c | sort -rn)
        if [ -n "$DISTRIBUTION" ]; then
            echo "$DISTRIBUTION" | while read count ip; do
                echo "    $ip -> $count requests"
            done
        else
            echo "    No requests logged yet"
        fi
        echo ""

        # Last 5 requests
        echo -e "  ${YELLOW}Recent Requests (last 5):${NC}"
        tail -5 "$LOG_FILE" 2>/dev/null | while IFS= read -r line; do
            echo "    $(echo $line | cut -c1-100)..."
        done
    fi
else
    echo -e "  ${RED}✗ Load Balancer is DOWN${NC}"
fi

echo ""

# ══════════════════════════════════════════════════════════════════════
# 5. Network Connectivity
# ══════════════════════════════════════════════════════════════════════

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}5. Network Connectivity${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

echo -e "  ${YELLOW}Node Connectivity:${NC}"
for i in 1 2 3; do
    if [ $i -eq 1 ]; then IP=$NODE1_IP; NAME="Node 1"; fi
    if [ $i -eq 2 ]; then IP=$NODE2_IP; NAME="Node 2"; fi
    if [ $i -eq 3 ]; then IP=$NODE3_IP; NAME="Node 3"; fi

    if ping -c 1 -W 1 $IP &> /dev/null; then
        echo -e "    ${GREEN}✓${NC} $NAME ($IP) - reachable"
    else
        echo -e "    ${RED}✗${NC} $NAME ($IP) - unreachable"
    fi
done

echo ""

# ══════════════════════════════════════════════════════════════════════
# 6. Docker Containers Status
# ══════════════════════════════════════════════════════════════════════

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}6. Docker Containers (Local Node)${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Визначаємо поточну ноду
CURRENT_IP=$(hostname -I | awk '{print $1}')
echo -e "  ${YELLOW}Current Node IP:${NC} $CURRENT_IP"
echo ""

# Показуємо контейнери
CONTAINERS=$(docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null | grep -E "pd[0-9]|tikv[0-9]|surrealdb[0-9]|nginx")

if [ -n "$CONTAINERS" ]; then
    echo "$CONTAINERS" | while IFS= read -r line; do
        if echo "$line" | grep -q "NAMES"; then
            echo -e "  ${YELLOW}$line${NC}"
        else
            # Визначаємо статус по кольору
            if echo "$line" | grep -q "Up"; then
                echo -e "  ${GREEN}$line${NC}"
            else
                echo -e "  ${RED}$line${NC}"
            fi
        fi
    done
else
    echo -e "  ${YELLOW}No cluster containers running on this node${NC}"
fi

echo ""

# ══════════════════════════════════════════════════════════════════════
# 7. Cluster Summary & Health Score
# ══════════════════════════════════════════════════════════════════════

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}7. Cluster Health Summary${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Calculate health score
HEALTH_SCORE=0
MAX_SCORE=100

# PD Cluster (30 points)
if [ -n "$PD_MEMBERS" ]; then
    PD_MEMBER_COUNT=$(echo "$PD_MEMBERS" | jq '.members | length' 2>/dev/null)
    if [ "$PD_MEMBER_COUNT" = "3" ]; then
        HEALTH_SCORE=$((HEALTH_SCORE + 30))
        PD_STATUS="${GREEN}Healthy (3/3 members)${NC}"
    else
        HEALTH_SCORE=$((HEALTH_SCORE + 15))
        PD_STATUS="${YELLOW}Degraded ($PD_MEMBER_COUNT/3 members)${NC}"
    fi
else
    PD_STATUS="${RED}Down${NC}"
fi

# TiKV Cluster (30 points)
if [ -n "$TIKV_STORES" ]; then
    TIKV_UP_COUNT=$(echo "$TIKV_STORES" | jq '[.stores[] | select(.store.state_name == "Up")] | length' 2>/dev/null)
    if [ "$TIKV_UP_COUNT" = "3" ]; then
        HEALTH_SCORE=$((HEALTH_SCORE + 30))
        TIKV_STATUS="${GREEN}Healthy (3/3 stores Up)${NC}"
    elif [ "$TIKV_UP_COUNT" -gt "0" ]; then
        HEALTH_SCORE=$((HEALTH_SCORE + 15))
        TIKV_STATUS="${YELLOW}Degraded ($TIKV_UP_COUNT/3 stores Up)${NC}"
    else
        TIKV_STATUS="${RED}Down${NC}"
    fi
else
    TIKV_STATUS="${RED}Down${NC}"
fi

# SurrealDB Nodes (30 points)
if [ "$SURREAL_UP" = "3" ]; then
    HEALTH_SCORE=$((HEALTH_SCORE + 30))
    SURREAL_STATUS="${GREEN}Healthy (3/3 nodes)${NC}"
elif [ "$SURREAL_UP" -gt "0" ]; then
    HEALTH_SCORE=$((HEALTH_SCORE + 15))
    SURREAL_STATUS="${YELLOW}Degraded ($SURREAL_UP/3 nodes)${NC}"
else
    SURREAL_STATUS="${RED}Down${NC}"
fi

# Load Balancer (10 points)
if [ "$LB_HEALTH" = "200" ]; then
    HEALTH_SCORE=$((HEALTH_SCORE + 10))
    LB_STATUS="${GREEN}Operational${NC}"
else
    LB_STATUS="${RED}Down${NC}"
fi

# Display summary
echo -e "  ${YELLOW}Component Status:${NC}"
echo -e "    PD Cluster:      $PD_STATUS"
echo -e "    TiKV Cluster:    $TIKV_STATUS"
echo -e "    SurrealDB:       $SURREAL_STATUS"
echo -e "    Load Balancer:   $LB_STATUS"
echo ""

# Overall health
echo -e "  ${YELLOW}Overall Health Score:${NC} $HEALTH_SCORE/$MAX_SCORE"
echo ""

if [ "$HEALTH_SCORE" -ge 90 ]; then
    echo -e "  ${GREEN}╔════════════════════════════════════════╗${NC}"
    echo -e "  ${GREEN}║  ✓ Cluster is HEALTHY                 ║${NC}"
    echo -e "  ${GREEN}╚════════════════════════════════════════╝${NC}"
elif [ "$HEALTH_SCORE" -ge 50 ]; then
    echo -e "  ${YELLOW}╔════════════════════════════════════════╗${NC}"
    echo -e "  ${YELLOW}║  ⚠ Cluster is DEGRADED                ║${NC}"
    echo -e "  ${YELLOW}║    Some components are unavailable    ║${NC}"
    echo -e "  ${YELLOW}╚════════════════════════════════════════╝${NC}"
else
    echo -e "  ${RED}╔════════════════════════════════════════╗${NC}"
    echo -e "  ${RED}║  ✗ Cluster is UNHEALTHY               ║${NC}"
    echo -e "  ${RED}║    Critical components are down       ║${NC}"
    echo -e "  ${RED}╚════════════════════════════════════════╝${NC}"
fi

echo ""

# ══════════════════════════════════════════════════════════════════════
# Footer
# ══════════════════════════════════════════════════════════════════════

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Report Complete                                                   ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Quick access commands
echo -e "${YELLOW}Quick Access:${NC}"
echo "  • Cluster Status:    make cluster-status"
echo "  • Health Check:      make health"
echo "  • View Logs:         make logs"
echo "  • Connect CLI:       surreal sql -e http://$NODE1_IP -u root -p SecurePassword123! --ns test --db test"
echo "  • Surrealist GUI:    https://surrealist.app (endpoint: http://$NODE1_IP)"
echo ""
