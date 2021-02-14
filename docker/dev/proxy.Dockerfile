FROM golang:1.15.8-buster

ENV GO111MODULE=on
ENV APP_HOME=/app
ENV CGO_ENABLED=0
ENV GOOS=linux

ARG GROUP_ID
ARG USER_ID

WORKDIR $APP_HOME

COPY go.mod go.sum ./

RUN go mod download
RUN go mod verify

COPY cmd/ ./cmd
COPY internal/ ./internal
COPY .env ./

RUN go build -o proxy ./cmd/proxy

EXPOSE 3000

CMD ["./proxy"]



