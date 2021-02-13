GO=go
GOTEST=$(GO) test
GOCOVER=$(GO) tool cover
COVEROUT=./.cover/c.out
USER_ID ?= $$(id -u)
GROUP_ID ?= $$(id -g)

BACKOFFICE=./cmd/backoffice/backoffice-api
PROXY=./cmd/proxy/proxy-server

.PHONY: up down local data clean

vars:
	@echo USER_ID=${USER_ID}
	@echo GROUP_ID=${GROUP_ID}

up: vars
	@echo Starting Mongo and Minio
	docker-compose up

up-recreate: vars
	@echo Recreating anf starting Mongo and Minio
	docker-compose up --force-recreate --build --remove-orphans --renew-anon-volumes

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




