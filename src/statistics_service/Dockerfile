FROM golang:1.22-alpine

WORKDIR /src/statistics_service
COPY main.go main.go
COPY server.go server.go
COPY go.mod go.mod

RUN go mod tidy
RUN go build

ENTRYPOINT ["./statistics_service"]
