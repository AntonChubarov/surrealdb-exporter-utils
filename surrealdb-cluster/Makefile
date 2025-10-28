# SurrealDB Cluster Management Makefile
# Usage: make [target] NODE=1|2|3

# Визначення поточної ноди (за замовчуванням спробуємо визначити автоматично)
HOSTNAME := $(shell hostname)
LOCAL_IP := $(shell hostname -I | awk '{print $$1}')

# Мапінг IP адрес до номерів нод
ifeq ($(LOCAL_IP),192.168.1.161)
    AUTO_NODE := 1
else ifeq ($(LOCAL_IP),192.168.1.182)
    AUTO_NODE := 2
else ifeq ($(LOCAL_IP),192.168.1.203)
    AUTO_NODE := 3
else
    AUTO_NODE := 1
endif

# Використовуємо NODE з командного рядку або автоматично визначену
NODE ?= $(AUTO_NODE)

# Директорії
NODE_DIR := node-$(NODE)
COMPOSE_FILE := $(NODE_DIR)/compose.yml

# Кольори для виводу
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Перевірка чи існує compose файл для вказаної ноди
check-node:
	@if [ ! -f "$(COMPOSE_FILE)" ]; then \
		echo "$(RED)Error: $(COMPOSE_FILE) not found!$(NC)"; \
		echo "Available nodes: 1, 2, 3"; \
		exit 1; \
	fi

# ══════════════════════════════════════════════════════════
# Підготовка середовища
# ══════════════════════════════════════════════════════════

.PHONY: check-deps
check-deps: ## Перевірити всі необхідні залежності
	@echo "$(BLUE)Checking dependencies...$(NC)"
	@echo ""
	@MISSING_DEPS=0; \
	echo -n "  Docker:          "; \
	if command -v docker &> /dev/null; then \
		echo "$(GREEN)✓ $(shell docker --version)$(NC)"; \
	else \
		echo "$(RED)✗ Not found$(NC)"; \
		MISSING_DEPS=1; \
	fi; \
	echo -n "  Docker Compose:  "; \
	if docker compose version &> /dev/null; then \
		echo "$(GREEN)✓ $(shell docker compose version --short)$(NC)"; \
	else \
		echo "$(RED)✗ Not found$(NC)"; \
		MISSING_DEPS=1; \
	fi; \
	echo -n "  curl:            "; \
	if command -v curl &> /dev/null; then \
		echo "$(GREEN)✓ $(shell curl --version | head -1)$(NC)"; \
	else \
		echo "$(RED)✗ Not found$(NC)"; \
		MISSING_DEPS=1; \
	fi; \
	echo -n "  jq:              "; \
	if command -v jq &> /dev/null; then \
		echo "$(GREEN)✓ $(shell jq --version)$(NC)"; \
	else \
		echo "$(RED)✗ Not found$(NC)"; \
		MISSING_DEPS=1; \
	fi; \
	echo -n "  wget:            "; \
	if command -v wget &> /dev/null; then \
		echo "$(GREEN)✓ $(shell wget --version | head -1 | awk '{print $$3}')$(NC)"; \
	else \
		echo "$(RED)✗ Not found$(NC)"; \
		MISSING_DEPS=1; \
	fi; \
	echo -n "  make:            "; \
	if command -v make &> /dev/null; then \
		echo "$(GREEN)✓ $(shell make --version | head -1)$(NC)"; \
	else \
		echo "$(RED)✗ Not found$(NC)"; \
		MISSING_DEPS=1; \
	fi; \
	echo ""; \
	if [ $$MISSING_DEPS -eq 1 ]; then \
		echo "$(RED)Some dependencies are missing!$(NC)"; \
		echo ""; \
		echo "Install them with:"; \
		echo "  sudo apt update"; \
		echo "  sudo apt install -y docker.io docker-compose-plugin curl jq wget make"; \
		echo ""; \
		exit 1; \
	else \
		echo "$(GREEN)✓ All dependencies are installed$(NC)"; \
	fi

.PHONY: check-docker
check-docker: ## Перевірити чи Docker запущений і доступний
	@echo "$(BLUE)Checking Docker status...$(NC)"
	@if docker info &> /dev/null; then \
		echo "$(GREEN)✓ Docker is running$(NC)"; \
		echo ""; \
		docker info | grep -E "Server Version|Storage Driver|Operating System|Total Memory|CPUs"; \
	else \
		echo "$(RED)✗ Docker is not running or current user doesn't have permissions$(NC)"; \
		echo ""; \
		echo "Try:"; \
		echo "  sudo systemctl start docker"; \
		echo "  sudo usermod -aG docker $$USER"; \
		echo "  (then logout and login again)"; \
		exit 1; \
	fi

.PHONY: setup-dirs
setup-dirs: check-node ## Створити необхідні директорії
	@echo "$(GREEN)Creating directories for Node $(NODE)...$(NC)"
	@mkdir -p $(NODE_DIR)/logs
	@mkdir -p backups
	@mkdir -p scripts
	@if [ "$(NODE)" = "1" ]; then \
		mkdir -p $(NODE_DIR)/logs; \
		echo "$(GREEN)✓ Created logs directory for Nginx$(NC)"; \
	fi
	@echo "$(GREEN)✓ Directory structure created$(NC)"
	@echo ""
	@echo "Created:"
	@ls -la $(NODE_DIR)/ 2>/dev/null || true
	@if [ "$(NODE)" = "1" ]; then ls -la $(NODE_DIR)/logs/ 2>/dev/null || true; fi

.PHONY: setup-hosts
setup-hosts: ## Налаштувати /etc/hosts для резолюції імен
	@echo "$(YELLOW)Setting up /etc/hosts...$(NC)"
	@echo ""
	@echo "The following entries will be added to /etc/hosts:"
	@echo "  192.168.1.161 node1 pd1 tikv1"
	@echo "  192.168.1.182 node2 pd2 tikv2"
	@echo "  192.168.1.203 node3 pd3 tikv3"
	@echo ""
	@if grep -q "192.168.1.161 node1" /etc/hosts 2>/dev/null; then \
		echo "$(YELLOW)Entries already exist in /etc/hosts$(NC)"; \
	else \
		echo "$(YELLOW)Adding entries to /etc/hosts (requires sudo)...$(NC)"; \
		echo "192.168.1.161 node1 pd1 tikv1" | sudo tee -a /etc/hosts > /dev/null; \
		echo "192.168.1.182 node2 pd2 tikv2" | sudo tee -a /etc/hosts > /dev/null; \
		echo "192.168.1.203 node3 pd3 tikv3" | sudo tee -a /etc/hosts > /dev/null; \
		echo "$(GREEN)✓ /etc/hosts updated$(NC)"; \
	fi

.PHONY: setup-firewall
setup-firewall: ## Налаштувати firewall правила
	@echo "$(YELLOW)Setting up firewall rules...$(NC)"
	@echo ""
	@if command -v ufw &> /dev/null; then \
		echo "$(BLUE)Detected UFW firewall$(NC)"; \
		echo ""; \
		echo "Opening ports for cluster communication..."; \
		sudo ufw allow from 192.168.1.0/24 to any port 2379 proto tcp comment 'PD client'; \
		sudo ufw allow from 192.168.1.0/24 to any port 2380 proto tcp comment 'PD peer'; \
		sudo ufw allow from 192.168.1.0/24 to any port 20160 proto tcp comment 'TiKV'; \
		sudo ufw allow from 192.168.1.0/24 to any port 8000 proto tcp comment 'SurrealDB'; \
		if [ "$(NODE)" = "1" ]; then \
			echo "Opening port 80 for Load Balancer..."; \
			sudo ufw allow 80/tcp comment 'Nginx LB'; \
		fi; \
		echo ""; \
		echo "$(YELLOW)Current UFW status:$(NC)"; \
		sudo ufw status numbered | grep -E '2379|2380|20160|8000|80' || echo "No rules found yet"; \
		echo ""; \
		echo "$(GREEN)✓ Firewall rules configured$(NC)"; \
		echo ""; \
		echo "$(YELLOW)Note: UFW must be enabled. Enable with:$(NC)"; \
		echo "  sudo ufw enable"; \
	else \
		echo "$(YELLOW)UFW not found. Using iptables...$(NC)"; \
		echo ""; \
		sudo iptables -I INPUT -p tcp --dport 2379 -s 192.168.1.0/24 -j ACCEPT -m comment --comment "PD client" 2>/dev/null || true; \
		sudo iptables -I INPUT -p tcp --dport 2380 -s 192.168.1.0/24 -j ACCEPT -m comment --comment "PD peer" 2>/dev/null || true; \
		sudo iptables -I INPUT -p tcp --dport 20160 -s 192.168.1.0/24 -j ACCEPT -m comment --comment "TiKV" 2>/dev/null || true; \
		sudo iptables -I INPUT -p tcp --dport 8000 -s 192.168.1.0/24 -j ACCEPT -m comment --comment "SurrealDB" 2>/dev/null || true; \
		if [ "$(NODE)" = "1" ]; then \
			sudo iptables -I INPUT -p tcp --dport 80 -j ACCEPT -m comment --comment "Nginx LB" 2>/dev/null || true; \
		fi; \
		echo "$(GREEN)✓ iptables rules added$(NC)"; \
		echo ""; \
		echo "$(YELLOW)To persist rules, install iptables-persistent:$(NC)"; \
		echo "  sudo apt install iptables-persistent"; \
		echo "  sudo netfilter-persistent save"; \
	fi

.PHONY: check-network
check-network: ## Перевірити мережеву зв'язність з іншими нодами
	@echo "$(BLUE)Checking network connectivity...$(NC)"
	@echo ""
	@echo "$(YELLOW)Current node IP:$(NC) $(LOCAL_IP)"
	@echo ""
	@echo "$(YELLOW)Testing connectivity to cluster nodes:$(NC)"
	@for ip in 192.168.1.161 192.168.1.182 192.168.1.203; do \
		echo -n "  $$ip: "; \
		if ping -c 1 -W 2 $$ip &> /dev/null; then \
			echo "$(GREEN)✓ reachable$(NC)"; \
		else \
			echo "$(RED)✗ unreachable$(NC)"; \
		fi; \
	done
	@echo ""
	@echo "$(YELLOW)Testing required ports:$(NC)"
	@for port in 2379 2380 20160 8000; do \
		for ip in 192.168.1.161 192.168.1.182 192.168.1.203; do \
			if [ "$$ip" != "$(LOCAL_IP)" ]; then \
				echo -n "  $$ip:$$port - "; \
				if nc -z -w 2 $$ip $$port 2>/dev/null; then \
					echo "$(GREEN)✓ open$(NC)"; \
				else \
					echo "$(YELLOW)⊘ closed/filtered$(NC)"; \
				fi; \
			fi; \
		done; \
	done

.PHONY: check-resources
check-resources: ## Перевірити системні ресурси
	@echo "$(BLUE)Checking system resources...$(NC)"
	@echo ""
	@echo "$(YELLOW)Memory:$(NC)"
	@free -h | grep -E "Mem|Swap"
	@echo ""
	@TOTAL_MEM=$$(free -g | awk '/^Mem:/{print $$2}'); \
	if [ $$TOTAL_MEM -lt 4 ]; then \
		echo "$(RED)⚠ Warning: Less than 4GB RAM available. Cluster may be slow.$(NC)"; \
	else \
		echo "$(GREEN)✓ Sufficient memory available$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)Disk Space:$(NC)"
	@df -h / | grep -v Filesystem
	@echo ""
	@AVAIL_DISK=$$(df / | awk 'NR==2 {print $$4}' | sed 's/G//'); \
	if [ $$AVAIL_DISK -lt 20 ]; then \
		echo "$(RED)⚠ Warning: Less than 20GB disk space available.$(NC)"; \
	else \
		echo "$(GREEN)✓ Sufficient disk space available$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)CPU:$(NC)"
	@lscpu | grep -E "Model name|CPU\(s\):" | head -2

.PHONY: setup
setup: check-deps check-docker setup-dirs setup-hosts setup-firewall ## Повне налаштування середовища (виконати перед першим запуском)
	@echo ""
	@echo "$(GREEN)╔════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(GREEN)║  Environment Setup Complete for Node $(NODE)                    ║$(NC)"
	@echo "$(GREEN)╚════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "  1. Verify network connectivity:  make check-network"
	@echo "  2. Check system resources:       make check-resources"
	@echo "  3. Start cluster components:     make init-cluster"
	@echo ""

.PHONY: verify
verify: check-deps check-docker check-network check-resources ## Повна перевірка готовності до запуску
	@echo ""
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║  Pre-flight Verification Complete                         ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@if [ -f "$(COMPOSE_FILE)" ]; then \
		echo "$(GREEN)✓ Node $(NODE) is ready to start$(NC)"; \
		echo ""; \
		echo "To start the cluster, run:"; \
		echo "  make init-cluster"; \
	else \
		echo "$(RED)✗ Compose file not found: $(COMPOSE_FILE)$(NC)"; \
		exit 1; \
	fi

.PHONY: setup-scripts
setup-scripts: ## Зробити скрипти виконуваними
	@echo "$(GREEN)Making scripts executable...$(NC)"
	@if [ -f "scripts/health-check.sh" ]; then \
		chmod +x scripts/health-check.sh; \
		echo "$(GREEN)✓ scripts/health-check.sh$(NC)"; \
	fi
	@if [ -f "scripts/cluster-info.sh" ]; then \
		chmod +x scripts/cluster-info.sh; \
		echo "$(GREEN)✓ scripts/cluster-info.sh$(NC)"; \
	fi
	@echo "$(GREEN)✓ Scripts are ready$(NC)"

.PHONY: install-tools
install-tools: ## Встановити додаткові корисні інструменти
	@echo "$(BLUE)Installing additional tools...$(NC)"
	@echo ""
	@echo "This will install:"
	@echo "  • netcat-openbsd (for port checking)"
	@echo "  • htop (for monitoring)"
	@echo "  • net-tools (for network diagnostics)"
	@echo ""
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		sudo apt update; \
		sudo apt install -y netcat-openbsd htop net-tools; \
		echo "$(GREEN)✓ Additional tools installed$(NC)"; \
	else \
		echo "$(YELLOW)Cancelled$(NC)"; \
	fi

.PHONY: uninstall
uninstall: ## Видалити всі компоненти кластера та дані
	@echo "$(RED)╔════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(RED)║  WARNING: Complete Cluster Uninstallation                 ║$(NC)"
	@echo "$(RED)╚════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "This will:"
	@echo "  • Stop all containers"
	@echo "  • Remove all volumes and data"
	@echo "  • Remove all Docker images"
	@echo "  • Clean up directories"
	@echo ""
	@read -p "Are you absolutely sure? Type 'DELETE' to confirm: " -r CONFIRM; \
	if [ "$$CONFIRM" = "DELETE" ]; then \
		echo ""; \
		echo "$(YELLOW)Stopping and removing containers...$(NC)"; \
		docker compose -f $(COMPOSE_FILE) down -v --rmi all 2>/dev/null || true; \
		echo "$(YELLOW)Removing logs...$(NC)"; \
		rm -rf $(NODE_DIR)/logs/* 2>/dev/null || true; \
		echo "$(YELLOW)Removing backups...$(NC)"; \
		rm -rf backups/* 2>/dev/null || true; \
		echo ""; \
		echo "$(GREEN)✓ Uninstallation complete$(NC)"; \
	else \
		echo "$(YELLOW)Cancelled - nothing was deleted$(NC)"; \
	fi

.PHONY: help
help: ## Показати це повідомлення допомоги
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║  SurrealDB Cluster Management                              ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(GREEN)Current Node: $(NODE) (IP: $(LOCAL_IP))$(NC)"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@echo "  make <target> [NODE=1|2|3]"
	@echo ""
	@echo "$(CYAN)═══ Setup & Preparation ═══$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E 'setup|check-|verify|install' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(CYAN)═══ Cluster Operations ═══$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E 'start|stop|restart|init' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(CYAN)═══ Monitoring & Logs ═══$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E 'status|health|logs|cluster|info' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(CYAN)═══ Maintenance ═══$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E 'clean|pull|update|backup|uninstall' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

.PHONY: info
info: ## Показати інформацію про поточну ноду
	@echo "$(BLUE)Node Information:$(NC)"
	@echo "  Node Number: $(NODE)"
	@echo "  Local IP: $(LOCAL_IP)"
	@echo "  Compose File: $(COMPOSE_FILE)"
	@echo "  Node Directory: $(NODE_DIR)"

# ══════════════════════════════════════════════════════════
# Запуск компонентів
# ══════════════════════════════════════════════════════════

.PHONY: start-all
start-all: check-node ## Запустити всі сервіси на поточній ноді
	@echo "$(GREEN)Starting all services on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) up -d
	@echo "$(GREEN)✓ All services started$(NC)"

.PHONY: start-pd
start-pd: check-node ## Запустити тільки Placement Driver
	@echo "$(GREEN)Starting PD on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) up -d pd$(NODE)
	@sleep 5
	@echo "$(GREEN)✓ PD$(NODE) started$(NC)"

.PHONY: start-tikv
start-tikv: check-node ## Запустити тільки TiKV
	@echo "$(GREEN)Starting TiKV on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) up -d tikv$(NODE)
	@sleep 5
	@echo "$(GREEN)✓ TiKV$(NODE) started$(NC)"

.PHONY: start-surrealdb
start-surrealdb: check-node ## Запустити тільки SurrealDB
	@echo "$(GREEN)Starting SurrealDB on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) up -d surrealdb$(NODE)
	@sleep 5
	@echo "$(GREEN)✓ SurrealDB$(NODE) started$(NC)"

.PHONY: start-nginx
start-nginx: check-node ## Запустити Nginx Load Balancer (тільки Node 1)
	@if [ "$(NODE)" != "1" ]; then \
		echo "$(RED)Error: Nginx is only available on Node 1$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Starting Nginx Load Balancer...$(NC)"
	docker compose -f $(COMPOSE_FILE) up -d nginx-lb
	@echo "$(GREEN)✓ Nginx started$(NC)"

# ══════════════════════════════════════════════════════════
# Зупинка компонентів
# ══════════════════════════════════════════════════════════

.PHONY: stop-all
stop-all: check-node ## Зупинити всі сервіси на поточній ноді
	@echo "$(YELLOW)Stopping all services on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) down
	@echo "$(GREEN)✓ All services stopped$(NC)"

.PHONY: stop-pd
stop-pd: check-node ## Зупинити Placement Driver
	@echo "$(YELLOW)Stopping PD on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) stop pd$(NODE)
	@echo "$(GREEN)✓ PD$(NODE) stopped$(NC)"

.PHONY: stop-tikv
stop-tikv: check-node ## Зупинити TiKV
	@echo "$(YELLOW)Stopping TiKV on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) stop tikv$(NODE)
	@echo "$(GREEN)✓ TiKV$(NODE) stopped$(NC)"

.PHONY: stop-surrealdb
stop-surrealdb: check-node ## Зупинити SurrealDB
	@echo "$(YELLOW)Stopping SurrealDB on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) stop surrealdb$(NODE)
	@echo "$(GREEN)✓ SurrealDB$(NODE) stopped$(NC)"

.PHONY: stop-nginx
stop-nginx: check-node ## Зупинити Nginx Load Balancer
	@if [ "$(NODE)" != "1" ]; then \
		echo "$(RED)Error: Nginx is only available on Node 1$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Stopping Nginx Load Balancer...$(NC)"
	docker compose -f $(COMPOSE_FILE) stop nginx-lb
	@echo "$(GREEN)✓ Nginx stopped$(NC)"

# ══════════════════════════════════════════════════════════
# Перезапуск компонентів
# ══════════════════════════════════════════════════════════

.PHONY: restart-all
restart-all: check-node ## Перезапустити всі сервіси
	@echo "$(YELLOW)Restarting all services on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) restart
	@echo "$(GREEN)✓ All services restarted$(NC)"

.PHONY: restart-surrealdb
restart-surrealdb: check-node ## Перезапустити SurrealDB
	@echo "$(YELLOW)Restarting SurrealDB on Node $(NODE)...$(NC)"
	docker compose -f $(COMPOSE_FILE) restart surrealdb$(NODE)
	@echo "$(GREEN)✓ SurrealDB$(NODE) restarted$(NC)"

# ══════════════════════════════════════════════════════════
# Перегляд статусу та логів
# ══════════════════════════════════════════════════════════

.PHONY: status
status: check-node ## Показати статус всіх контейнерів
	@echo "$(BLUE)Status of services on Node $(NODE):$(NC)"
	@docker compose -f $(COMPOSE_FILE) ps

.PHONY: logs
logs: check-node ## Показати логи всіх сервісів
	docker compose -f $(COMPOSE_FILE) logs -f

.PHONY: logs-pd
logs-pd: check-node ## Показати логи PD
	docker compose -f $(COMPOSE_FILE) logs -f pd$(NODE)

.PHONY: logs-tikv
logs-tikv: check-node ## Показати логи TiKV
	docker compose -f $(COMPOSE_FILE) logs -f tikv$(NODE)

.PHONY: logs-surrealdb
logs-surrealdb: check-node ## Показати логи SurrealDB
	docker compose -f $(COMPOSE_FILE) logs -f surrealdb$(NODE)

.PHONY: logs-nginx
logs-nginx: check-node ## Показати логи Nginx
	@if [ "$(NODE)" != "1" ]; then \
		echo "$(RED)Error: Nginx is only available on Node 1$(NC)"; \
		exit 1; \
	fi
	docker compose -f $(COMPOSE_FILE) logs -f nginx-lb

# ══════════════════════════════════════════════════════════
# Діагностика та перевірка
# ══════════════════════════════════════════════════════════

.PHONY: health
health: check-node ## Перевірити здоров'я всіх сервісів
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║  Health Check - Node $(NODE)                                    ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Docker Containers:$(NC)"
	@docker compose -f $(COMPOSE_FILE) ps
	@echo ""
	@echo "$(YELLOW)PD Health:$(NC)"
	@curl -s http://localhost:2379/pd/api/v1/health | jq '.' 2>/dev/null || echo "  PD not available"
	@echo ""
	@echo "$(YELLOW)SurrealDB Health:$(NC)"
	@curl -s -o /dev/null -w "  HTTP Status: %{http_code}\n" http://localhost:8000/health

.PHONY: cluster-status
cluster-status: ## Показати статус всього кластера
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║  Cluster Status                                            ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)PD Cluster:$(NC)"
	@curl -s http://192.168.1.161:2379/pd/api/v1/members | jq '.' 2>/dev/null || echo "  Not available"
	@echo ""
	@echo "$(YELLOW)TiKV Stores:$(NC)"
	@curl -s http://192.168.1.161:2379/pd/api/v1/stores | jq '.stores[] | {id: .store.id, address: .store.address, state: .store.state_name}' 2>/dev/null || echo "  Not available"
	@echo ""
	@echo "$(YELLOW)SurrealDB Nodes:$(NC)"
	@for ip in 161 182 203; do \
		status=$$(curl -s -o /dev/null -w "%{http_code}" http://192.168.1.$$ip:8000/health 2>/dev/null); \
		if [ "$$status" = "200" ]; then \
			echo "  $(GREEN)✓$(NC) 192.168.1.$$ip:8000 - UP"; \
		else \
			echo "  $(RED)✗$(NC) 192.168.1.$$ip:8000 - DOWN"; \
		fi \
	done
	@echo ""
	@if [ "$(NODE)" = "1" ]; then \
		echo "$(YELLOW)Load Balancer:$(NC)"; \
		lb_status=$$(curl -s -o /dev/null -w "%{http_code}" http://192.168.1.161/health 2>/dev/null); \
		if [ "$$lb_status" = "200" ]; then \
			echo "  $(GREEN)✓$(NC) Nginx LB - UP"; \
		else \
			echo "  $(RED)✗$(NC) Nginx LB - DOWN"; \
		fi; \
	fi

# ══════════════════════════════════════════════════════════
# Очищення та видалення даних
# ══════════════════════════════════════════════════════════

.PHONY: clean
clean: check-node ## Зупинити та видалити контейнери (без видалення даних)
	@echo "$(YELLOW)Cleaning Node $(NODE) (keeping volumes)...$(NC)"
	docker compose -f $(COMPOSE_FILE) down
	@echo "$(GREEN)✓ Cleanup complete (data preserved)$(NC)"

.PHONY: clean-all
clean-all: check-node ## Зупинити та видалити контейнери разом з даними
	@echo "$(RED)WARNING: This will delete all data on Node $(NODE)!$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "$(YELLOW)Removing all containers and volumes on Node $(NODE)...$(NC)"; \
		docker compose -f $(COMPOSE_FILE) down -v; \
		echo "$(GREEN)✓ Complete cleanup done$(NC)"; \
	else \
		echo "$(YELLOW)Cancelled$(NC)"; \
	fi

# ══════════════════════════════════════════════════════════
# Оновлення образів
# ══════════════════════════════════════════════════════════

.PHONY: pull
pull: check-node ## Завантажити нові версії Docker образів
	@echo "$(GREEN)Pulling latest images...$(NC)"
	docker compose -f $(COMPOSE_FILE) pull
	@echo "$(GREEN)✓ Images updated$(NC)"

.PHONY: update
update: pull restart-all ## Оновити та перезапустити всі сервіси

# ══════════════════════════════════════════════════════════
# Backup та відновлення
# ══════════════════════════════════════════════════════════

.PHONY: backup
backup: ## Створити backup бази даних (тільки через Node 1)
	@if [ "$(NODE)" != "1" ]; then \
		echo "$(YELLOW)Note: Backup should be done from Node 1 (Load Balancer)$(NC)"; \
	fi
	@mkdir -p backups
	@BACKUP_FILE="backups/backup-$$(date +%Y%m%d-%H%M%S).surql"; \
	echo "$(GREEN)Creating backup: $$BACKUP_FILE$(NC)"; \
	surreal export --endpoint http://192.168.1.161 \
		--username root --password SecurePassword123! \
		--namespace test --database test \
		$$BACKUP_FILE && \
	echo "$(GREEN)✓ Backup created: $$BACKUP_FILE$(NC)"

# ══════════════════════════════════════════════════════════
# Швидкий запуск кластера
# ══════════════════════════════════════════════════════════

.PHONY: init-cluster
init-cluster: ## Ініціалізувати кластер (запустити PD на всіх нодах)
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║  Initializing Cluster - Starting PD on all nodes          ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Starting PD on Node $(NODE)...$(NC)"
	@make start-pd NODE=$(NODE)
	@echo ""
	@echo "$(GREEN)✓ PD started on Node $(NODE)$(NC)"
	@echo ""
	@echo "$(YELLOW)Note: Run this command on all 3 nodes before starting TiKV$(NC)"

.DEFAULT_GOAL := help

.PHONY: cluster-info
cluster-info: ## Детальна інформація про кластер
	@./scripts/cluster-info.sh
