# MultiDialogo - MailCulator Processor

## Requirements

- docker
- docker compose v2
- git

## Provisioning

### Scripts

#### How to start/stop local development environment

```bash
docker compose -f deployments/compose.yml --profile local-dev up -d --build --force-recreate
```

```bash
docker compose -f deployments/compose.yml --profile local-dev down --remove-orphans -v
```

#### Run tests

```bash
docker compose -f deployments/compose.yml exec local-dev sh deployments/resources/coverage.sh
```

```bash
open ".coverage/report.html"
```

#### Simulate deployment stages

```bash
/bin/sh ./deployments/test.sh
```

### Graphic tools

- database administration (dbadmin): http://localhost:8001
- smtp (mailpit): http://localhost:8025
