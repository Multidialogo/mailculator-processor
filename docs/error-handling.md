# Gestione Errori e Retry

## Retry MySQL

### Condizioni di Retry
Il sistema effettua retry automatico per i seguenti errori MySQL:
- `1205` - Lock wait timeout exceeded
- `1213` - Deadlock found when trying to get lock
- `1040` - Too many connections
- `1203` - User already has more than max_user_connections active connections
- `driver.ErrBadConn` - Connessione al database persa

### Errori NON Soggetti a Retry
- `ErrLockNotAcquired` - Conflitto di lock ottimistico (il record è stato modificato da un altro processo)

### Backoff Strategy
- **Max Attempts**: 8 tentativi
- **Base Delay**: 30 millisecondi
- **Max Delay**: 1 secondo
- **Formula**: Durata casuale tra 0 e min(2^attempt * base_delay, max_delay)

### Transazioni
Le operazioni MySQL (Update, Ready, Create) sono eseguite in transazione:
- In caso di errore, viene eseguito automaticamente il rollback
- In caso di successo, viene eseguito il commit
- Gli errori transitori sono gestiti con retry (la transazione viene ritentata dall'inizio)


## Retry Callback HTTP

### Condizioni di Retry
- **Status Code**: 409 (Conflict)
- **Max Retries**: Configurabile via `callback.max_retries` (default: 3)
- **Retry Interval**: Configurabile via `callback.retry_interval` (default: 5 secondi)


## Stati di Errore

### Email Failed
Quando l'invio SMTP fallisce:
- Stato: `FAILED`
- Il sistema notifica il chiamante tramite la FailedCallbackPipeline (HTTP POST con code: "DISPATCH-ERROR")
- Stato finale: `FAILED-ACKNOWLEDGED`

### Email Invalid
Quando la validazione del payload fallisce durante l'intake:
- Stato: `INVALID`
- Il sistema notifica il chiamante tramite la InvalidCallbackPipeline (HTTP POST con code: "DISPATCH-ERROR")
- Stato finale: `INVALID-ACKNOWLEDGED`

### Sanitizzazione del Reason nelle Callback
Il campo `reason` nel database conserva sempre l'errore originale completo per il debugging. La sanitizzazione avviene esclusivamente al momento dell'invio della callback HTTP, nella `CallbackPipeline`:

- **Errori SMTP** (reason che inizia con codice a 3 cifre, es. "552 5.3.4 Message too long"): il messaggio del server viene preservato, in quanto informativo per il cliente
- **Errori interni** (caricamento payload, costruzione messaggio, connessione, autenticazione, TLS, validazione): viene inviato un messaggio generico ("Errore interno di elaborazione")

Il database mantiene sempre l'errore originale completo per il debugging interno.

### SMTP Throttling (454)
Quando l'invio SMTP fallisce con codice `454` (throttling):
- Stato: ritorna a `READY`
- Record `PROCESSING` rimosso da `email_statuses`
- Reason: vuota

### Callback Failed
Quando il callback HTTP fallisce dopo tutti i retry:
- Stato rimane: `CALLING-SENT-CALLBACK`, `CALLING-FAILED-CALLBACK` o `CALLING-INVALID-CALLBACK`
- Log di errore con status code e response body

### Lock Acquisition Failed
Quando non riesce ad acquisire il lock di processamento:
- Operazione saltata
- Log warning: "failed to acquire processing lock"
- **Nessun retry**: `ErrLockNotAcquired` indica che un altro worker sta già processando il record

## Context Cancellation
Tutti i retry rispettano il context cancellation:
- Operazioni interrotte se context è cancelled
- Ritorno dell'errore del context
