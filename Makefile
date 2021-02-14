GO=go
GOTEST=$(GO) test
GOCOVER=$(GO) tool cover
COVEROUT=./.cover/c.out
USER_ID ?= $$(id -u)
GROUP_ID ?= $$(id -g)

BACKOFFICE=./cmd/backoffice/backoffice-api
PROXY=./cmd/proxy/proxy-server

.PHONY: app down local data clean

vars:
	@echo USER_ID=${USER_ID}
	@echo GROUP_ID=${GROUP_ID}

data: vars
	@echo Starting Mongo and Minio
	docker-compose up -d s3 mongo-primary mongo-secondary mongo-arbiter mongo-setup mongo-express

data-recreate: vars
	@echo Recreating and starting Mongo and Minio
	docker-compose up -d --force-recreate --build --remove-orphans --renew-anon-volumes s3 mongo-primary mongo-secondary mongo-arbiter mongo-setup mongo-express

app: vars
	@echo Starting Backoffice and Proxy
	docker-compose up -d backoffice proxy

app-recreate: vars
	@echo
	docker-compose up -d --force-recreate --build backoffice proxy

down:
	@echo Stopping Mongo and Minio
	docker-compose rm --force --stop -v

clean: down

local/lint:
	golangci-lint run ./...

local/test:
	@echo Starting tests
	$(GOTEST) ./internal/registry/mgoregistry ./internal/media/manipulator ./internal/media ./internal/backoffice ./internal/proxy  -race

local/test/cover:
	@echo Starting tests with coverage
	$(GOTEST) ./internal/registry/mgoregistry ./internal/media/manipulator ./internal/media ./internal/backoffice ./internal/proxy -cover -coverpkg=./... -coverprofile=$(COVEROUT) . && $(GOCOVER) -html=$(COVEROUT)

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




