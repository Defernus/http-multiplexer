FROM golang:1.16.6-alpine3.14 as builder

WORKDIR /go/src/app

RUN apk add gcc musl-dev

COPY ./go.mod .

COPY . .

RUN go build -o ./bin/main ./cmd/server/main.go

FROM alpine:3.14

WORKDIR /app

COPY --from=builder /go/src/app/bin .

CMD ["./main"]
