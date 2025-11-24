package store

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sicko7947/gorkflow"
)

// DynamoDBStore implements workflow.WorkflowStore using AWS DynamoDB
type DynamoDBStore struct {
	client    DynamoDBClient
	tableName string
}

// NewDynamoDBStore creates a new DynamoDB-backed workflow store
func NewDynamoDBStore(client DynamoDBClient, tableName string) workflow.WorkflowStore {
	return &DynamoDBStore{
		client:    client,
		tableName: tableName,
	}
}

// Workflow run operations

func (s *DynamoDBStore) CreateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	// Marshal the run
	item, err := attributevalue.MarshalMap(run)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow run: %w", err)
	}

	// Add keys
	item[AttrPK] = &types.AttributeValueMemberS{Value: workflowRunPK(run.RunID)}
	item[AttrSK] = &types.AttributeValueMemberS{Value: workflowRunSK()}
	item[AttrEntityType] = &types.AttributeValueMemberS{Value: EntityTypeWorkflowRun}

	// Add GSI keys
	if run.WorkflowID != "" {
		item[AttrGSI1PK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI1PK(run.WorkflowID, string(run.Status)),
		}
		item[AttrGSI1SK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI1SK(run.CreatedAt.Format(time.RFC3339)),
		}
	}

	if run.ResourceID != "" {
		item[AttrGSI2PK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI2PK(run.ResourceID, string(run.Status)),
		}
		item[AttrGSI2SK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI2SK(run.CreatedAt.Format(time.RFC3339)),
		}
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow run: %w", err)
	}

	return nil
}

func (s *DynamoDBStore) GetRun(ctx context.Context, runID string) (*workflow.WorkflowRun, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			AttrPK: &types.AttributeValueMemberS{Value: workflowRunPK(runID)},
			AttrSK: &types.AttributeValueMemberS{Value: workflowRunSK()},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("workflow run %s not found", runID)
	}

	var run workflow.WorkflowRun
	if err := attributevalue.UnmarshalMap(result.Item, &run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow run: %w", err)
	}

	return &run, nil
}

func (s *DynamoDBStore) UpdateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	run.UpdatedAt = time.Now()

	// Marshal the run
	item, err := attributevalue.MarshalMap(run)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow run: %w", err)
	}

	// Add keys
	item[AttrPK] = &types.AttributeValueMemberS{Value: workflowRunPK(run.RunID)}
	item[AttrSK] = &types.AttributeValueMemberS{Value: workflowRunSK()}
	item[AttrEntityType] = &types.AttributeValueMemberS{Value: EntityTypeWorkflowRun}

	// Update GSI keys (status may have changed)
	if run.WorkflowID != "" {
		item[AttrGSI1PK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI1PK(run.WorkflowID, string(run.Status)),
		}
		item[AttrGSI1SK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI1SK(run.CreatedAt.Format(time.RFC3339)),
		}
	}

	if run.ResourceID != "" {
		item[AttrGSI2PK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI2PK(run.ResourceID, string(run.Status)),
		}
		item[AttrGSI2SK] = &types.AttributeValueMemberS{
			Value: workflowRunGSI2SK(run.CreatedAt.Format(time.RFC3339)),
		}
	}

	// Use transaction for atomic update
	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: aws.String(s.tableName),
					Item:      item,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update workflow run: %w", err)
	}

	return nil
}

func (s *DynamoDBStore) UpdateRunStatus(ctx context.Context, runID string, status workflow.RunStatus, wfErr *workflow.WorkflowError) error {
	// Load current run
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return err
	}

	// Update status and error
	run.Status = status
	run.Error = wfErr
	run.UpdatedAt = time.Now()

	if status.IsTerminal() {
		now := time.Now()
		run.CompletedAt = &now
	}

	// Save
	return s.UpdateRun(ctx, run)
}

func (s *DynamoDBStore) ListRuns(ctx context.Context, filter workflow.RunFilter) ([]*workflow.WorkflowRun, error) {
	// TODO: Implement with Query using GSI1 or GSI2 based on filter
	// For now, return empty list
	return []*workflow.WorkflowRun{}, nil
}

// Step execution operations

func (s *DynamoDBStore) CreateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	exec.UpdatedAt = time.Now()

	// Marshal
	item, err := attributevalue.MarshalMap(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal step execution: %w", err)
	}

	// Add keys
	item[AttrPK] = &types.AttributeValueMemberS{Value: stepExecutionPK(exec.RunID)}
	item[AttrSK] = &types.AttributeValueMemberS{Value: stepExecutionSK(exec.StepID)}
	item[AttrEntityType] = &types.AttributeValueMemberS{Value: EntityTypeStepExecution}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create step execution: %w", err)
	}

	return nil
}

func (s *DynamoDBStore) GetStepExecution(ctx context.Context, runID, stepID string) (*workflow.StepExecution, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			AttrPK: &types.AttributeValueMemberS{Value: stepExecutionPK(runID)},
			AttrSK: &types.AttributeValueMemberS{Value: stepExecutionSK(stepID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get step execution: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("step execution %s/%s not found", runID, stepID)
	}

	var exec workflow.StepExecution
	if err := attributevalue.UnmarshalMap(result.Item, &exec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step execution: %w", err)
	}

	return &exec, nil
}

func (s *DynamoDBStore) UpdateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	exec.UpdatedAt = time.Now()

	// Marshal
	item, err := attributevalue.MarshalMap(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal step execution: %w", err)
	}

	// Add keys
	item[AttrPK] = &types.AttributeValueMemberS{Value: stepExecutionPK(exec.RunID)}
	item[AttrSK] = &types.AttributeValueMemberS{Value: stepExecutionSK(exec.StepID)}
	item[AttrEntityType] = &types.AttributeValueMemberS{Value: EntityTypeStepExecution}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update step execution: %w", err)
	}

	return nil
}

func (s *DynamoDBStore) ListStepExecutions(ctx context.Context, runID string) ([]*workflow.StepExecution, error) {
	var executions []*workflow.StepExecution
	var lastEvaluatedKey map[string]types.AttributeValue

	// Paginate through all results
	for {
		queryInput := &dynamodb.QueryInput{
			TableName:              aws.String(s.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: stepExecutionPK(runID)},
				":sk": &types.AttributeValueMemberS{Value: stepPrefix()},
			},
		}

		if lastEvaluatedKey != nil {
			queryInput.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := s.client.Query(ctx, queryInput)
		if err != nil {
			return nil, fmt.Errorf("failed to list step executions: %w", err)
		}

		for _, item := range result.Items {
			var exec workflow.StepExecution
			if err := attributevalue.UnmarshalMap(item, &exec); err != nil {
				return nil, fmt.Errorf("failed to unmarshal step execution: %w", err)
			}
			executions = append(executions, &exec)
		}

		// Check if there are more results
		if result.LastEvaluatedKey == nil {
			break
		}
		lastEvaluatedKey = result.LastEvaluatedKey
	}

	return executions, nil
}

// Step output operations

func (s *DynamoDBStore) SaveStepOutput(ctx context.Context, runID, stepID string, output []byte) error {
	item := map[string]types.AttributeValue{
		AttrPK:         &types.AttributeValueMemberS{Value: stepOutputPK(runID)},
		AttrSK:         &types.AttributeValueMemberS{Value: stepOutputSK(stepID)},
		AttrEntityType: &types.AttributeValueMemberS{Value: EntityTypeStepOutput},
		"output":       &types.AttributeValueMemberB{Value: output},
		"updated_at":   &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save step output: %w", err)
	}

	return nil
}

func (s *DynamoDBStore) LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			AttrPK: &types.AttributeValueMemberS{Value: stepOutputPK(runID)},
			AttrSK: &types.AttributeValueMemberS{Value: stepOutputSK(stepID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load step output: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("step output %s/%s not found", runID, stepID)
	}

	outputAttr, ok := result.Item["output"]
	if !ok {
		return nil, fmt.Errorf("step output %s/%s has no output field", runID, stepID)
	}

	outputBytes, ok := outputAttr.(*types.AttributeValueMemberB)
	if !ok {
		return nil, fmt.Errorf("step output %s/%s output field is not binary", runID, stepID)
	}

	return outputBytes.Value, nil
}

// State operations

func (s *DynamoDBStore) SaveState(ctx context.Context, runID, key string, value []byte) error {
	item := map[string]types.AttributeValue{
		AttrPK:         &types.AttributeValueMemberS{Value: statePK(runID)},
		AttrSK:         &types.AttributeValueMemberS{Value: stateSK(key)},
		AttrEntityType: &types.AttributeValueMemberS{Value: EntityTypeState},
		"value":        &types.AttributeValueMemberB{Value: value},
		"updated_at":   &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

func (s *DynamoDBStore) LoadState(ctx context.Context, runID, key string) ([]byte, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			AttrPK: &types.AttributeValueMemberS{Value: statePK(runID)},
			AttrSK: &types.AttributeValueMemberS{Value: stateSK(key)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("state key %s not found", key)
	}

	valueAttr, ok := result.Item["value"]
	if !ok {
		return nil, fmt.Errorf("state key %s has no value field", key)
	}

	valueBytes, ok := valueAttr.(*types.AttributeValueMemberB)
	if !ok {
		return nil, fmt.Errorf("state key %s value field is not binary", key)
	}

	return valueBytes.Value, nil
}

func (s *DynamoDBStore) DeleteState(ctx context.Context, runID, key string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			AttrPK: &types.AttributeValueMemberS{Value: statePK(runID)},
			AttrSK: &types.AttributeValueMemberS{Value: stateSK(key)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

func (s *DynamoDBStore) GetAllState(ctx context.Context, runID string) (map[string][]byte, error) {
	stateData := make(map[string][]byte)
	var lastEvaluatedKey map[string]types.AttributeValue

	// Paginate through all results
	for {
		queryInput := &dynamodb.QueryInput{
			TableName:              aws.String(s.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: statePK(runID)},
				":sk": &types.AttributeValueMemberS{Value: statePrefix()},
			},
		}

		if lastEvaluatedKey != nil {
			queryInput.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := s.client.Query(ctx, queryInput)
		if err != nil {
			return nil, fmt.Errorf("failed to get all state: %w", err)
		}

		for _, item := range result.Items {
			skAttr, ok := item[AttrSK]
			if !ok {
				continue
			}

			sk := skAttr.(*types.AttributeValueMemberS).Value
			key := sk[len(statePrefix()):] // Remove STATE# prefix

			valueAttr, ok := item["value"]
			if !ok {
				continue
			}

			valueBytes := valueAttr.(*types.AttributeValueMemberB).Value
			stateData[key] = valueBytes
		}

		// Check if there are more results
		if result.LastEvaluatedKey == nil {
			break
		}
		lastEvaluatedKey = result.LastEvaluatedKey
	}

	return stateData, nil
}

// Query operations

func (s *DynamoDBStore) CountRunsByStatus(ctx context.Context, resourceID string, status workflow.RunStatus) (int, error) {
	// Query GSI2 with resourceID and status
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String(IndexResourceIndex),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: workflowRunGSI2PK(resourceID, string(status))},
		},
		Select: types.SelectCount,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count runs: %w", err)
	}

	return int(result.Count), nil
}
