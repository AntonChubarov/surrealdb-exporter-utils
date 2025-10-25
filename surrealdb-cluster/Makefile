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
	@echo "$(YELLOW)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

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
