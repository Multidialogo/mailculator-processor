FROM golang:1.24-alpine3.21 AS builder

WORKDIR /build/app

COPY . .

RUN go mod tidy && go build -o bin/main .


FROM alpine:3.21 AS deploy

WORKDIR /app/bin

COPY --from=builder /build/app/bin .

CMD ["./main"]
