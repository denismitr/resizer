GO=go
GOTEST=$(GO) test
GOCOVER=$(GO) tool cover
COVEROUT=./.cover/c.out

BACKOFFICE=./cmd/backoffice/backoffice-api
PROXY=./cmd/proxy/proxy-server

.PHONY: up down local data clean

data:
	@echo Starting Mongo and Minio
	docker-compose up -d
	docker-compose exec mongo-primary mongo /root/000_init_rs.js
	@sleep 30
	docker-compose exec mongo-primary sh /root/init.sh

up-recreate:
	@echo Recreating anf starting Mongo and Minio
	docker-compose up -d --force-recreate --remove-orphans
	docker-compose exec mongo-primary mongo /root/000_init_rs.js
	@sleep 30
	docker-compose exec mongo-primary sh /root/init.sh

down:
	@echo Stopping Mongo and Minio
	docker-compose rm --force --stop -v

clean: down

local/lint:
	golangci-lint run ./...

local/test:
	@echo Starting tests
	$(GOTEST) ./registry/mgoregistry ./manipulator ./media ./backoffice ./proxy  -race

local/test/cover:
	@echo Starting tests with coverage
	$(GOTEST) ./registry/mgoregistry ./manipulator ./media ./backoffice ./proxy -cover -coverpkg=./... -coverprofile=$(COVEROUT) . && $(GOCOVER) -html=$(COVEROUT)

local/build:
	@echo Building backoffice API...
	${GO} build -o cmd/backoffice/backoffice-api cmd/backoffice/main.go
	@echo Building proxy server...
	${GO} build -o cmd/proxy/proxy-server cmd/proxy/main.go

local/run/backoffice:
	@echo Launcing backoffice
	${BACKOFFICE} -migrate

local/run/proxy:
	@echo Launcing proxy server
	${PROXY}




