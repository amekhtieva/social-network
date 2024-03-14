FROM golang:1.22-alpine

WORKDIR /src/user_service
COPY go.* .

RUN go mod download -x

COPY * .

RUN go build .

ENTRYPOINT ["./user_service"]
