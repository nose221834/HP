start-html:
	python3 -m http.server 8001

migrate:
	docker compose exec -it api bin/rails db:migrate
