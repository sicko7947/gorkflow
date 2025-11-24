//go:build integration

package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sicko7947/gorkflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestTable creates a temporary DynamoDB table for integration testing
func createTestTable(ctx context.Context, client *dynamodb.Client, tableName string) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI1PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI1SK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI2PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI2SK"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GSI1"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("GSI1PK"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("GSI1SK"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String("GSI2"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("GSI2PK"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("GSI2SK"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Wait for table to be active
	waiter := dynamodb.NewTableExistsWaiter(client)
	return waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, 2*time.Minute)
}

// deleteTestTable deletes the temporary DynamoDB table
func deleteTestTable(ctx context.Context, client *dynamodb.Client, tableName string) error {
	_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	return err
}

// setupIntegrationTest creates a test table and returns a store instance
func setupIntegrationTest(t *testing.T) (*DynamoDBStore, *dynamodb.Client, string, func()) {
	ctx := context.Background()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err, "Failed to load AWS config")

	client := dynamodb.NewFromConfig(cfg)

	// Create unique table name with timestamp
	tableName := fmt.Sprintf("workflow-integration-test-%d", time.Now().Unix())

	// Create table
	err = createTestTable(ctx, client, tableName)
	require.NoError(t, err, "Failed to create test table")

	t.Logf("Created test table: %s", tableName)

	// Create store
	store := NewDynamoDBStore(client, tableName).(*DynamoDBStore)

	// Return cleanup function
	cleanup := func() {
		err := deleteTestTable(context.Background(), client, tableName)
		if err != nil {
			t.Logf("Warning: Failed to delete test table %s: %v", tableName, err)
		} else {
			t.Logf("Deleted test table: %s", tableName)
		}
	}

	return store, client, tableName, cleanup
}

func TestIntegration_CreateAndGetRun(t *testing.T) {
	store, _, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

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

	// Create run
	err := store.CreateRun(ctx, run)
	require.NoError(t, err, "Failed to create run")

	// Get run
	retrieved, err := store.GetRun(ctx, run.RunID)
	require.NoError(t, err, "Failed to get run")

	assert.Equal(t, run.RunID, retrieved.RunID)
	assert.Equal(t, run.WorkflowID, retrieved.WorkflowID)
	assert.Equal(t, run.ResourceID, retrieved.ResourceID)
	assert.Equal(t, run.Status, retrieved.Status)
}

func TestIntegration_UpdateRunWithTransaction(t *testing.T) {
	store, _, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-2",
		WorkflowID: "test-workflow",
		ResourceID: "resource-1",
		Status:     workflow.RunStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Create run
	err := store.CreateRun(ctx, run)
	require.NoError(t, err)

	// Update run status (uses transaction)
	run.Status = workflow.RunStatusRunning
	err = store.UpdateRun(ctx, run)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.GetRun(ctx, run.RunID)
	require.NoError(t, err)
	assert.Equal(t, workflow.RunStatusRunning, retrieved.Status)
}

func TestIntegration_ListStepExecutions_Pagination(t *testing.T) {
	store, _, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	runID := "test-run-pagination"

	// Create run first
	run := &workflow.WorkflowRun{
		RunID:      runID,
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err := store.CreateRun(ctx, run)
	require.NoError(t, err)

	// Create 15 step executions (more than typical page size)
	for i := 0; i < 15; i++ {
		exec := &workflow.StepExecution{
			RunID:     runID,
			StepID:    fmt.Sprintf("step-%02d", i),
			Status:    workflow.StepStatusCompleted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := store.CreateStepExecution(ctx, exec)
		require.NoError(t, err, "Failed to create step execution %d", i)
	}

	// List all - should handle pagination automatically
	executions, err := store.ListStepExecutions(ctx, runID)
	require.NoError(t, err, "Failed to list step executions")
	assert.Len(t, executions, 15, "Should return all 15 executions")
}

func TestIntegration_GetAllState_Pagination(t *testing.T) {
	store, _, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	runID := "test-run-state"

	// Create run first
	run := &workflow.WorkflowRun{
		RunID:      runID,
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err := store.CreateRun(ctx, run)
	require.NoError(t, err)

	// Create 20 state entries
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("state-key-%02d", i)
		value := []byte(fmt.Sprintf(`{"index": %d}`, i))
		err := store.SaveState(ctx, runID, key, value)
		require.NoError(t, err, "Failed to save state %d", i)
	}

	// Get all state - should handle pagination automatically
	allState, err := store.GetAllState(ctx, runID)
	require.NoError(t, err, "Failed to get all state")
	assert.Len(t, allState, 20, "Should return all 20 state entries")
}

func TestIntegration_CountRunsByStatus_WithGSI2(t *testing.T) {
	store, _, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	resourceID := "resource-test"

	// Create runs with different statuses
	statuses := []workflow.RunStatus{
		workflow.RunStatusPending,
		workflow.RunStatusPending,
		workflow.RunStatusRunning,
		workflow.RunStatusCompleted,
	}

	for i, status := range statuses {
		run := &workflow.WorkflowRun{
			RunID:      fmt.Sprintf("run-%d", i),
			WorkflowID: "test-workflow",
			ResourceID: resourceID,
			Status:     status,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := store.CreateRun(ctx, run)
		require.NoError(t, err, "Failed to create run %d", i)
	}

	// Wait a moment for GSI to update
	time.Sleep(2 * time.Second)

	// Count pending runs
	count, err := store.CountRunsByStatus(ctx, resourceID, workflow.RunStatusPending)
	require.NoError(t, err, "Failed to count pending runs")
	assert.Equal(t, 2, count, "Should have 2 pending runs")

	// Count running runs
	count, err = store.CountRunsByStatus(ctx, resourceID, workflow.RunStatusRunning)
	require.NoError(t, err, "Failed to count running runs")
	assert.Equal(t, 1, count, "Should have 1 running run")

	// Count completed runs
	count, err = store.CountRunsByStatus(ctx, resourceID, workflow.RunStatusCompleted)
	require.NoError(t, err, "Failed to count completed runs")
	assert.Equal(t, 1, count, "Should have 1 completed run")
}

func TestIntegration_StepOutputOperations(t *testing.T) {
	store, _, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	runID := "test-run-output"
	stepID := "step-1"

	// Create run first
	run := &workflow.WorkflowRun{
		RunID:      runID,
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err := store.CreateRun(ctx, run)
	require.NoError(t, err)

	// Save step output
	output := []byte(`{"result": "success", "data": {"count": 42}}`)
	err = store.SaveStepOutput(ctx, runID, stepID, output)
	require.NoError(t, err, "Failed to save step output")

	// Load step output
	loaded, err := store.LoadStepOutput(ctx, runID, stepID)
	require.NoError(t, err, "Failed to load step output")
	assert.Equal(t, output, loaded, "Output should match")
}
