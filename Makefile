include .env
export

export PROJECT_ROOT=$(shell pwd)

# PostgreSQL управление
env-up:
	@docker compose up -d raportichka-postgres

env-down:
	@docker compose down raportichka-postgres

env-cleanup:
	@read -p "Ты уверен, что хочешь очистить volume файлы окружения [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
	  docker compose down raportichka-postgres && \
	  sudo rm -rf out/postgres && \
	  echo "очистка прошла успешно"; \
  	else \
      echo "Очистка отменена"; \
	fi

env-port-forward:
	@docker compose up -d raportichka-port-forwarder

env-port-close:
	@docker compose down raportichka-port-forwarder

# Приложение управление
app-build:
	@docker compose build raportichka-app

app-up:
	@docker compose up -d raportichka-app

app-down:
	@docker compose down raportichka-app

app-restart:
	@docker compose restart raportichka-app

app-logs:
	@docker compose logs -f raportichka-app

app-shell:
	@docker exec -it raportichka-app sh

# Общие команды
up:
	@docker compose up -d

down:
	@docker compose down

logs:
	@docker compose logs -f

ps:
	@docker compose ps

build:
	@docker compose build

restart:
	@docker compose restart

# Быстрый перезапуск всего
rebuild:
	@docker compose down
	@docker compose build --no-cache
	@docker compose up -d

# Статус PostgreSQL
pg-status:
	@docker exec -it raportichka-postgres pg_isready -U postgres

# Подключение к PostgreSQL
pg-shell:
	@docker exec -it raportichka-postgres psql -U postgres -d raportichka