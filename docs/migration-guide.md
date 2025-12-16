# Migration Guide: DynamoDB to MariaDB to Laravel/SQS

This document describes the complete migration path from the current DynamoDB-based system to MariaDB, and subsequently to Laravel with SQS.

## Table of Contents

1. [Current State](#current-state)
2. [Phase 1: Producer Switch to MySQL](#phase-1-producer-switch-to-mysql)
3. [Phase 2: Drain DynamoDB](#phase-2-drain-dynamodb)
4. [Phase 3: Disable DynamoDB Pipelines](#phase-3-disable-dynamodb-pipelines)
5. [Phase 4: Migration to Laravel + SQS](#phase-4-migration-to-laravel--sqs)
6. [Rollback Procedures](#rollback-procedures)

---

## Current State

After implementing the MySQL support, the system now has **parallel pipelines**:

```
┌─────────────────────────────────────────────────────────────┐
│                        Go Application                        │
│                                                              │
│  ┌─────────────────────────┐  ┌─────────────────────────┐   │
│  │   DynamoDB Pipelines    │  │    MySQL Pipelines      │   │
│  │   (processing old msgs) │  │   (idle - no messages)  │   │
│  │                         │  │                         │   │
│  │  - IntakePipeline       │  │  - IntakePipeline       │   │
│  │  - SenderPipeline       │  │  - SenderPipeline       │   │
│  │  - SentCallbackPipeline │  │  - SentCallbackPipeline │   │
│  │  - FailedCallbackPipe   │  │  - FailedCallbackPipe   │   │
│  └───────────┬─────────────┘  └───────────┬─────────────┘   │
│              │                            │                  │
│              ▼                            ▼                  │
│        ┌──────────┐                ┌──────────┐             │
│        │ DynamoDB │                │ MariaDB  │             │
│        └──────────┘                └──────────┘             │
└─────────────────────────────────────────────────────────────┘
                    │
                    │ Producer writes to DynamoDB
                    ▼
            ┌──────────────┐
            │   Producer   │
            │ (external)   │
            └──────────────┘
```

### Configuration

Current environment variables:

```bash
# Pipeline toggles
DYNAMODB_PIPELINES_ENABLED=true  # DynamoDB pipelines active
MYSQL_PIPELINES_ENABLED=true     # MySQL pipelines active (but idle)

# MariaDB connection
MYSQL_HOST=<rds-endpoint>
MYSQL_PORT=3306
MYSQL_USER=<username>
MYSQL_PASSWORD=<password>
MYSQL_DATABASE=mailculator
```

---

## Phase 1: Producer Switch to MariaDB

### Prerequisites

1. MariaDB instance is provisioned and accessible
2. Schema has been applied (migrations executed)
3. Application is deployed with both pipeline types enabled
4. MariaDB pipelines have been tested (manual test messages)

### Steps

#### 1.1 Verify MariaDB Connectivity

```bash
# From application container
mysql -h $MYSQL_HOST -u $MYSQL_USER -p$MYSQL_PASSWORD $MYSQL_DATABASE -e "SELECT 1"
```

#### 1.2 Test MariaDB Pipelines (Optional)

Insert a test message directly into MariaDB:

```sql
INSERT INTO emails (id, status, payload_file_path, ttl)
VALUES (UUID(), 'ACCEPTED', '/path/to/test/payload.json', UNIX_TIMESTAMP() + 3600);
```

Verify:
- Message transitions through states
- Email is sent successfully
- Callback is executed

#### 1.3 Switch Producer

Update the producer application to write to MariaDB instead of DynamoDB.

**For the producer (external system):**

```sql
-- New email insertion (MariaDB)
INSERT INTO emails (id, status, payload_file_path, ttl)
VALUES (?, 'ACCEPTED', ?, ?);

INSERT INTO email_status_history (email_id, status, reason)
VALUES (?, 'ACCEPTED', 'Created by producer');
```

#### 1.4 Verify

After switching:

```sql
-- Check new messages are being created in MariaDB
SELECT COUNT(*) FROM emails WHERE created_at > NOW() - INTERVAL 5 MINUTE;

-- Check DynamoDB is no longer receiving new messages
-- (via AWS Console or CLI)
```

---

## Phase 2: Drain DynamoDB

### Timeline

With a TTL of 10-14 days, DynamoDB will naturally drain within 2 weeks after the producer switch.

### Monitoring

#### Check DynamoDB Queue Depth

Monitor these states in DynamoDB:

```bash
# Via AWS CLI or Console
# Count messages in each "active" state:
# - ACCEPTED
# - INTAKING
# - READY
# - PROCESSING
# - SENT
# - FAILED
# - CALLING-SENT-CALLBACK
# - CALLING-FAILED-CALLBACK
```

#### Application Logs

Monitor logs for DynamoDB pipeline activity:

```bash
# Look for log entries from DynamoDB pipelines
# When queue is empty, pipelines will log 0 messages found
```

### Expected Behavior

```
Day 0:   Producer switches to MySQL
Day 1:   DynamoDB: ~30k messages (backlog), MySQL: ~30k messages (new)
Day 2:   DynamoDB: ~25k messages, MySQL: ~60k messages
...
Day 7:   DynamoDB: ~5k messages, MySQL: ~210k messages
Day 14:  DynamoDB: ~0 messages (TTL expired), MySQL: ~420k messages
```

---

## Phase 3: Disable DynamoDB Pipelines

### Prerequisites

1. DynamoDB has been empty for at least 24 hours
2. All messages have been processed (no ACCEPTED/READY/PROCESSING states)
3. Monitoring confirms no activity on DynamoDB pipelines

### Steps

#### 3.1 Disable DynamoDB Pipelines

Update environment variables:

```bash
DYNAMODB_PIPELINES_ENABLED=false
MYSQL_PIPELINES_ENABLED=true
```

Deploy the application with new configuration.

#### 3.2 Verify Application Health

```bash
# Check health endpoint
curl http://localhost:8080/health

# Check logs for successful startup with only MySQL pipelines
```

#### 3.3 Remove DynamoDB Infrastructure

After confirming stability (wait 1-2 days):

1. Delete DynamoDB table via AWS Console or CDK
2. Remove DynamoDB-related environment variables
3. (Optional) Remove DynamoDB code from application

---

## Phase 4: Migration to Laravel + SQS

### Architecture Target

```
┌─────────────┐
│  Producer   │
│  (Laravel)  │
└──────┬──────┘
       │ dispatch()
       ▼
┌──────────────────────────────────────────────────────────────┐
│                         SQS Queues                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │  intake  │  │   send   │  │   sent   │  │  failed  │     │
│  │  queue   │  │  queue   │  │ callback │  │ callback │     │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘     │
└───────┼─────────────┼─────────────┼─────────────┼────────────┘
        ▼             ▼             ▼             ▼
┌──────────────────────────────────────────────────────────────┐
│                      Laravel Workers                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │ Intake   │  │  Send    │  │   Sent   │  │  Failed  │     │
│  │   Job    │  │   Job    │  │ Callback │  │ Callback │     │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘     │
│                            │                                  │
│                            ▼                                  │
│                    ┌──────────────┐                          │
│                    │    MySQL     │                          │
│                    │  (existing)  │                          │
│                    └──────────────┘                          │
└──────────────────────────────────────────────────────────────┘
```

### Implementation Steps

#### 4.1 Create Laravel Project

```bash
laravel new mailculator-laravel
cd mailculator-laravel

# Install dependencies
composer require aws/aws-sdk-php-laravel
```

#### 4.2 Configure SQS

```php
// config/queue.php
'connections' => [
    'sqs' => [
        'driver' => 'sqs',
        'key' => env('AWS_ACCESS_KEY_ID'),
        'secret' => env('AWS_SECRET_ACCESS_KEY'),
        'prefix' => env('SQS_PREFIX'),
        'queue' => env('SQS_QUEUE', 'default'),
        'region' => env('AWS_DEFAULT_REGION', 'eu-west-1'),
        'after_commit' => true,
    ],
],
```

#### 4.3 Create Job Classes

Create four job classes corresponding to the current pipelines:

```php
// app/Jobs/Email/ProcessIntakeJob.php
// app/Jobs/Email/SendEmailJob.php
// app/Jobs/Email/SentCallbackJob.php
// app/Jobs/Email/FailedCallbackJob.php
```

Each job should:
1. Implement `ShouldQueue`
2. Use optimistic locking via version column
3. Dispatch the next job in the chain upon success

#### 4.4 Database Configuration

Use the same MariaDB database:

```php
// config/database.php
'mysql' => [
    'driver' => 'mysql',
    'host' => env('DB_HOST', '127.0.0.1'),
    'port' => env('DB_PORT', '3306'),
    'database' => env('DB_DATABASE', 'mailculator'),
    'username' => env('DB_USERNAME', 'root'),
    'password' => env('DB_PASSWORD', ''),
    // ...
],
```

#### 4.5 Migration Strategy

Similar to DynamoDB → MySQL migration:

1. **Deploy Laravel workers (idle)** - No messages in SQS yet
2. **Switch producer** - Start dispatching to SQS instead of inserting directly
3. **Run both systems in parallel** - Go processes MariaDB backlog, Laravel processes new SQS messages
4. **Drain Go application** - Wait for MySQL queue to empty
5. **Decommission Go** - Remove Go application

### Producer Code (Laravel)

```php
// When creating a new email
public function sendEmail(Request $request)
{
    // Validate request
    $validated = $request->validate([...]);
    
    // Store payload file
    $payloadPath = $this->storePayload($validated);
    
    // Create email record
    $email = Email::create([
        'id' => Str::uuid(),
        'status' => 'ACCEPTED',
        'payload_file_path' => $payloadPath,
        'ttl' => now()->addDays(14)->timestamp,
    ]);
    
    // Dispatch intake job
    ProcessIntakeJob::dispatch($email->id);
    
    return response()->json(['id' => $email->id], 202);
}
```

---

## Rollback Procedures

### Rollback from Phase 1 (Producer Switch)

If issues occur after switching the producer to MariaDB:

1. Switch producer back to DynamoDB
2. MariaDB messages will still be processed by MariaDB pipelines
3. New messages will go to DynamoDB

### Rollback from Phase 3 (DynamoDB Disabled)

If issues occur after disabling DynamoDB:

1. Re-enable DynamoDB pipelines:
   ```bash
   DYNAMODB_PIPELINES_ENABLED=true
   ```
2. Switch producer back to DynamoDB (if needed)

### Rollback from Phase 4 (Laravel Migration)

If issues occur during Laravel migration:

1. Stop Laravel workers
2. Switch producer back to direct MariaDB insert
3. Go application will continue processing

---

## Monitoring Checklist

### During Phase 1-2

- [ ] MariaDB pipeline processing rate
- [ ] DynamoDB pipeline processing rate (should decrease)
- [ ] Email delivery success rate
- [ ] Callback success rate
- [ ] Error logs

### During Phase 3

- [ ] Application startup logs
- [ ] Health check endpoint
- [ ] MariaDB connection pool metrics
- [ ] No DynamoDB errors (pipelines disabled)

### During Phase 4

- [ ] SQS queue depth
- [ ] Laravel worker logs
- [ ] Job failure rate
- [ ] Dead Letter Queue (DLQ) messages
- [ ] Database query performance

---

## Timeline Summary

| Phase | Duration | Description |
|-------|----------|-------------|
| Phase 1 | 1 day | Switch producer to MariaDB |
| Phase 2 | 10-14 days | Wait for DynamoDB to drain (TTL) |
| Phase 3 | 1 day | Disable DynamoDB pipelines |
| Phase 4 | 2-3 weeks | Develop and deploy Laravel + SQS |

**Total estimated time: 4-6 weeks**

---

## Cost Impact

| Phase | DynamoDB Cost | MariaDB Cost | SQS Cost |
|-------|---------------|------------|----------|
| Before | ~$1,600/month | $0 | $0 |
| Phase 1-2 | ~$800/month (decreasing) | ~$116/month | $0 |
| Phase 3+ | $0 | ~$116/month | $0 |
| Phase 4+ | $0 | ~$116/month | ~$1/month |

**Annual savings: ~$17,800**
