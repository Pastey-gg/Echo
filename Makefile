.PHONY: up down restart purge restart-build up-build

up:
	docker compose up -d

up-build:
	docker compose up --build -d

down:
	docker compose down

restart:
	docker compose restart

restart-build:
	docker compose up --build -d --force-recreate

purge:
	docker compose down --volumes --remove-orphans
