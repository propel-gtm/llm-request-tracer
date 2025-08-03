package llmtracer

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	traceIDKey    contextKey = "llm_trace_id"
	userIDKey     contextKey = "llm_user_id"
	workflowKey   contextKey = "llm_workflow"
	featureKey    contextKey = "llm_feature"
	dimensionsKey contextKey = "llm_dimensions"
)

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithNewTraceID adds a new generated trace ID to the context
func WithNewTraceID(ctx context.Context) context.Context {
	return WithTraceID(ctx, uuid.New().String())
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithWorkflow adds a workflow type to the context
func WithWorkflow(ctx context.Context, workflow string) context.Context {
	return context.WithValue(ctx, workflowKey, workflow)
}

// WithFeature adds a feature name to the context
func WithFeature(ctx context.Context, feature string) context.Context {
	return context.WithValue(ctx, featureKey, feature)
}

// WithDimensions adds custom dimensions to the context
func WithDimensions(ctx context.Context, dimensions map[string]interface{}) context.Context {
	return context.WithValue(ctx, dimensionsKey, dimensions)
}

// GetTraceIDFromContext extracts trace ID from context, generates one if missing
func GetTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return uuid.New().String()
	}

	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		return traceID
	}

	return uuid.New().String()
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}

	return ""
}

// GetDimensionsFromContext extracts all tracking dimensions from context
func GetDimensionsFromContext(ctx context.Context) map[string]interface{} {
	dimensions := make(map[string]interface{})

	if ctx == nil {
		return dimensions
	}

	// Add explicit dimensions
	if customDims, ok := ctx.Value(dimensionsKey).(map[string]interface{}); ok {
		for k, v := range customDims {
			dimensions[k] = v
		}
	}

	// Add individual context values
	if userID := GetUserIDFromContext(ctx); userID != "" {
		dimensions["user_id"] = userID
	}

	if workflow, ok := ctx.Value(workflowKey).(string); ok && workflow != "" {
		dimensions["workflow"] = workflow
	}

	if feature, ok := ctx.Value(featureKey).(string); ok && feature != "" {
		dimensions["feature"] = feature
	}

	return dimensions
}
