package engine

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/require"
)

// Test input/output types used across engine tests

type DiscoverInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type DiscoverOutput struct {
	Companies []string `json:"companies"`
	Count     int      `json:"count"`
}

type EnrichInput struct {
	Companies []string `json:"companies"`
}

type EnrichOutput struct {
	Enriched map[string]any `json:"enriched"`
}

type FilterInput struct {
	Data map[string]any `json:"data"`
}

type FilterOutput struct {
	Filtered []string `json:"filtered"`
}

// createTestEngine creates a test engine with memory store and configured logger
func createTestEngine(t *testing.T) (*Engine, gorkflow.WorkflowStore) {
	t.Helper()
	wfStore := store.NewMemoryStore()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	engine := NewEngine(wfStore,
		WithLogger(logger),
		WithConfig(gorkflow.EngineConfig{
			MaxConcurrentWorkflows: 10,
			DefaultTimeout:         5 * time.Minute,
		}),
	)
	return engine, wfStore
}

// waitForCompletion waits for a workflow to reach a terminal state
func waitForCompletion(t *testing.T, engine *Engine, runID string, timeout time.Duration) *gorkflow.WorkflowRun {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for workflow completion")
		case <-ticker.C:
			run, err := engine.GetRun(context.Background(), runID)
			require.NoError(t, err)

			if run.Status.IsTerminal() {
				return run
			}
		}
	}
}

// Test step handlers used across engine tests

func discoverCompanies(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
	ctx.Logger.Info().
		Str("query", input.Query).
		Int("limit", input.Limit).
		Msg("Discovering companies")

	companies := []string{"CompanyA", "CompanyB", "CompanyC"}
	return DiscoverOutput{
		Companies: companies,
		Count:     len(companies),
	}, nil
}

func enrichCompanies(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
	discoverResult, err := gorkflow.GetOutput[DiscoverOutput](ctx, "discover")
	if err != nil {
		return EnrichOutput{}, err
	}

	ctx.Logger.Info().
		Int("companies_count", len(discoverResult.Companies)).
		Msg("Enriching companies")

	enriched := make(map[string]any)
	for _, company := range discoverResult.Companies {
		enriched[company] = map[string]any{
			"name":     company,
			"size":     "medium",
			"industry": "tech",
		}
	}

	return EnrichOutput{Enriched: enriched}, nil
}

func filterCompanies(ctx *gorkflow.StepContext, input FilterInput) (FilterOutput, error) {
	enrichResult, err := gorkflow.GetOutput[EnrichOutput](ctx, "enrich")
	if err != nil {
		return FilterOutput{}, err
	}

	ctx.Logger.Info().
		Int("companies_count", len(enrichResult.Enriched)).
		Msg("Filtering companies")

	filtered := []string{"CompanyA", "CompanyB"}
	return FilterOutput{Filtered: filtered}, nil
}
