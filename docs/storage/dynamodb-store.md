# DynamoDB Store

AWS DynamoDB backend for large-scale, cloud-native workflow applications.

## Overview

The DynamoDB store provides a fully managed, highly scalable persistence layer for Gorkflow using AWS DynamoDB.

## Features

- ✅ **Fully managed** - No server management
- ✅ **Auto-scaling** - Handles traffic spikes automatically
- ✅ **Multi-region** - Global table replication
- ✅ **Pay-per-request** - Cost-effective billing
- ✅ **High availability** - 99.99% SLA
- ✅ **Point-in-time recovery** - Built-in backups
- ✅ **TTL support** - Automatic data cleanup

## Installation

```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
```

## Quick Start

```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/sicko7947/gorkflow/store"
    "github.com/sicko7947/gorkflow/engine"
)

func main() {
    // Load AWS configuration
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        panic(err)
    }

    // Create DynamoDB client
    client := dynamodb.NewFromConfig(cfg)

    // Create store
    store, err := store.NewDynamoDBStore(client, "workflow-executions")
    if err != nil {
        panic(err)
    }

    // Use with engine
    eng := engine.NewEngine(store)
}
```

## Table Setup

### Using Helper Script

Gorkflow includes a helper script to create the DynamoDB table:

```bash
# With default settings (ap-southeast-2, workflow_executions)
./scripts/create-dynamodb-table.sh

# Custom region and table name
export AWS_REGION=us-east-1
export AWS_DYNAMODB_TABLE_NAME=my_workflows
./scripts/create-dynamodb-table.sh
```

### Manual Table Creation

Create a table with the following specification:

**Table Settings:**

- **Partition Key (PK)**: String
- **Sort Key (SK)**: String
- **Billing Mode**: PAY_PER_REQUEST
- **Delete Protection**: Enabled (recommended)

**Global Secondary Indexes:**

1. **GSI1**

   - **PK**: `GSI1PK` (String)
   - **SK**: `GSI1SK` (String)
   - **Projection**: ALL

2. **GSI2**
   - **PK**: `GSI2PK` (String)
   - **SK**: `GSI2SK` (String)
   - **Projection**: ALL

**TTL:**

- **Attribute**: `ttl` (Number)
- Status: Enabled

### AWS CLI Table Creation

```bash
aws dynamodb create-table \
    --table-name workflow_executions \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
        AttributeName=GSI2PK,AttributeType=S \
        AttributeName=GSI2SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [
                    {\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},
                    {\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}
                ],
                \"Projection\": {\"ProjectionType\":\"ALL\"}
            },
            {
                \"IndexName\": \"GSI2\",
                \"KeySchema\": [
                    {\"AttributeName\":\"GSI2PK\",\"KeyType\":\"HASH\"},
                    {\"AttributeName\":\"GSI2SK\",\"KeyType\":\"RANGE\"}
                ],
                \"Projection\": {\"ProjectionType\":\"ALL\"}
            }
        ]" \
    --billing-mode PAY_PER_REQUEST \
    --region us-east-1
```

Enable TTL:

```bash
aws dynamodb update-time-to-live \
    --table-name workflow_executions \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --region us-east-1
```

## Configuration

### AWS Credentials

The DynamoDB store uses the standard AWS SDK credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (when running on AWS)

### Custom AWS Config

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Load with custom region
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithRegion("us-west-2"),
)

client := dynamodb.NewFromConfig(cfg)
store, _ := store.NewDynamoDBStore(client, "workflows")
```

### With Credentials

```go
import (
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/config"
)

cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithCredentialsProvider(
        credentials.NewStaticCredentialsProvider(
            "ACCESS_KEY_ID",
            "SECRET_ACCESS_KEY",
            "",
        ),
    ),
)
```

## Data Model

Gorkflow uses a **Single Table Design** pattern:

| Entity        | PK            | SK                |
| ------------- | ------------- | ----------------- |
| WorkflowRun   | `RUN#<runID>` | `RUN#<runID>`     |
| StepExecution | `RUN#<runID>` | `STEP#<stepID>`   |
| StepOutput    | `RUN#<runID>` | `OUTPUT#<stepID>` |
| WorkflowState | `RUN#<runID>` | `STATE#<key>`     |

### Access Patterns

1. **Get Workflow Run** - PK = `RUN#<runID>`, SK = `RUN#<runID>`
2. **Get All Steps for Run** - PK = `RUN#<runID>`, SK begins_with `STEP#`
3. **Get Step Output** - PK = `RUN#<runID>`, SK = `OUTPUT#<stepID>`
4. **Query by Workflow ID** - GSI1: PK = `WORKFLOW#<workflowID>`
5. **Query by Status** - GSI2: PK = `STATUS#<status>`

## IAM Permissions

Required IAM permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan"
      ],
      "Resource": [
        "arn:aws:dynamodb:us-east-1:123456789012:table/workflow-executions",
        "arn:aws:dynamodb:us-east-1:123456789012:table/workflow-executions/index/*"
      ]
    }
  ]
}
```

## Monitoring

### CloudWatch Metrics

Monitor these DynamoDB metrics:

- `ConsumedReadCapacityUnits`
- `ConsumedWriteCapacityUnits`
- `UserErrors` (e.g., throttling)
- `SystemErrors`

### Cost Optimization

1. **Use PAY_PER_REQUEST** for unpredictable workloads
2. **Enable TTL** to auto-delete old workflows
3. **Archive to S3** for long-term storage
4. **Use provisioned capacity** for predictable workloads

## TTL (Time To Live)

Configure TTL to automatically delete old workflows:

```go
import "time"

// Set TTL when creating run (30 days from now)
run.TTL = time.Now().Add(30 * 24 * time.Hour).Unix()

// Store will automatically delete after TTL expires
```

## Backup and Recovery

### Point-in-Time Recovery

Enable PITR in AWS Console or CLI:

```bash
aws dynamodb update-continuous-backups \
    --table-name workflow_executions \
    --point-in-time-recovery-specification PointInTimeRecoveryEnabled=true
```

### On-Demand Backups

```bash
aws dynamodb create-backup \
    --table-name workflow_executions \
    --backup-name workflow-backup-2024-01-01
```

## Multi-Region Setup

Create a global table for multi-region replication:

```bash
aws dynamodb create-global-table \
    --global-table-name workflow_executions \
    --replication-group RegionName=us-east-1 RegionName=eu-west-1
```

## Best Practices

### 1. Use PAY_PER_REQUEST for Variable Workloads

```go
// No capacity planning needed
// Automatically scales with traffic
```

### 2. Enable TTL for Cleanup

```go
// Automatically delete workflows after 30 days
ttl := time.Now().Add(30 * 24 * time.Hour).Unix()
```

### 3. Use Tags for Cost Allocation

```bash
aws dynamodb tag-resource \
    --resource-arn arn:aws:dynamodb:us-east-1:123456789012:table/workflow_executions \
    --tags Key=Project,Value=Workflows Key=Environment,Value=Production
```

### 4. Monitor Read/Write Capacity

Set CloudWatch alarms for throttling:

```bash
aws cloudwatch put-metric-alarm \
    --alarm-name workflow-throttle \
    --metric-name UserErrors \
    --threshold 10
```

### 5. Use Batch Operations

For bulk operations, use batch read/write:

```go
// Batch get operations are more efficient
// Store implementation handles this automatically
```

## Troubleshooting

### Throttling Errors

If you see throttling errors:

1. Switch to PAY_PER_REQUEST billing
2. Increase provisioned capacity
3. Use exponential backoff (built-in)

### High Costs

To reduce costs:

1. Enable TTL to delete old data
2. Archive to S3 for long-term storage
3. Use provisioned capacity if predictable
4. Monitor with Cost Explorer

### Permission Errors

Ensure IAM role/user has required permissions:

```bash
# Testdynamodb permissions
aws dynamodb describe-table --table-name workflow_executions
```

---

**Next**: Learn about [LibSQL Store](libsql-store.md) or return to [Storage Overview](overview.md) →
