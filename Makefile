.PHONY: up down logs clean
up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

clean:
	docker-compose down -v
	rm -rf neuroboost-web/dist
