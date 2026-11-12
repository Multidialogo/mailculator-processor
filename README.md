# MultiDialogo - MailCulator Processor

Un'applicazione scritta in Go che elabora email attraverso pipeline parallele utilizzando MySQL come storage principale.

## 📚 Documentazione

La documentazione completa del progetto è disponibile nella directory [`docs/`](./docs/):

- [**Architettura Generale**](./docs/architecture.md) - Panoramica dei componenti principali e infrastruttura AWS
- [**Pipeline Parallele**](./docs/pipeline.md) - Dettagli sui flussi di elaborazione degli email
- [**Database**](./docs/database.md) - Schema MySQL e pattern di versionamento
- [**Gestione Errori**](./docs/error-handling.md) - Strategie di retry e gestione degli errori

## 🚀 Avvio Rapido

### Prerequisiti

- docker
- docker compose v2
- git

### Ambiente di Sviluppo Locale

```bash
# Avvia le dipendenze per lo sviluppo
docker compose --profile devcontainer-deps up -d --build
```

```bash
# Ferma le dipendenze
docker compose --profile devcontainer-deps down --remove-orphans
```

### Testing

```bash
# Esegui i test locali
/bin/sh ./run-tests-local.sh
```

Un report di coverage verrà esportato in `.coverage/report.html`

```bash
# Apri il report di coverage
open ".coverage/report.html"
```

```bash
# Simula gli stage di deployment
/bin/sh ./run-tests-ci.sh
```

### Strumenti Grafici

- **SMTP (mailpit)**: http://localhost:9002

## 🏗️ Costruzione e Deployment

Il progetto utilizza CDK per il deployment su AWS ECS Fargate con:
- MySQL 8.4 (RDS) in stage e MariaDB (RDS) in produzione per lo storage dei metadati
- EFS per i file email
- CloudWatch e Datadog per il monitoring
