.PHONY: all
all: up logs

.PHONY: up
up:
	docker-compose up -d

.PHONY: logs
logs:
	docker-compose logs --tail=1000 -f

.PHONY: ps pull restart stop kill
ps pull restart stop kill:
	docker-compose $@

.PHONY: down
down:
	docker-compose down --remove-orphans
