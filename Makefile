GO=go
GOTEST=$(GO) test ./... -race
GOCOVER=$(GO) tool cover
COVEROUT=./cover/c.out

.PHONY: build

test:
	@echo Starting tests
	$(GOTEST)


