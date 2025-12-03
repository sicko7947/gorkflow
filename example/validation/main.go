package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rs/zerolog"
	workflow "github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
)

// Input types with validation tags
type UserRegistrationInput struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
	Age      int    `json:"age" validate:"required,gte=18,lte=120"`
	Password string `json:"password" validate:"required,min=8"`
}

type ValidatedUserOutput struct {
	UserID   string `json:"userId" validate:"required,uuid4"`
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required"`
	IsActive bool   `json:"isActive"`
}

type EmailSentOutput struct {
	MessageID string `json:"messageId" validate:"required"`
	Status    string `json:"status" validate:"required,oneof=sent failed pending"`
	SentTo    string `json:"sentTo" validate:"required,email"`
}

type WorkflowResult struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	EmailSent bool   `json:"emailSent"`
}

// Step 1: Validate and create user
func NewValidateUserStep() *workflow.Step[UserRegistrationInput, ValidatedUserOutput] {
	return workflow.NewStep(
		"validate_user",
		"Validate and Create User",
		func(ctx *workflow.StepContext, input UserRegistrationInput) (ValidatedUserOutput, error) {
			ctx.Logger.Info().
				Str("email", input.Email).
				Str("username", input.Username).
				Msg("Creating user")

			// Simulate user creation
			userID := "550e8400-e29b-41d4-a716-446655440000" // UUID v4

			return ValidatedUserOutput{
				UserID:   userID,
				Email:    input.Email,
				Username: input.Username,
				IsActive: true,
			}, nil
		},
		// Validation is enabled by default - no need to specify!
	)
}

// Step 2: Send welcome email
func NewSendEmailStep() *workflow.Step[ValidatedUserOutput, EmailSentOutput] {
	return workflow.NewStep(
		"send_email",
		"Send Welcome Email",
		func(ctx *workflow.StepContext, input ValidatedUserOutput) (EmailSentOutput, error) {
			ctx.Logger.Info().
				Str("email", input.Email).
				Str("userId", input.UserID).
				Msg("Sending welcome email")

			// Simulate email sending
			return EmailSentOutput{
				MessageID: "msg-12345",
				Status:    "sent", // Must be one of: sent, failed, pending
				SentTo:    input.Email,
			}, nil
		},
		// Validation is enabled by default
	)
}

// Step 3: Format final result
func NewFormatResultStep() *workflow.Step[EmailSentOutput, WorkflowResult] {
	return workflow.NewStep(
		"format_result",
		"Format Final Result",
		func(ctx *workflow.StepContext, input EmailSentOutput) (WorkflowResult, error) {
			// Get user data from previous step
			var userData ValidatedUserOutput
			if err := ctx.Outputs.GetOutput("validate_user", &userData); err != nil {
				return WorkflowResult{}, fmt.Errorf("failed to get user data: %w", err)
			}

			return WorkflowResult{
				UserID:    userData.UserID,
				Email:     userData.Email,
				EmailSent: input.Status == "sent",
			}, nil
		},
		// No validation needed for final output (optional)
	)
}

func main() {
	// Create logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create workflow
	wf, err := workflow.NewWorkflow("user_registration", "User Registration Workflow").
		WithVersion("1.0").
		Sequence(
			NewValidateUserStep(),
			NewSendEmailStep(),
			NewFormatResultStep(),
		).
		Build()

	if err != nil {
		log.Fatal("Failed to build workflow:", err)
	}

	// Create store and engine
	memStore := store.NewMemoryStore()
	eng := engine.NewEngine(memStore, engine.WithLogger(logger))

	ctx := context.Background()

	// Example 1: Valid input - should succeed
	fmt.Println("\n=== Example 1: Valid Input ===")
	validInput := UserRegistrationInput{
		Email:    "john.doe@example.com",
		Username: "johndoe",
		Age:      25,
		Password: "SecurePass123",
	}

	runID1, err := eng.StartWorkflow(ctx, wf, validInput, workflow.WithSynchronousExecution())
	if err != nil {
		logger.Error().Err(err).Msg("Workflow failed")
	} else {
		logger.Info().Str("run_id", runID1).Msg("Workflow completed successfully")

		// Get result
		run, _ := eng.GetRun(ctx, runID1)
		logger.Info().
			Str("status", string(run.Status)).
			RawJSON("output", run.Output).
			Msg("Workflow result")
	}

	// Example 2: Invalid email - should fail validation
	fmt.Println("\n=== Example 2: Invalid Email ===")
	invalidEmailInput := UserRegistrationInput{
		Email:    "not-an-email",
		Username: "johndoe",
		Age:      25,
		Password: "SecurePass123",
	}

	runID2, err := eng.StartWorkflow(ctx, wf, invalidEmailInput, workflow.WithSynchronousExecution())
	if err != nil {
		logger.Error().Err(err).Msg("Workflow failed as expected")
	} else {
		run, _ := eng.GetRun(ctx, runID2)
		logger.Info().
			Str("status", string(run.Status)).
			Msg("Workflow status")
	}

	// Example 3: Username too short - should fail validation
	fmt.Println("\n=== Example 3: Username Too Short ===")
	invalidUsernameInput := UserRegistrationInput{
		Email:    "jane@example.com",
		Username: "ab", // Less than 3 characters
		Age:      30,
		Password: "SecurePass123",
	}

	runID3, err := eng.StartWorkflow(ctx, wf, invalidUsernameInput, workflow.WithSynchronousExecution())
	if err != nil {
		logger.Error().Err(err).Msg("Workflow failed as expected")
	} else {
		run, _ := eng.GetRun(ctx, runID3)
		logger.Info().
			Str("status", string(run.Status)).
			Msg("Workflow status")
	}

	// Example 4: Age below minimum - should fail validation
	fmt.Println("\n=== Example 4: Age Below Minimum ===")
	invalidAgeInput := UserRegistrationInput{
		Email:    "teen@example.com",
		Username: "teenager",
		Age:      16, // Below 18
		Password: "SecurePass123",
	}

	runID4, err := eng.StartWorkflow(ctx, wf, invalidAgeInput, workflow.WithSynchronousExecution())
	if err != nil {
		logger.Error().Err(err).Msg("Workflow failed as expected")
	} else {
		run, _ := eng.GetRun(ctx, runID4)
		logger.Info().
			Str("status", string(run.Status)).
			Msg("Workflow status")
	}
}
