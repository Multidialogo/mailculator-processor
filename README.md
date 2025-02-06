
# MultiDialogo - MailCulator Processor

## Provisioning

This Dockerfile is designed to build and deploy the `mailculator processor` application using three distinct stages:
1. Builder Stage
2. Development Stage
3. Production Stage

Each stage serves a specific purpose, and you can use them based on your needs.

### Stage 1: Builder

Purpose:
This stage is responsible for building the Go application, running tests, and preparing it for further stages (Development or Production).

Description:
- The base image used is `golang:1.23`.
- The `mailculator processor` project is copied into the container and the necessary dependencies are downloaded using `go mod tidy` and `go mod download`.
- The tests are run with `go test ./...` to ensure everything is correct.
- The application is built with `go build` and the resulting binary is copied to `/usr/local/bin/mailculator-processor` in a file called `daemon`.
- Finally, the binary is made executable with `chmod +x`.

To build the image:
```bash
 docker build --no-cache -t mailculatorp-builder --target mailculatorp-builder .
 ```

To introspect the builder image:

```bash
docker run -ti --rm mailculatorp-builder bash
```

### Stage 2: Development

Purpose: This stage is used for local development.

Description:

The base image used is golang:1.23.
The binary generated in the builder stage is copied into this container.
To build the image:
```bash
docker build --no-cache -t mailculatorp-dev --target mailculatorp-dev .
```

If you want to generate some dummy data:
```bash
sudo chown -R "$(whoami):$(id -gn)" ./data && ./data/maildir/dummy-gen.sh
```

To run the development container:
```bash
docker run --rm -v$(pwd)/data:/var/lib/mailculator-processor mailculatorp-dev
```

To access `Prometheus` live stats [click here](http://localhost:9090/prometheus).

### Stage 3: Production

Purpose: This stage is optimized for production deployment. It creates a minimal container to run the mailculator processor binary in a secure and efficient environment.

Description:

The base image used is gcr.io/distroless/base-debian12, which is a minimal image without unnecessary tools or packages.
The binary from the builder stage is copied into the container.
The container is configured to run the mailculator processor binary.
To build the image:
```bash
docker build --no-cache -t mailculatorp-prod --target mailculatorp-prod .
```

To run the production container:
```bash
docker run --rm mailculatorp-prod
```
