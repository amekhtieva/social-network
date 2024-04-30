FROM golang:1.22-alpine

WORKDIR /src/user_service
COPY proto/ proto/
COPY authentication.go authentication.go
COPY main.go main.go
COPY post_handlers.go post_handlers.go
COPY user_handlers.go user_handlers.go
COPY go.mod go.mod

RUN go mod tidy
RUN go build

ENTRYPOINT ["./user_service"]
