
# MultiDialogo - MailCulator Processor

## Provisioning

### How to start local development environment

```bash
docker compose -f ./local/compose.yml --profile develop up -d
```

### Run migrations

```bash
docker compose -f ./local/compose.yml --profile migrations run --rm
```
