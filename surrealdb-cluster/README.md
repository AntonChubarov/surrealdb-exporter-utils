# SurrealDB Cluster Deployment Guide

Розподілений кластер SurrealDB з TiKV backend та Nginx Load Balancer на трьох вузлах.

## 📋 Архітектура
```
┌─────────────────────────────────────────────────┐
│           Nginx Load Balancer                   │
│           192.168.1.161:80                      │
└────────────────┬────────────────────────────────┘
                 │
    ┌────────────┴──────────────┐
    ▼            ▼              ▼
┌─────────┐  ┌─────────┐  ┌─────────┐
│SurrealDB│  │SurrealDB│  │SurrealDB│
│ Node 1  │  │ Node 2  │  │ Node 3  │
│  :8000  │  │  :8000  │  │  :8000  │
└────┬────┘  └────┬────┘  └────┬────┘
     └────────────┼─────────────┘
                  │
      tikv://pd-cluster:2379
                  │
    ┌─────────────┴──────────────┐
    ▼             ▼              ▼
┌─────────┐  ┌─────────┐  ┌─────────┐
│   PD 1  │◄─►   PD 2  │◄─►   PD 3  │
│ :2379   │  │ :2379   │  │ :2379   │
└────┬────┘  └────┬────┘  └────┬────┘
     ▼            ▼             ▼
┌─────────┐  ┌─────────┐  ┌─────────┐
│ TiKV 1  │◄─► TiKV 2  │◄─► TiKV 3  │
│ :20160  │  │ :20160  │  │ :20160  │
└─────────┘  └─────────┘  └─────────┘

Node 1: 192.168.1.161
Node 2: 192.168.1.182
Node 3: 192.168.1.203
```

## 🚀 Швидкий старт

### Передумови

**На кожній машині:**
- Docker 20.10+
- Docker Compose 2.0+
- Make
- curl, jq (для моніторингу)
- Мінімум 4GB RAM
- 20GB вільного місця

### Встановлення залежностей
```bash
# Debian/Ubuntu
sudo apt update
sudo apt install -y docker.io docker-compose-plugin make curl jq git

# Додайте користувача до групи docker
sudo usermod -aG docker $USER
# Вийдіть та зайдіть заново для застосування змін
```

### Налаштування firewall
```bash
# На КОЖНІЙ машині
sudo ufw allow from 192.168.1.0/24 to any port 2379 proto tcp
sudo ufw allow from 192.168.1.0/24 to any port 2380 proto tcp
sudo ufw allow from 192.168.1.0/24 to any port 20160 proto tcp
sudo ufw allow from 192.168.1.0/24 to any port 8000 proto tcp

# На машині 1 додатково
sudo ufw allow 80/tcp
sudo ufw reload
```

## 📦 Крок 1: Клонування репозиторію
```bash
# На КОЖНІЙ з трьох машин
git clone <your-repo-url> ~/surrealdb-cluster
cd ~/surrealdb-cluster
```

## ⚙️ Крок 2: Запуск кластера

### Послідовність запуску (ВАЖЛИВО!)

#### 2.1. Запуск Placement Drivers (PD)

**На машині 1 (192.168.1.161):**
```bash
cd ~/surrealdb-cluster
make start-pd NODE=1
```

**На машині 2 (192.168.1.182):**
```bash
cd ~/surrealdb-cluster
make start-pd NODE=2
```

**На машині 3 (192.168.1.203):**
```bash
cd ~/surrealdb-cluster
make start-pd NODE=3
```

**Зачекайте 30 секунд** для формування PD кластера.

#### 2.2. Перевірка PD кластера
```bash
# На будь-якій машині
curl http://192.168.1.161:2379/pd/api/v1/members | jq
```

Повинен показати всі 3 PD ноди.

#### 2.3. Запуск TiKV

**На машині 1:**
```bash
make start-tikv NODE=1
```

**На машині 2:**
```bash
make start-tikv NODE=2
```

**На машині 3:**
```bash
make start-tikv NODE=3
```

**Зачекайте 60 секунд** для ініціалізації TiKV.

#### 2.4. Запуск SurrealDB

**На машині 1:**
```bash
make start-surrealdb NODE=1
```

**На машині 2:**
```bash
make start-surrealdb NODE=2
```

**На машині 3:**
```bash
make start-surrealdb NODE=3
```

#### 2.5. Запуск Load Balancer (тільки Node 1)

**На машині 1:**
```bash
make start-nginx NODE=1
```

## 📊 Крок 3: Перевірка кластера

### Базова перевірка
```bash
# Показати статус поточної ноди
make status

# Показати статус всього кластера
make cluster-status

# Детальна перевірка здоров'я
make health
```

### Тест підключення
```bash
# Перевірка через Load Balancer
curl http://192.168.1.161/health

# Перевірка версії SurrealDB
curl http://192.168.1.161/version

# Тестовий SQL запит
curl -X POST http://192.168.1.161/sql \
  -u "root:SecurePassword123!" \
  -H "NS: test" -H "DB: test" \
  -H "Accept: application/json" \
  -d "CREATE test:1 SET message = 'Hello from cluster';"
```

## 🎯 Makefile команди

### Основні команди

| Команда | Опис |
|---------|------|
| `make help` | Показати всі доступні команди |
| `make info` | Інформація про поточну ноду |
| `make start-all` | Запустити всі сервіси |
| `make stop-all` | Зупинити всі сервіси |
| `make restart-all` | Перезапустити всі сервіси |
| `make status` | Статус контейнерів |
| `make logs` | Переглянути логи |
| `make health` | Перевірка здоров'я |
| `make cluster-status` | Статус всього кластера |

### Вибіркове управління компонентами
```bash
# Запуск окремих компонентів
make start-pd NODE=1
make start-tikv NODE=2
make start-surrealdb NODE=3

# Зупинка окремих компонентів
make stop-pd
make stop-tikv
make stop-surrealdb

# Перегляд логів окремих компонентів
make logs-pd
make logs-tikv
make logs-surrealdb
```

### Очищення
```bash
# Зупинити без видалення даних
make clean

# Зупинити і ВИДАЛИТИ всі дані (будьте обережні!)
make clean-all
```

## 🔧 Управління кластером

### Перезапуск окремої ноди
```bash
# На потрібній машині
make restart-all
```

### Оновлення Docker образів
```bash
# Завантажити нові версії
make pull

# Оновити і перезапустити
make update
```

### Backup бази даних
```bash
# На машині 1
make backup
```

Backup буде збережено в `backups/backup-YYYYMMDD-HHMMSS.surql`

## 📝 Підключення до кластера

### SurrealDB CLI
```bash
# Встановлення
curl -sSf https://install.surrealdb.com | sh

# Підключення через Load Balancer
surreal sql \
  --endpoint http://192.168.1.161 \
  --username root \
  --password SecurePassword123! \
  --namespace test \
  --database test
```

### HTTP API
```bash
# Створення даних
curl -X POST http://192.168.1.161/sql \
  -u "root:SecurePassword123!" \
  -H "NS: test" -H "DB: test" \
  -H "Accept: application/json" \
  -d "CREATE person:john SET name = 'John', age = 30;"

# Запит даних
curl -X POST http://192.168.1.161/sql \
  -u "root:SecurePassword123!" \
  -H "NS: test" -H "DB: test" \
  -H "Accept: application/json" \
  -d "SELECT * FROM person;"
```

### GUI (Surrealist)

1. Відкрийте https://surrealist.app
2. Створіть нове підключення:
    - **Endpoint:** `http://192.168.1.161`
    - **Username:** `root`
    - **Password:** `SecurePassword123!`
    - **Namespace:** `test`
    - **Database:** `test`

## 🔍 Моніторинг

### Nginx статистика
```bash
# На машині 1
curl http://192.168.1.161/nginx_status

# Логи Load Balancer
tail -f ~/surrealdb-cluster/node-1/logs/access.log
```

### Перегляд розподілу навантаження
```bash
# Останні 100 запитів
tail -100 ~/surrealdb-cluster/node-1/logs/access.log | \
  grep -oP 'upstream: \K[^:]+' | sort | uniq -c
```

## 🚨 Troubleshooting

### Контейнери не запускаються
```bash
# Перевірте логи
make logs

# Перевірте статус
docker ps -a
```

### PD кластер не формується
```bash
# Видаліть volumes і перезапустіть
make clean-all
make start-pd

# Перевірте мережу між нодами
ping 192.168.1.161
ping 192.168.1.182
ping 192.168.1.203
```

### SurrealDB не може підключитись до TiKV
```bash
# Перевірте TiKV stores
curl http://192.168.1.161:2379/pd/api/v1/stores | jq

# Всі stores повинні мати state: "Up"
```

## 📚 Корисні посилання

- **Офіційна документація:** https://surrealdb.com/docs
- **TiKV документація:** https://tikv.org/docs/
- **Surrealist GUI:** https://surrealist.app
- **GitHub репозиторій:** https://github.com/surrealdb/surrealdb

## 🔐 Безпека

**⚠️ ВАЖЛИВО для продакшн:**

1. Змініть пароль в `compose.yml` файлах
2. Налаштуйте SSL/TLS для Nginx
3. Обмежте доступ через firewall
4. Використовуйте secrets замість паролів в конфігурації

## 📄 Ліцензія

MIT License
