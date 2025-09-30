# Database

## DynamoDB Schema

### Tabella Outbox
- **Nome**: Configurabile via `EMAIL_OUTBOX_TABLE`
- **Chiave Primaria**: `Id` (String)
- **Sort Key**: `Status` (String)

### Attributi
```go
type Email struct {
    Id          string  // Chiave primaria
    Status      string  // Sort key / Stato corrente
    EmlFilePath string  // Path del file .eml su EFS
    UpdatedAt   string  // Timestamp RFC3339 dell'ultimo aggiornamento
    Reason      string  // Motivo dell'ultimo stato
}
```

## Pattern di Versionamento

### Status Meta
- **Costante**: `StatusMeta = "_META"`
- **Scopo**: Tiene traccia dello stato più recente dell'email

### Status Index
- **Nome**: `StatusIndex`
- **Proiezione**: `Id, Status, Attributes`
- **Query**: Filtra per `Status` e `Attributes.Latest`

### Struttura Dati
Ogni email ha due tipi di record:
1. **Record Meta**: `Status = "_META"` con `Attributes.Latest = "{stato_corrente}"`
2. **Record Stato**: `Status = "{stato_corrente}"` con `Attributes = {}` (vuoto)

### Transazione Update
Ogni cambio di stato esegue una transazione con due statement:
```sql
-- Update meta record
UPDATE "table" SET Attributes.Latest=?, Attributes.UpdatedAt=?, Attributes.Reason=? 
WHERE Id=? AND Status=?

-- Insert new status record  
INSERT INTO "table" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}
```

## Stati Disponibili
- `READY`
- `PROCESSING`
- `SENT`
- `FAILED`
- `CALLING-SENT-CALLBACK`
- `CALLING-FAILED-CALLBACK`
- `SENT-ACKNOWLEDGED`
- `FAILED-ACKNOWLEDGED`

## Query Pattern

### Query per Stato
```sql
SELECT Id, Status, Attributes FROM "table"."StatusIndex" 
WHERE Status=? AND Attributes.Latest =?
```
- **Parametri**: `[StatusMeta, target_status]`
- **Limit**: 25 record per query

### Paginazione
- Utilizza `NextToken` di DynamoDB per paginazione automatica
- Interrompe quando raggiunge il limite di 25 record

## PartiQL Operations

### ExecuteStatement
Utilizzato per query con parametri e paginazione.

### ExecuteTransaction  
Utilizzato per aggiornamenti atomici che coinvolgono meta record e nuovo stato record.
