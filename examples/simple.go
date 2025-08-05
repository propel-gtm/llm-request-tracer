package main

import (
	"context"
	"fmt"
	"log"

	"github.com/anthropics/anthropic-sdk-go"
	mistral "github.com/gage-technologies/mistral-go"
	"github.com/google/generative-ai-go/genai"
	llmtracer "github.com/propel-gtm/llm-request-tracer"
	"github.com/propel-gtm/llm-request-tracer/adapters"
	"github.com/sashabaranov/go-openai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Setup token tracking storage (one time)
	db, err := gorm.Open(sqlite.Open("token_usage.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	storage, err := adapters.NewGormAdapter(db)
	if err != nil {
		log.Fatal("Failed to create storage adapter:", err)
	}

	// Create the unified AI client
	client := llmtracer.NewClient(storage)
	defer client.Close()

	// Create your own AI clients
	openaiClient := openai.NewClient("your-openai-api-key-here")
	anthropicClient := anthropic.NewClient()
	mistralClient := mistral.NewMistralClientDefault("your-mistral-api-key-here")

	// Google client requires more setup
	googleClient, err := genai.NewClient(context.Background())
	if err != nil {
		log.Fatal("Failed to create Google client:", err)
	}
	defer googleClient.Close()
	googleModel := googleClient.GenerativeModel("gemini-1.5-flash")

	// Note: Set ANTHROPIC_API_KEY and GOOGLE_API_KEY environment variables

	// Example 1: Call OpenAI with automatic tracking
	ctx := context.Background()
	ctx = llmtracer.WithUserID(ctx, "user-123")
	ctx = llmtracer.WithFeature(ctx, "geography-quiz")

	response, err := client.TraceOpenAIRequest(ctx, openai.ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
			{Role: openai.ChatMessageRoleUser, Content: "What is the capital of France?"},
		},
	}, openaiClient.CreateChatCompletion)
	if err != nil {
		log.Printf("OpenAI error: %v", err)
	} else if len(response.Choices) > 0 {
		fmt.Printf("OpenAI Response: %s\n", response.Choices[0].Message.Content)
	}

	// Example 2: Call Anthropic with automatic tracking
	ctx = llmtracer.WithUserID(context.Background(), "user-123")
	ctx = llmtracer.WithFeature(ctx, "creative-writing")

	anthropicResponse, err := client.TraceAnthropicRequest(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5SonnetLatest,
		MaxTokens: 1000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock("Write a haiku about programming"),
			),
		},
	}, anthropicClient.Messages.New)
	if err != nil {
		log.Printf("Anthropic error: %v", err)
	} else if len(anthropicResponse.Content) > 0 {
		textBlock := anthropicResponse.Content[0].AsText()
		fmt.Printf("Anthropic Response: %s\n", textBlock.Text)
	}

	// Example 3: Call Mistral with automatic tracking
	ctx = llmtracer.WithUserID(context.Background(), "user-123")
	ctx = llmtracer.WithFeature(ctx, "code-help")

	mistralResponse, err := client.TraceMistralRequest(ctx, mistral.ModelMistralLargeLatest, []mistral.ChatMessage{
		{Role: mistral.RoleSystem, Content: "You are a helpful coding assistant."},
		{Role: mistral.RoleUser, Content: "Write a simple hello world function in Go"},
	}, &mistral.ChatRequestParams{
		MaxTokens:   1000,
		Temperature: 0.7,
	}, mistralClient.Chat)
	if err != nil {
		log.Printf("Mistral error: %v", err)
	} else if len(mistralResponse.Choices) > 0 {
		fmt.Printf("Mistral Response: %s\n", mistralResponse.Choices[0].Message.Content)
	}

	// Example 4: Call Google Generative AI with automatic tracking
	ctx = llmtracer.WithUserID(context.Background(), "user-123")
	ctx = llmtracer.WithFeature(ctx, "creative-writing")

	googleResponse, err := client.TraceGoogleRequest(ctx, "gemini-1.5-flash", []genai.Part{
		genai.Text("Write a short poem about artificial intelligence"),
	}, googleModel.GenerateContent)
	if err != nil {
		log.Printf("Google error: %v", err)
	} else if len(googleResponse.Candidates) > 0 && len(googleResponse.Candidates[0].Content.Parts) > 0 {
		if text, ok := googleResponse.Candidates[0].Content.Parts[0].(genai.Text); ok {
			fmt.Printf("Google Response: %s\n", text)
		}
	}

	// Example 5: Another OpenAI call with different context
	ctx = llmtracer.WithUserID(context.Background(), "user-123")
	ctx = llmtracer.WithFeature(ctx, "math-help")

	response, err = client.TraceOpenAIRequest(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
			{Role: openai.ChatMessageRoleUser, Content: "What is 2+2?"},
		},
	}, openaiClient.CreateChatCompletion)
	if err != nil {
		log.Printf("OpenAI error: %v", err)
	} else if len(response.Choices) > 0 {
		fmt.Printf("OpenAI Response: %s\n", response.Choices[0].Message.Content)
	}

	// Get token usage statistics
	stats, err := client.GetTokenStats(context.Background(), nil)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		return
	}

	fmt.Println("\n=== Token Usage Stats ===")
	for model, stat := range stats {
		fmt.Printf("%s:\n", model)
		fmt.Printf("  Total Requests: %d\n", stat.TotalRequests)
		fmt.Printf("  Input Tokens: %d\n", stat.InputTokens)
		fmt.Printf("  Output Tokens: %d\n", stat.OutputTokens)
		if stat.ErrorCount > 0 {
			fmt.Printf("  Errors: %d\n", stat.ErrorCount)
		}
		fmt.Println()
	}
}
