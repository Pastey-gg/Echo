.PHONY: up down restart setup purge

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose restart

setup:
	docker compose up --build -d

purge:
	docker compose down --volumes --remove-orphans
