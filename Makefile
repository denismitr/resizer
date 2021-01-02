GO=go
GOTEST=$(GO) test ./... -race
GOCOVER=$(GO) tool cover
COVEROUT=./cover/c.out

BACKOFFICE=./cmd/backoffice/backoffice-api

local/up:
	@echo Starting Mongo and Minio
	docker-compose up -d --force-recreate
	docker-compose exec mongo-primary mongo /root/000_init_rs.js
	docker-compose exec mongo-primary mongo /root/001_create_db.js

local/down:
	@echo Stopping Mongo and Minio
	docker-compose rm --force --stop -v

local/test:
	@echo Starting tests
	$(GOTEST)

local/build:
	@echo Building...
	${GO} build -o cmd/backoffice/backoffice-api cmd/backoffice/main.go

local/run/backoffice:
	@echo Launcing backoffice
	${BACKOFFICE}




