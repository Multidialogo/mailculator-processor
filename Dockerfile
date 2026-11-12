FROM golang:1.25-alpine3.21 AS builder

WORKDIR /build/app

COPY . .

RUN go mod tidy && go build -o bin/main ./cmd/main


FROM alpine:3.21 AS deploy

RUN apk add --no-cache tzdata

WORKDIR /app/bin

COPY --from=builder /build/app/bin .

CMD ["./main"]
