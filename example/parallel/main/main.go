package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/example/parallel"
	"github.com/sicko7947/gorkflow/store"
)

// Shared state used by both local and Lambda deployments
var (
	wfEngine *engine.Engine
	workflow *gorkflow.Workflow
)

// ReadableStepExecution wraps StepExecution with decoded input/output
type ReadableStepExecution struct {
	*gorkflow.StepExecution
	Input  json.RawMessage `json:"input,omitempty"`
	Output json.RawMessage `json:"output,omitempty"`
}

// WorkflowStatus represents the current status of a workflow run
type WorkflowStatus struct {
	*gorkflow.WorkflowRun
	Input          json.RawMessage          `json:"input,omitempty"`
	StepExecutions []*ReadableStepExecution `json:"stepExecutions,omitempty"`
	Output         *parallel.FormatOutput   `json:"output,omitempty"`
}

// initializeApp performs common initialization for both deployment modes
func initializeApp() {
	var err error

	// Initialize workflow store
	workflowStore := store.NewMemoryStore()

	// Initialize simple math workflow
	workflow, err = parallel.NewSimpleMathWorkflow()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create simple math workflow")
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})

	// Initialize engine
	wfEngine = engine.NewEngine(
		workflowStore,
		engine.WithLogger(log.Logger),
		engine.WithConfig(engine.EngineConfig{
			MaxConcurrentWorkflows: 50,
			DefaultTimeout:         1 * time.Minute,
		}),
	)

	log.Info().Msg("Workflow engine initialized successfully")
}

// registerRoutes registers all HTTP routes
func registerRoutes(app *fiber.App) {
	// Health check endpoint
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "tendor-email-agent-simple-math",
			"version": "1.0.0",
		})
	})

	// Root endpoint
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service":     "Tendor Email Agent - Simple Math Workflow Server",
			"version":     "1.0.0",
			"framework":   "New Type-Safe Workflow Engine",
			"description": "Simple Math Workflow Example",
			"endpoints": fiber.Map{
				"health":         "GET /health",
				"startWorkflow":  "POST /api/v1/workflows/simple-math",
				"getStatus":      "GET /api/v1/workflows/:runId",
				"cancelWorkflow": "POST /api/v1/workflows/:runId/cancel",
			},
		})
	})

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Workflow endpoints
	workflows := v1.Group("/workflows")

	// Simple math workflow endpoints
	workflows.Post("/simple-math", handleStartWorkflow)
	workflows.Get("/:runId", handleGetStatus)
	workflows.Post("/:runId/cancel", handleCancelWorkflow)
}

// handleStartWorkflow starts a new simple math workflow
func handleStartWorkflow(c fiber.Ctx) error {
	var input parallel.WorkflowInput
	if err := c.Bind().JSON(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Start workflow execution
	runID, err := wfEngine.StartWorkflow(
		c.Context(),
		workflow,
		input,
		gorkflow.WithTags(map[string]string{
			"type": "simple_math",
		}),
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to start workflow")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start workflow",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"runId":   runID,
		"status":  "PENDING",
		"message": "Workflow started successfully",
	})
}

// handleGetStatus retrieves workflow status
func handleGetStatus(c fiber.Ctx) error {
	runID := c.Params("runId")

	// Get run metadata
	run, err := wfEngine.GetRun(c.Context(), runID)
	if err != nil {
		log.Error().Err(err).Str("runId", runID).Msg("Failed to get workflow run")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// Get step executions for detailed progress
	stepExecs, err := wfEngine.GetStepExecutions(c.Context(), runID)
	if err != nil {
		log.Error().Err(err).Str("runId", runID).Msg("Failed to get step executions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get step executions",
		})
	}

	// Convert to readable step executions
	readableSteps := make([]*ReadableStepExecution, len(stepExecs))
	for i, step := range stepExecs {
		readableSteps[i] = &ReadableStepExecution{
			StepExecution: step,
		}
		if len(step.Input) > 0 {
			readableSteps[i].Input = json.RawMessage(step.Input)
		}
		if len(step.Output) > 0 {
			readableSteps[i].Output = json.RawMessage(step.Output)
		}
	}

	status := &WorkflowStatus{
		WorkflowRun:    run,
		StepExecutions: readableSteps,
	}
	if len(run.Input) > 0 {
		status.Input = json.RawMessage(run.Input)
	}

	// If completed, parse output
	if run.Status == gorkflow.RunStatusCompleted && len(run.Output) > 0 {
		var output parallel.FormatOutput
		if err := json.Unmarshal(run.Output, &output); err != nil {
			log.Warn().
				Err(err).
				Str("run_id", runID).
				Msg("Failed to parse workflow output")
		} else {
			status.Output = &output
		}
	}

	return c.JSON(status)
}

// handleCancelWorkflow cancels a running workflow
func handleCancelWorkflow(c fiber.Ctx) error {
	runID := c.Params("runId")

	err := wfEngine.Cancel(c.Context(), runID)
	if err != nil {
		log.Error().Err(err).Str("runId", runID).Msg("Failed to cancel workflow")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to cancel workflow",
		})
	}

	return c.JSON(fiber.Map{
		"runId":   runID,
		"status":  "CANCELLED",
		"message": "Workflow cancelled successfully",
	})
}

func main() {
	// Initialize shared components
	initializeApp()

	// Create Fiber app with routes
	app := fiber.New()

	// Register routes
	registerRoutes(app)

	// Start server in a goroutine
	go func() {
		addr := ":3000"
		log.Info().Str("address", addr).Msg("Starting HTTP server")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}
