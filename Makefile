run:
	go run cmd/app/main.go

load-images:
	docker load -i exc/exchange1_amd64.tar
	docker load -i exc/exchange2_amd64.tar
	docker load -i exc/exchange3_amd64.tar

run-images:
	docker run -p 40101:40101 --name exchange1 -d exchange1
	docker run -p 40102:40102 --name exchange2 -d exchange2
	docker run -p 40103:40103 --name exchange3 -d exchange3

rm-docker:
	docker rmi -f $$(docker images -q)

run-ex-docker:
	docker start exchange1
	docker start exchange2
	docker start exchange3

up: load-images
	docker compose up --build

down: 
	docker compose down