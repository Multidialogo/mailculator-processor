# Database

## DynamoDB Schema

### Tabella Outbox
- **Nome**: Configurabile via `EMAIL_OUTBOX_TABLE`
- **Chiave Primaria**: `Id` (String)
- **Sort Key**: `Status` (String)

### Attributi
```go
type Email struct {
    Id              string  // Chiave primaria
    Status          string  // Sort key / Stato corrente
    EmlFilePath     string  // Path del file .eml su EFS
    PayloadFilePath string  // Path del file .json contenente il payload per intake
    UpdatedAt       string  // Timestamp RFC3339 dell'ultimo aggiornamento
    Reason          string  // Motivo dell'ultimo stato
    TTL             *int64  // Timestamp Unix in secondi per DynamoDB TTL (nil se non presente)
    Version         int     // Versione per optimistic locking (MySQL) o 0 (DynamoDB)
}
```

### Time To Live (TTL)
DynamoDB TTL è configurato per eliminare automaticamente i record obsoleti. L'attributo `TTL` deve contenere un timestamp Unix (epoch time) in secondi che indica quando il record deve essere eliminato.

**Posizionamento TTL**:
- **Nuovi record**: TTL è posizionato alla radice del record DynamoDB
- **Record legacy**: TTL può essere presente in `Attributes.TTL` per retrocompatibilità
- **Logica di lettura**: Il sistema prima cerca TTL alla radice, poi in `Attributes.TTL` come fallback. Restituisce `nil` se TTL non è presente da nessuna parte.

**Esempio**:
```go
// Record che scade tra 7 giorni
ttl := time.Now().Add(7 * 24 * time.Hour).Unix()
```

## Pattern di Versionamento (DynamoDB)

### Status Meta
- **Costante**: `StatusMeta = "_META"`
- **Scopo**: Tiene traccia dello stato più recente dell'email

### Status Index
- **Nome**: `StatusIndex`
- **Proiezione**: `Id, Status, Attributes`
- **Query**: Filtra per `Status` e `Attributes.Latest`

### Struttura Dati
Ogni email ha due tipi di record:
1. **Record Meta**: `Status = "_META"` con `Attributes.Latest = "{stato_corrente}"` e `TTL = timestamp` alla radice (se presente)
2. **Record Stato**: `Status = "{stato_corrente}"` con `TTL = timestamp` alla radice del record (se presente)

### Transazione Update
Ogni cambio di stato esegue una transazione con due statement:
```sql
-- Update meta record (con TTL se presente)
UPDATE "table" SET Attributes.Latest=?, Attributes.UpdatedAt=?, Attributes.Reason=?, TTL=?
WHERE Id=? AND Status=?

-- Insert new status record (con TTL se presente)
INSERT INTO "table" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?, 'TTL': ?}
```

**Nota**: Il TTL viene sempre sincronizzato tra il record _META e i record di stato. Quando un TTL è presente, viene impostato sia alla radice del record _META che alla radice del nuovo record di stato.

## Unificazione dei Tipi

Entrambi i backend (DynamoDB e MySQL) utilizzano ora lo stesso tipo `Email` con tutti i campi necessari:

- **DynamoDB**: `Version = 0` (non usa locking basato su versione)
- **MySQL**: `Version` popolato dal database per optimistic locking
- **TTL**: Presente per DynamoDB, `nil` per MySQL (non supportato)

Questo approccio elimina la duplicazione dei tipi e semplifica l'architettura.

---

## MySQL Schema

### Tabella `emails`
Tabella principale per la gestione delle email.

```sql
CREATE TABLE IF NOT EXISTS emails (
    id CHAR(36) PRIMARY KEY,
    status ENUM(
        'ACCEPTED','INTAKING','READY','PROCESSING',
        'SENT','FAILED','INVALID',
        'CALLING-SENT-CALLBACK','CALLING-FAILED-CALLBACK',
        'SENT-ACKNOWLEDGED','FAILED-ACKNOWLEDGED'
    ) NOT NULL,
    eml_file_path VARCHAR(500),
    payload_file_path VARCHAR(500),
    reason TEXT,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_status (status),
    INDEX idx_status_updated (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Nota**: A differenza di DynamoDB, MySQL non supporta TTL nativo. La pulizia dei record obsoleti deve essere gestita esternamente.

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

**Nota**: DynamoDB non usa version-based locking, quindi restituisce sempre `Version = 0`.

### Query con SKIP LOCKED
Le query di lettura utilizzano `FOR UPDATE SKIP LOCKED` per:
- Evitare blocchi su righe già in uso da altri worker
- Migliorare il throughput in scenari con più processori concorrenti

```sql
SELECT id, status, eml_file_path, payload_file_path, reason, version, updated_at
FROM emails
WHERE status = ?
ORDER BY updated_at ASC
LIMIT ?
FOR UPDATE SKIP LOCKED
```

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
- `SENT-ACKNOWLEDGED` - Callback per email inviato completato
- `FAILED-ACKNOWLEDGED` - Callback per email fallito completato

## Query Pattern (DynamoDB)

### Query per Stato
```sql
SELECT Id, Status, Attributes, TTL FROM "table"."StatusIndex" 
WHERE Status=? AND Attributes.Latest =?
```
- **Parametri**: `[StatusMeta, target_status]`
- **Limit**: 25 record per query

### Paginazione
- Utilizza `NextToken` di DynamoDB per paginazione automatica
- Interrompe quando raggiunge il limite di 25 record

## PartiQL Operations (DynamoDB)

### ExecuteStatement
Utilizzato per query con parametri e paginazione.

### ExecuteTransaction  
Utilizzato per aggiornamenti atomici che coinvolgono meta record e nuovo stato record.
