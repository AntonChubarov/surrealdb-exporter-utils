# Initial Setup Guide

## First-Time Setup Workflow

### На кожній з трьох машин виконайте:

#### Крок 1: Клонування репозиторію
```bash
git clone <your-repo-url> ~/surrealdb-cluster
cd ~/surrealdb-cluster
```

#### Крок 2: Повна підготовка середовища
```bash
# Виконає всі необхідні перевірки та налаштування
make setup
```

Ця команда автоматично:
- ✅ Перевірить всі залежності (Docker, curl, jq, wget)
- ✅ Перевірить що Docker запущений
- ✅ Створить необхідні директорії
- ✅ Налаштує /etc/hosts
- ✅ Налаштує firewall правила

#### Крок 3: Перевірка готовності
```bash
# Повна перевірка всіх аспектів
make verify
```

Ця команда перевірить:
- ✅ Всі залежності встановлені
- ✅ Docker працює
- ✅ Мережева зв'язність між нодами
- ✅ Достатньо системних ресурсів

#### Крок 4: Запуск кластера

Дотримуйтесь послідовності з README.md:
```bash
# 1. Запустити PD на всіх нодах
make start-pd

# 2. Після 30 секунд запустити TiKV
make start-tikv

# 3. Після 60 секунд запустити SurrealDB
make start-surrealdb

# 4. На Node 1 запустити Nginx
make start-nginx  # тільки на Node 1
```

## Окремі команди налаштування

Якщо потрібно виконати тільки певні кроки:
```bash
# Перевірити залежності
make check-deps

# Перевірити Docker
make check-docker

# Створити директорії
make setup-dirs

# Налаштувати /etc/hosts
make setup-hosts

# Налаштувати firewall
make setup-firewall

# Перевірити мережу
make check-network

# Перевірити ресурси
make check-resources

# Зробити скрипти виконуваними
make setup-scripts
```

## Встановлення додаткових інструментів
```bash
# Netcat, htop, net-tools
make install-tools
```

## Troubleshooting Setup

### Проблема: "Docker command not found"
```bash
# Встановіть Docker
sudo apt update
sudo apt install -y docker.io docker-compose-plugin

# Додайте користувача до групи docker
sudo usermod -aG docker $USER

# Вийдіть та зайдіть заново
exit
```

### Проблема: "Permission denied" при запуску Docker
```bash
# Перевірте чи користувач в групі docker
groups

# Якщо немає, додайте:
sudo usermod -aG docker $USER
newgrp docker  # або вийдіть та зайдіть заново
```

### Проблема: Firewall блокує з'єднання
```bash
# Перевірте статус
sudo ufw status

# Якщо вимкнено, увімкніть:
sudo ufw enable

# Повторно виконайте налаштування
make setup-firewall
```

### Проблема: Неможливо підключитись до інших нод
```bash
# Перевірте мережу
make check-network

# Перевірте firewall на всіх нодах
sudo ufw status numbered

# Ping інші ноди
ping 192.168.1.161
ping 192.168.1.182
ping 192.168.1.203
```
