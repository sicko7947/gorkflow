package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sicko7947/gorkflow"
)

// mockDynamoDBClient implements DynamoDBClient interface for testing
type mockDynamoDBClient struct {
	putItemFunc            func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	getItemFunc            func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	queryFunc              func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	deleteItemFunc         func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	transactWriteItemsFunc func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

func (m *mockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.putItemFunc != nil {
		return m.putItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if m.getItemFunc != nil {
		return m.getItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.GetItemOutput{}, nil
}

func (m *mockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, params, optFns...)
	}
	return &dynamodb.QueryOutput{}, nil
}

func (m *mockDynamoDBClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if m.deleteItemFunc != nil {
		return m.deleteItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.DeleteItemOutput{}, nil
}

func (m *mockDynamoDBClient) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	if m.transactWriteItemsFunc != nil {
		return m.transactWriteItemsFunc(ctx, params, optFns...)
	}
	return &dynamodb.TransactWriteItemsOutput{}, nil
}

func TestNewDynamoDBStore(t *testing.T) {
	client := &mockDynamoDBClient{}
	store := NewDynamoDBStore(client, "test-table")

	if store == nil {
		t.Fatal("NewDynamoDBStore() returned nil")
	}

	// Verify it implements the interface
	var _ workflow.WorkflowStore = store
}

func TestDynamoDBStore_CreateRun(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = params
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table").(*DynamoDBStore)
	ctx := context.Background()

	now := time.Now()
	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		ResourceID: "resource-1",
		Status:     workflow.RunStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := store.CreateRun(ctx, run)
	if err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	// Verify the correct table name
	if capturedInput == nil {
		t.Fatal("PutItem was not called")
	}

	if *capturedInput.TableName != "test-table" {
		t.Errorf("TableName = %s, want test-table", *capturedInput.TableName)
	}

	// Check PK
	pk, ok := capturedInput.Item[AttrPK]
	if !ok {
		t.Fatal("PK not set")
	}
	pkValue := pk.(*types.AttributeValueMemberS).Value
	expectedPK := workflowRunPK(run.RunID)
	if pkValue != expectedPK {
		t.Errorf("PK = %s, want %s", pkValue, expectedPK)
	}

	// Check SK
	sk, ok := capturedInput.Item[AttrSK]
	if !ok {
		t.Fatal("SK not set")
	}
	skValue := sk.(*types.AttributeValueMemberS).Value
	expectedSK := workflowRunSK()
	if skValue != expectedSK {
		t.Errorf("SK = %s, want %s", skValue, expectedSK)
	}

	// Check entity type
	entityType, ok := capturedInput.Item[AttrEntityType]
	if !ok {
		t.Fatal("EntityType not set")
	}
	entityTypeValue := entityType.(*types.AttributeValueMemberS).Value
	if entityTypeValue != EntityTypeWorkflowRun {
		t.Errorf("EntityType = %s, want %s", entityTypeValue, EntityTypeWorkflowRun)
	}

	// Check GSI1PK is set
	if _, ok := capturedInput.Item[AttrGSI1PK]; !ok {
		t.Error("GSI1PK not set")
	}

	// Check GSI2PK is set
	if _, ok := capturedInput.Item[AttrGSI2PK]; !ok {
		t.Error("GSI2PK not set")
	}
}

func TestDynamoDBStore_CreateRun_Error(t *testing.T) {
	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("dynamodb error")
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.CreateRun(ctx, run)
	if err == nil {
		t.Error("CreateRun() should have failed with DynamoDB error")
	}
}

func TestDynamoDBStore_GetRun(t *testing.T) {
	now := time.Now()
	runID := "test-run-1"

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			// Return a complete mock item matching WorkflowRun struct with DynamoDB attribute names (snake_case)
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"run_id":           &types.AttributeValueMemberS{Value: runID},
					"workflow_id":      &types.AttributeValueMemberS{Value: "test-workflow"},
					"workflow_version": &types.AttributeValueMemberS{Value: ""},
					"resource_id":      &types.AttributeValueMemberS{Value: ""},
					"status":           &types.AttributeValueMemberS{Value: string(workflow.RunStatusPending)},
					"progress":         &types.AttributeValueMemberN{Value: "0"},
					"created_at":       &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
					"updated_at":       &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
					"current_step":     &types.AttributeValueMemberS{Value: ""},
				},
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	run, err := store.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetRun() failed: %v", err)
	}

	if run.RunID != runID {
		t.Errorf("RunID = %s, want %s", run.RunID, runID)
	}

	if run.Status != workflow.RunStatusPending {
		t.Errorf("Status = %s, want %s", run.Status, workflow.RunStatusPending)
	}
}

func TestDynamoDBStore_GetRun_NotFound(t *testing.T) {
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.GetRun(ctx, "non-existent")
	if err == nil {
		t.Error("GetRun() should have failed for non-existent run")
	}
}

func TestDynamoDBStore_GetRun_Error(t *testing.T) {
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("dynamodb error")
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.GetRun(ctx, "test-run")
	if err == nil {
		t.Error("GetRun() should have failed with DynamoDB error")
	}
}

func TestDynamoDBStore_UpdateRun(t *testing.T) {
	var capturedInput *dynamodb.TransactWriteItemsInput

	client := &mockDynamoDBClient{
		transactWriteItemsFunc: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			capturedInput = params
			return &dynamodb.TransactWriteItemsOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	now := time.Now()
	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusRunning,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := store.UpdateRun(ctx, run)
	if err != nil {
		t.Fatalf("UpdateRun() failed: %v", err)
	}

	if capturedInput == nil {
		t.Fatal("TransactWriteItems was not called")
	}

	// Verify UpdatedAt was set to a recent time
	if run.UpdatedAt.Before(now) {
		t.Error("UpdatedAt should have been updated")
	}

	// Verify transaction contains one item
	if len(capturedInput.TransactItems) != 1 {
		t.Errorf("TransactWriteItems should contain 1 item, got %d", len(capturedInput.TransactItems))
	}
}

func TestDynamoDBStore_UpdateRunStatus(t *testing.T) {
	now := time.Now()
	runID := "test-run-1"

	var capturedGetInput *dynamodb.GetItemInput
	var capturedTransactInput *dynamodb.TransactWriteItemsInput

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			capturedGetInput = params
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"RunID":      &types.AttributeValueMemberS{Value: runID},
					"WorkflowID": &types.AttributeValueMemberS{Value: "test-workflow"},
					"Status":     &types.AttributeValueMemberS{Value: string(workflow.RunStatusRunning)},
					"CreatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
					"UpdatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
				},
			}, nil
		},
		transactWriteItemsFunc: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			capturedTransactInput = params
			return &dynamodb.TransactWriteItemsOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	wfErr := &workflow.WorkflowError{
		Code:    "TEST_ERROR",
		Message: "test error",
	}

	err := store.UpdateRunStatus(ctx, runID, workflow.RunStatusFailed, wfErr)
	if err != nil {
		t.Fatalf("UpdateRunStatus() failed: %v", err)
	}

	if capturedGetInput == nil {
		t.Fatal("GetItem was not called")
	}

	if capturedTransactInput == nil {
		t.Fatal("TransactWriteItems was not called")
	}

	// Verify the correct keys were used for GetItem
	pk := capturedGetInput.Key[AttrPK].(*types.AttributeValueMemberS).Value
	expectedPK := workflowRunPK(runID)
	if pk != expectedPK {
		t.Errorf("GetItem PK = %s, want %s", pk, expectedPK)
	}
}

func TestDynamoDBStore_UpdateRunStatus_TerminalStatus(t *testing.T) {
	now := time.Now()
	runID := "test-run-1"

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"RunID":      &types.AttributeValueMemberS{Value: runID},
					"WorkflowID": &types.AttributeValueMemberS{Value: "test-workflow"},
					"Status":     &types.AttributeValueMemberS{Value: string(workflow.RunStatusRunning)},
					"CreatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
					"UpdatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
				},
			}, nil
		},
		transactWriteItemsFunc: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			return &dynamodb.TransactWriteItemsOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	// Test with terminal status (Completed)
	err := store.UpdateRunStatus(ctx, runID, workflow.RunStatusCompleted, nil)
	if err != nil {
		t.Fatalf("UpdateRunStatus() failed: %v", err)
	}
}

func TestDynamoDBStore_ListRuns(t *testing.T) {
	client := &mockDynamoDBClient{}
	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	// ListRuns is not fully implemented (returns empty list)
	runs, err := store.ListRuns(ctx, workflow.RunFilter{})
	if err != nil {
		t.Fatalf("ListRuns() failed: %v", err)
	}

	if len(runs) != 0 {
		t.Errorf("ListRuns() returned %d runs, want 0 (not fully implemented)", len(runs))
	}
}

func TestDynamoDBStore_CreateStepExecution(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = params
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	exec := &workflow.StepExecution{
		RunID:     "test-run-1",
		StepID:    "step-1",
		Status:    workflow.StepStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.CreateStepExecution(ctx, exec)
	if err != nil {
		t.Fatalf("CreateStepExecution() failed: %v", err)
	}

	if capturedInput == nil {
		t.Fatal("PutItem was not called")
	}

	// Verify keys
	pk := capturedInput.Item[AttrPK].(*types.AttributeValueMemberS).Value
	expectedPK := stepExecutionPK(exec.RunID)
	if pk != expectedPK {
		t.Errorf("PK = %s, want %s", pk, expectedPK)
	}

	sk := capturedInput.Item[AttrSK].(*types.AttributeValueMemberS).Value
	expectedSK := stepExecutionSK(exec.StepID)
	if sk != expectedSK {
		t.Errorf("SK = %s, want %s", sk, expectedSK)
	}

	entityType := capturedInput.Item[AttrEntityType].(*types.AttributeValueMemberS).Value
	if entityType != EntityTypeStepExecution {
		t.Errorf("EntityType = %s, want %s", entityType, EntityTypeStepExecution)
	}
}

func TestDynamoDBStore_GetStepExecution(t *testing.T) {
	runID := "test-run-1"
	stepID := "step-1"

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			now := time.Now()
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"run_id":          &types.AttributeValueMemberS{Value: runID},
					"step_id":         &types.AttributeValueMemberS{Value: stepID},
					"execution_index": &types.AttributeValueMemberN{Value: "0"},
					"status":          &types.AttributeValueMemberS{Value: string(workflow.StepStatusPending)},
					"duration_ms":     &types.AttributeValueMemberN{Value: "0"},
					"created_at":      &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
					"updated_at":      &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
				},
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	exec, err := store.GetStepExecution(ctx, runID, stepID)
	if err != nil {
		t.Fatalf("GetStepExecution() failed: %v", err)
	}

	if exec.StepID != stepID {
		t.Errorf("StepID = %s, want %s", exec.StepID, stepID)
	}
}

func TestDynamoDBStore_GetStepExecution_NotFound(t *testing.T) {
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.GetStepExecution(ctx, "non-existent-run", "non-existent-step")
	if err == nil {
		t.Error("GetStepExecution() should have failed for non-existent execution")
	}
}

func TestDynamoDBStore_UpdateStepExecution(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = params
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	now := time.Now()
	exec := &workflow.StepExecution{
		RunID:     "test-run-1",
		StepID:    "step-1",
		Status:    workflow.StepStatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.UpdateStepExecution(ctx, exec)
	if err != nil {
		t.Fatalf("UpdateStepExecution() failed: %v", err)
	}

	if capturedInput == nil {
		t.Fatal("PutItem was not called")
	}

	// Verify UpdatedAt was updated
	if exec.UpdatedAt.Before(now) {
		t.Error("UpdatedAt should have been updated")
	}
}

func TestDynamoDBStore_ListStepExecutions(t *testing.T) {
	runID := "test-run-1"

	client := &mockDynamoDBClient{
		queryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"RunID":     &types.AttributeValueMemberS{Value: runID},
						"StepID":    &types.AttributeValueMemberS{Value: "step-1"},
						"Status":    &types.AttributeValueMemberS{Value: string(workflow.StepStatusPending)},
						"CreatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339Nano)},
						"UpdatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339Nano)},
					},
					{
						"RunID":     &types.AttributeValueMemberS{Value: runID},
						"StepID":    &types.AttributeValueMemberS{Value: "step-2"},
						"Status":    &types.AttributeValueMemberS{Value: string(workflow.StepStatusCompleted)},
						"CreatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339Nano)},
						"UpdatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339Nano)},
					},
				},
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	executions, err := store.ListStepExecutions(ctx, runID)
	if err != nil {
		t.Fatalf("ListStepExecutions() failed: %v", err)
	}

	if len(executions) != 2 {
		t.Errorf("ListStepExecutions() returned %d executions, want 2", len(executions))
	}
}

func TestDynamoDBStore_ListStepExecutions_Error(t *testing.T) {
	client := &mockDynamoDBClient{
		queryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("dynamodb error")
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.ListStepExecutions(ctx, "test-run")
	if err == nil {
		t.Error("ListStepExecutions() should have failed with DynamoDB error")
	}
}

func TestDynamoDBStore_SaveStepOutput(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = params
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	output := []byte(`{"result": "success"}`)

	err := store.SaveStepOutput(ctx, "test-run-1", "step-1", output)
	if err != nil {
		t.Fatalf("SaveStepOutput() failed: %v", err)
	}

	if capturedInput == nil {
		t.Fatal("PutItem was not called")
	}

	// Verify keys
	pk := capturedInput.Item[AttrPK].(*types.AttributeValueMemberS).Value
	expectedPK := stepOutputPK("test-run-1")
	if pk != expectedPK {
		t.Errorf("PK = %s, want %s", pk, expectedPK)
	}

	sk := capturedInput.Item[AttrSK].(*types.AttributeValueMemberS).Value
	expectedSK := stepOutputSK("step-1")
	if sk != expectedSK {
		t.Errorf("SK = %s, want %s", sk, expectedSK)
	}

	entityType := capturedInput.Item[AttrEntityType].(*types.AttributeValueMemberS).Value
	if entityType != EntityTypeStepOutput {
		t.Errorf("EntityType = %s, want %s", entityType, EntityTypeStepOutput)
	}

	// Verify output is stored
	if _, ok := capturedInput.Item["output"]; !ok {
		t.Error("output field not set")
	}
}

func TestDynamoDBStore_LoadStepOutput(t *testing.T) {
	output := []byte(`{"result": "success"}`)

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"output": &types.AttributeValueMemberB{Value: output},
				},
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	loaded, err := store.LoadStepOutput(ctx, "test-run-1", "step-1")
	if err != nil {
		t.Fatalf("LoadStepOutput() failed: %v", err)
	}

	if string(loaded) != string(output) {
		t.Errorf("LoadStepOutput() = %s, want %s", string(loaded), string(output))
	}
}

func TestDynamoDBStore_LoadStepOutput_NotFound(t *testing.T) {
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.LoadStepOutput(ctx, "test-run", "step-1")
	if err == nil {
		t.Error("LoadStepOutput() should have failed for non-existent output")
	}
}

func TestDynamoDBStore_LoadStepOutput_NoOutputField(t *testing.T) {
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"other_field": &types.AttributeValueMemberS{Value: "value"},
				},
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.LoadStepOutput(ctx, "test-run", "step-1")
	if err == nil {
		t.Error("LoadStepOutput() should have failed when output field is missing")
	}
}

func TestDynamoDBStore_SaveState(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = params
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	value := []byte(`{"key": "value"}`)

	err := store.SaveState(ctx, "test-run-1", "my-key", value)
	if err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	if capturedInput == nil {
		t.Fatal("PutItem was not called")
	}

	// Verify keys
	pk := capturedInput.Item[AttrPK].(*types.AttributeValueMemberS).Value
	expectedPK := statePK("test-run-1")
	if pk != expectedPK {
		t.Errorf("PK = %s, want %s", pk, expectedPK)
	}

	sk := capturedInput.Item[AttrSK].(*types.AttributeValueMemberS).Value
	expectedSK := stateSK("my-key")
	if sk != expectedSK {
		t.Errorf("SK = %s, want %s", sk, expectedSK)
	}

	entityType := capturedInput.Item[AttrEntityType].(*types.AttributeValueMemberS).Value
	if entityType != EntityTypeState {
		t.Errorf("EntityType = %s, want %s", entityType, EntityTypeState)
	}
}

func TestDynamoDBStore_LoadState(t *testing.T) {
	value := []byte(`{"key": "value"}`)

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"value": &types.AttributeValueMemberB{Value: value},
				},
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	loaded, err := store.LoadState(ctx, "test-run-1", "my-key")
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	if string(loaded) != string(value) {
		t.Errorf("LoadState() = %s, want %s", string(loaded), string(value))
	}
}

func TestDynamoDBStore_LoadState_NotFound(t *testing.T) {
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.LoadState(ctx, "test-run", "key")
	if err == nil {
		t.Error("LoadState() should have failed for non-existent state")
	}
}

func TestDynamoDBStore_DeleteState(t *testing.T) {
	var capturedInput *dynamodb.DeleteItemInput

	client := &mockDynamoDBClient{
		deleteItemFunc: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			capturedInput = params
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	err := store.DeleteState(ctx, "test-run-1", "my-key")
	if err != nil {
		t.Fatalf("DeleteState() failed: %v", err)
	}

	if capturedInput == nil {
		t.Fatal("DeleteItem was not called")
	}

	// Verify table name
	if *capturedInput.TableName != "test-table" {
		t.Errorf("TableName = %s, want test-table", *capturedInput.TableName)
	}

	// Verify keys
	pk := capturedInput.Key[AttrPK].(*types.AttributeValueMemberS).Value
	expectedPK := statePK("test-run-1")
	if pk != expectedPK {
		t.Errorf("PK = %s, want %s", pk, expectedPK)
	}

	sk := capturedInput.Key[AttrSK].(*types.AttributeValueMemberS).Value
	expectedSK := stateSK("my-key")
	if sk != expectedSK {
		t.Errorf("SK = %s, want %s", sk, expectedSK)
	}
}

func TestDynamoDBStore_GetAllState(t *testing.T) {
	runID := "test-run-1"

	client := &mockDynamoDBClient{
		queryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						AttrSK:  &types.AttributeValueMemberS{Value: "STATE#key1"},
						"value": &types.AttributeValueMemberB{Value: []byte(`{"value": 1}`)},
					},
					{
						AttrSK:  &types.AttributeValueMemberS{Value: "STATE#key2"},
						"value": &types.AttributeValueMemberB{Value: []byte(`{"value": 2}`)},
					},
				},
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	allState, err := store.GetAllState(ctx, runID)
	if err != nil {
		t.Fatalf("GetAllState() failed: %v", err)
	}

	if len(allState) != 2 {
		t.Errorf("GetAllState() returned %d items, want 2", len(allState))
	}

	if _, ok := allState["key1"]; !ok {
		t.Error("GetAllState() missing key1")
	}

	if _, ok := allState["key2"]; !ok {
		t.Error("GetAllState() missing key2")
	}
}

func TestDynamoDBStore_GetAllState_Error(t *testing.T) {
	client := &mockDynamoDBClient{
		queryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("dynamodb error")
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.GetAllState(ctx, "test-run")
	if err == nil {
		t.Error("GetAllState() should have failed with DynamoDB error")
	}
}

func TestDynamoDBStore_CountRunsByStatus(t *testing.T) {
	client := &mockDynamoDBClient{
		queryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			// Verify correct index is used
			if params.IndexName == nil || *params.IndexName != IndexResourceIndex {
				t.Errorf("IndexName = %v, want %s", params.IndexName, IndexResourceIndex)
			}

			// Verify Select is Count
			if params.Select != types.SelectCount {
				t.Errorf("Select = %v, want SelectCount", params.Select)
			}

			return &dynamodb.QueryOutput{
				Count: 3,
			}, nil
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	count, err := store.CountRunsByStatus(ctx, "resource-1", workflow.RunStatusPending)
	if err != nil {
		t.Fatalf("CountRunsByStatus() failed: %v", err)
	}

	if count != 3 {
		t.Errorf("CountRunsByStatus() = %d, want 3", count)
	}
}

func TestDynamoDBStore_CountRunsByStatus_Error(t *testing.T) {
	client := &mockDynamoDBClient{
		queryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("dynamodb error")
		},
	}

	store := NewDynamoDBStore(client, "test-table")
	ctx := context.Background()

	_, err := store.CountRunsByStatus(ctx, "resource-1", workflow.RunStatusPending)
	if err == nil {
		t.Error("CountRunsByStatus() should have failed with DynamoDB error")
	}
}
