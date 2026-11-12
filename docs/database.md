# Database

## Modello Email
```go
type Email struct {
    Id              string
    Status          string
    PayloadFilePath string
    UpdatedAt       string
    Reason          string
    Version         int     // Versione per optimistic locking
}
```

## MySQL Schema

### Tabella `emails`
Tabella principale per la gestione delle email.

```sql
CREATE TABLE IF NOT EXISTS emails (
    id CHAR(36) PRIMARY KEY,
    status ENUM(
        'ACCEPTED','INTAKING','READY','PROCESSING',
        'SENT','FAILED','INVALID',
        'CALLING-SENT-CALLBACK','CALLING-FAILED-CALLBACK','CALLING-INVALID-CALLBACK',
        'SENT-ACKNOWLEDGED','FAILED-ACKNOWLEDGED','INVALID-ACKNOWLEDGED'
    ) NOT NULL,
    payload_file_path VARCHAR(500),
    reason TEXT,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_status (status),
    INDEX idx_status_updated (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Tabella `email_statuses`
Tabella per lo storico dei cambi di stato (history).

```sql
CREATE TABLE IF NOT EXISTS email_statuses (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    email_id CHAR(36) NOT NULL,
    status VARCHAR(50) NOT NULL,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_email_id (email_id),
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Optimistic Locking (MySQL)
MySQL utilizza optimistic locking basato su:
- Campo `Version` nel tipo `Email` per tracciare le modifiche
- Campo `status` per validare la transizione di stato

Ogni update incrementa la versione e verifica lo stato atteso:
```sql
UPDATE emails
SET status = ?, reason = ?, version = version + 1
WHERE id = ? AND status = ?
```

Se `affected_rows = 0`, l'operazione restituisce `ErrLockNotAcquired`.

### Query con SKIP LOCKED
Le query di lettura utilizzano `FOR UPDATE SKIP LOCKED` per:
- Evitare blocchi su righe già in uso da altri worker
- Migliorare il throughput in scenari con più processori concorrenti

```sql
SELECT id, status, payload_file_path, reason, version, updated_at
FROM emails
WHERE status = ?
ORDER BY updated_at ASC
LIMIT ?
FOR UPDATE SKIP LOCKED
```

### Compatibilità MySQL 8.4 e MariaDB
Solo l'ambiente di stage usa MySQL 8.4 (RDS); produzione resta su MariaDB, quindi il codice deve restare compatibile con entrambi. In locale `compose.yml` usa l'immagine `mysql:8.4`, quindi i test di integrazione non coprono MariaDB. Il codice non usa `mysql_native_password`, disabilitato di default in 8.4: il driver gestisce `caching_sha2_password`. In stage si usa `mysql_native_password`, quindi il plugin deve essere abilitato sull'istanza (in 8.4 con `mysql_native_password=ON`). Il driver `github.com/go-sql-driver/mysql` è lo stesso per MySQL e MariaDB, quindi non richiede modifiche. Le query e le migrazioni non usano costrutti esclusivi di MariaDB. `FOR UPDATE SKIP LOCKED` richiede MySQL 8.0.1+.

La connessione è configurata tramite le variabili `MYSQL_HOST` (endpoint), `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE` (nome dello schema) e `MYSQL_TLS`. In AWS i primi cinque valori sono letti dai campi `host`, `port`, `username`, `password` e `dbname` del secret Secrets Manager indicato in `cdk/task_definition_stack.py`.

### Transazioni
Le operazioni di update e insert history sono eseguite in transazione per garantire atomicità:
1. `BEGIN`
2. `UPDATE emails ...`
3. `INSERT INTO email_statuses ...`
4. `COMMIT` (o `ROLLBACK` in caso di errore)

---

## Stati Disponibili
- `ACCEPTED` - Email accettato, in attesa di intake
- `INTAKING` - Email in fase di elaborazione intake
- `READY` - Email pronto per l'invio
- `PROCESSING` - Email in fase di elaborazione per l'invio
- `SENT` - Email inviato con successo
- `FAILED` - Invio email fallito
- `INVALID` - Intake email fallito
- `CALLING-SENT-CALLBACK` - In corso chiamata callback per email inviato
- `CALLING-FAILED-CALLBACK` - In corso chiamata callback per email fallito
- `CALLING-INVALID-CALLBACK` - In corso chiamata callback per email con payload invalido
- `SENT-ACKNOWLEDGED` - Callback per email inviato completato
- `FAILED-ACKNOWLEDGED` - Callback per email fallito completato
- `INVALID-ACKNOWLEDGED` - Callback per email con payload invalido completato
