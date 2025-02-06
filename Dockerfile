# Stage 1: Builder
FROM golang:1.23 AS mailculatorp-builder
RUN mkdir -p /usr/local/go/src/mailculator-processor
WORKDIR /usr/local/go/src/mailculator-processor
COPY . .
COPY .env.test /usr/local/go/src/mailculato-processorr/.env
RUN go mod tidy
RUN go mod download
RUN go test ./...
RUN go build -o /usr/local/bin/mailculator-processor/daemon .
RUN chmod +x /usr/local/bin/mailculator-processor/daemon


# Stage 2: Development
FROM golang:1.23 AS mailculatorp-dev
WORKDIR /usr/local/go/src/mailculator-processor
COPY . .
COPY .env.dev /usr/local/go/src/mailculator-processor/.env
RUN go mod tidy
RUN go mod download
EXPOSE 8080
CMD ["go", "run", "main.go"]

# Stage 3: Production
FROM gcr.io/distroless/base-debian12 AS mailculatorp-prod
WORKDIR /usr/local/bin/mailculator
COPY --from=mailculatorp-builder /usr/local/bin/mailculator-processor/daemon /usr/local/bin/mailculator-processor/daemon
COPY .env.prod /usr/local/bin/mailculator-processor/.env
EXPOSE 8080
CMD ["daemon"]
