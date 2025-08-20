.PHONY: run-infra run-booking run-edge-1 run-edge-2 stop-infra clean

run-infra:
	docker-compose up -d redis nats

stop-infra:
	docker-compose down

run-booking:
	go run booking-service/main.go

run-edge-1:
	PORT=3000 go run edge-server/*.go

run-edge-2:
	PORT=3001 go run edge-server/*.go

clean:
	docker-compose down -v
	rm -f go.sum