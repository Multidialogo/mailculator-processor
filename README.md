# MultiDialogo - MailCulator Processor

## Requirements

- docker
- docker compose v2
- git

## Provisioning

### Scripts

#### How to start/stop local development environment

```bash
/bin/sh ./local/start-devenv.sh
```

```bash
/bin/sh ./local/stop-devenv.sh
```

#### Run tests

```bash
/bin/sh ./local/test.sh
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
