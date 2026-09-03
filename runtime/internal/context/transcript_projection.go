package context

import (
	"strings"

	"gameagent/runtime/internal/model"
)

func projectCurrentTurnTranscript(messages []model.Message, bounds projectionBounds) []model.Message {
	if len(messages) == 0 {
		return nil
	}

	out := make([]model.Message, len(messages))
	for i, message := range messages {
		out[i] = model.Message{
			Role:        message.Role,
			Content:     strings.TrimSpace(message.Content),
			ToolCalls:   projectTranscriptToolCalls(message.ToolCalls, bounds),
			ToolResults: projectTranscriptToolResults(message.ToolResults, bounds),
		}
	}
	return out
}

func projectTranscriptToolCalls(calls []model.ToolCall, bounds projectionBounds) []model.ToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]model.ToolCall, len(calls))
	for i, call := range calls {
		out[i] = call
		out[i].Arguments = projectToolArguments(call.Arguments, bounds)
	}
	return out
}

func projectTranscriptToolResults(results []model.ToolResult, bounds projectionBounds) []model.ToolResult {
	if len(results) == 0 {
		return nil
	}

	out := make([]model.ToolResult, len(results))
	for i, result := range results {
		out[i] = result
		out[i].Message = sanitizeToolResultMessage(result.Message)
		out[i].Output = projectToolResultOutput(result.Output, bounds)
	}
	return out
}

func sanitizeToolResultMessage(message string) string {
	message = strings.TrimSpace(strings.ReplaceAll(message, "\r\n", "\n"))
	if message == "" {
		return ""
	}
	if index := strings.Index(message, "\n"); index >= 0 {
		message = strings.TrimSpace(message[:index])
	}
	if index := strings.Index(message, "{"); index >= 0 {
		message = strings.TrimSpace(message[:index])
	}
	if len([]rune(message)) <= 120 {
		return message
	}
	runes := []rune(message)
	return strings.TrimSpace(string(runes[:120]))
}
