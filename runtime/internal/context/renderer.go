package context

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gameagent/runtime/internal/definition"
	"gameagent/runtime/internal/model"
)

type RendererConfig struct {
	MemoryContextSizeLimit        int
	MaxToolResultOutputBytes      int
	MaxToolResultOutputDepth      int
	MaxToolResultOutputFields     int
	MaxToolResultOutputArrayItems int
}

type Renderer struct {
	config RendererConfig
}

// NewRenderer 创建 AgentContext Renderer。
// Renderer 负责把结构化上下文变成 Provider 可以消费的模型请求。
func NewRenderer(config RendererConfig) Renderer {
	return Renderer{config: config}
}

// Render 负责把 ContextProjection 转成 Provider Request。
// Renderer 在这里固定 Current Observation 优先于 Recent Memory 的上下文语义。
func (r Renderer) Render(projection ContextProjection) (model.Request, error) {
	messages := []model.Message{
		{
			Role:    model.RoleUser,
			Content: r.renderUserMessage(projection),
		},
	}
	messages = append(messages, r.renderTranscript(projection.Transcript)...)

	return model.Request{
		System:   projection.RuntimePolicy,
		Messages: messages,
		Tools:    append([]model.ToolDefinition(nil), projection.Tools...),
		Controls: []model.ControlDefinition{
			{
				Kind:        model.ControlSettle,
				Description: "Finish the current turn without an environment action.",
			},
		},
	}, nil
}

func (r Renderer) renderTranscript(transcript []model.Message) []model.Message {
	if len(transcript) == 0 {
		return nil
	}

	messages := make([]model.Message, 0, len(transcript))
	for _, message := range transcript {
		rendered := model.Message{
			Role:        message.Role,
			Content:     strings.TrimSpace(message.Content),
			ToolCalls:   copyToolCalls(message.ToolCalls),
			ToolResults: r.normalizeToolResults(message.ToolResults),
		}

		switch {
		case len(rendered.ToolCalls) > 0:
			rendered.Content = renderToolCalls(rendered.ToolCalls)
		case len(rendered.ToolResults) > 0:
			rendered.Content = renderToolResults(rendered.ToolResults)
		}
		messages = append(messages, rendered)
	}
	return messages
}

func renderToolCalls(calls []model.ToolCall) string {
	items := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		items = append(items, map[string]any{
			"tool_call_id": call.ID,
			"name":         call.Name,
			"arguments":    orderedMap(call.Arguments),
		})
	}
	return mustMarshalJSONString(items)
}

func renderToolResults(results []model.ToolResult) string {
	items := make([]map[string]any, 0, len(results))
	for _, result := range results {
		item := map[string]any{
			"tool_call_id": result.ToolCallID,
			"name":         result.Name,
			"status":       result.Status,
			"code":         result.Code,
		}
		if result.Message != "" {
			item["message"] = result.Message
		}
		if len(result.Output) > 0 {
			item["output"] = orderedMap(result.Output)
		}
		items = append(items, item)
	}
	return mustMarshalJSONString(items)
}

func (r Renderer) normalizeToolResults(results []model.ToolResult) []model.ToolResult {
	if len(results) == 0 {
		return nil
	}

	normalized := make([]model.ToolResult, len(results))
	for i, result := range results {
		normalized[i] = result
		normalized[i].Message = sanitizeToolResultMessage(result.Message)
		normalized[i].Output = r.projectToolResultOutput(result.Output)
	}
	return normalized
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

func (r Renderer) projectToolResultOutput(output map[string]any) map[string]any {
	if len(output) == 0 {
		return nil
	}

	bounds := r.toolResultOutputBounds()
	projected, ok := projectOutputValue(output, 1, bounds).(map[string]any)
	if !ok || len(projected) == 0 {
		return nil
	}

	data, err := json.Marshal(orderedMap(projected))
	if err == nil && bounds.maxBytes > 0 && len(data) > bounds.maxBytes {
		return map[string]any{
			"_truncated": "tool result output exceeded byte limit",
		}
	}
	return projected
}

type toolResultOutputBounds struct {
	maxBytes      int
	maxDepth      int
	maxFields     int
	maxArrayItems int
}

func (r Renderer) toolResultOutputBounds() toolResultOutputBounds {
	return toolResultOutputBounds{
		maxBytes:      positiveOrDefault(r.config.MaxToolResultOutputBytes, 8192),
		maxDepth:      positiveOrDefault(r.config.MaxToolResultOutputDepth, 4),
		maxFields:     positiveOrDefault(r.config.MaxToolResultOutputFields, 64),
		maxArrayItems: positiveOrDefault(r.config.MaxToolResultOutputArrayItems, 32),
	}
}

func positiveOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func projectOutputValue(value any, depth int, bounds toolResultOutputBounds) any {
	switch typed := value.(type) {
	case map[string]any:
		if bounds.maxDepth > 0 && depth > bounds.maxDepth {
			return "_truncated: max depth exceeded"
		}
		return projectOutputMap(typed, depth, bounds)
	case []any:
		if bounds.maxDepth > 0 && depth > bounds.maxDepth {
			return "_truncated: max depth exceeded"
		}
		return projectOutputArray(typed, depth, bounds)
	case string, bool, nil:
		return typed
	case int:
		return typed
	case int32:
		return typed
	case int64:
		return typed
	case float32:
		return typed
	case float64:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func projectOutputMap(values map[string]any, depth int, bounds toolResultOutputBounds) map[string]any {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	limit := len(keys)
	if bounds.maxFields > 0 && limit > bounds.maxFields {
		limit = bounds.maxFields
	}

	out := make(map[string]any, limit+1)
	for _, key := range keys[:limit] {
		out[key] = projectOutputValue(values[key], depth+1, bounds)
	}
	if limit < len(keys) {
		out["_truncated_fields"] = len(keys) - limit
	}
	return out
}

func projectOutputArray(values []any, depth int, bounds toolResultOutputBounds) []any {
	limit := len(values)
	if bounds.maxArrayItems > 0 && limit > bounds.maxArrayItems {
		limit = bounds.maxArrayItems
	}

	out := make([]any, 0, limit+1)
	for _, value := range values[:limit] {
		out = append(out, projectOutputValue(value, depth+1, bounds))
	}
	if limit < len(values) {
		out = append(out, fmt.Sprintf("_truncated_items:%d", len(values)-limit))
	}
	return out
}

func orderedMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out[key] = values[key]
	}
	return out
}

func mustMarshalJSONString(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
}

// renderUserMessage 渲染本轮模型输入的 user message。
// 它把 Recent Memory、Current Event 和 Current Observation 放进同一个可读上下文块。
func (r Renderer) renderUserMessage(projection ContextProjection) string {
	return fmt.Sprintf(`[Recent Memory]
%s

[Game Definition]
%s

[Agent Definition]
%s

[Agent Descriptor]
%s

[Current Event]
%s

[Current Event Context Facts]
%s

[Current Observation]
%s

[Instruction]
Current Observation is the current truth.
Recent Memory is historical context.
If Recent Memory conflicts with Current Observation, follow Current Observation.
If Recent Memory is from today and current game time has not clearly advanced much, treat it as nearby conversation context, not proof that the player left and returned.

Return tool calls only when an environment action is needed. If no action is needed, settle the current turn.
`,
		renderRecentMemoryProjection(projection.RecentMemory),
		renderGameDefinition(projection.GameDefinition),
		renderAgentDefinition(projection.AgentDefinition),
		renderAgentDescriptor(projection.AgentDescriptor),
		renderCurrentEvent(projection.CurrentEvent),
		renderCurrentEventContextFacts(projection.CurrentEventContextFacts),
		renderCurrentObservation(projection.CurrentObservation),
	)
}

func renderGameDefinition(game *definition.GameDefinition) string {
	if game == nil {
		return "(none)"
	}
	lines := make([]string, 0)
	appendLine(&lines, "title", game.Title)
	appendLine(&lines, "summary", game.Summary)
	appendList(&lines, "world_rules", game.WorldRules)
	appendList(&lines, "lore", game.Lore)
	appendList(&lines, "narrative_constraints", game.NarrativeConstraints)
	if len(lines) == 0 {
		return "(empty)"
	}
	return strings.Join(lines, "\n")
}

func renderAgentDefinition(agent *definition.AgentDefinition) string {
	if agent == nil {
		return "(none)"
	}
	lines := make([]string, 0)
	appendLine(&lines, "identity", agent.Identity)
	appendList(&lines, "personality", agent.Personality)
	appendList(&lines, "speech_style", agent.SpeechStyle)
	appendList(&lines, "preferences", agent.Preferences)
	appendList(&lines, "behavior_guidelines", agent.BehaviorGuidelines)
	if len(lines) == 0 {
		return "(empty)"
	}
	return strings.Join(lines, "\n")
}

func appendLine(lines *[]string, label string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*lines = append(*lines, fmt.Sprintf("%s: %s", label, value))
}

func appendList(lines *[]string, label string, values []string) {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	if len(items) == 0 {
		return
	}
	*lines = append(*lines, label+":")
	for _, item := range items {
		*lines = append(*lines, "- "+item)
	}
}

func renderAgentDescriptor(descriptor definition.AgentInstanceDescriptor) string {
	definitionID := strings.TrimSpace(descriptor.DefinitionID)
	if definitionID == "" {
		definitionID = "(unspecified)"
	}
	return fmt.Sprintf(
		"game_id: %s\nworld_id: %s\nentity_id: %s\nentity_type: %s\ndisplay_name: %s\ndefinition_id: %s",
		descriptor.SessionKey.GameID,
		descriptor.SessionKey.WorldID,
		descriptor.SessionKey.EntityID,
		emptyLabel(descriptor.EntityType),
		emptyLabel(descriptor.DisplayName),
		definitionID,
	)
}

func emptyLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(unspecified)"
	}
	return value
}

// renderCurrentEvent 渲染 Engine 给出的当前事件投影。
func renderCurrentEvent(event EventProjection) string {
	fields := make(map[string]any)
	appendStringField(fields, "event_id", event.EventID)
	appendStringField(fields, "event_type", event.EventType)
	appendStringField(fields, "world_id", event.WorldID)
	appendStringField(fields, "target_entity_id", event.TargetEntityID)
	if event.Sequence != 0 {
		fields["sequence"] = event.Sequence
	}
	if event.GameTime != nil {
		fields["game_time"] = gameTimeJSON(event.GameTime)
	}
	if event.CanonicalTarget != nil {
		fields["canonical_target"] = entityRefJSON(event.CanonicalTarget)
	}
	if len(event.Payload) > 0 {
		fields["payload"] = event.Payload
	}
	return renderJSON(fields)
}

func renderCurrentEventContextFacts(facts []ContextFactProjection) string {
	if len(facts) == 0 {
		return "(none)"
	}
	return renderJSON(facts)
}

func renderCurrentObservation(observation ObservationProjection) string {
	fields := make(map[string]any)
	appendStringField(fields, "world_id", observation.WorldID)
	appendStringField(fields, "entity_id", observation.EntityID)
	if observation.GameTime != nil {
		fields["game_time"] = gameTimeJSON(observation.GameTime)
	}
	if len(observation.State) > 0 {
		fields["state"] = observation.State
	}
	return renderJSON(fields)
}

func appendStringField(fields map[string]any, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		fields[key] = value
	}
}

func gameTimeJSON(gameTime interface {
	GetYear() int32
	GetSeason() int32
	GetDay() int32
	GetHour() int32
	GetMinute() int32
	GetTick() int64
}) map[string]any {
	return map[string]any{
		"day":    gameTime.GetDay(),
		"hour":   gameTime.GetHour(),
		"minute": gameTime.GetMinute(),
		"season": gameTime.GetSeason(),
		"tick":   gameTime.GetTick(),
		"year":   gameTime.GetYear(),
	}
}

func entityRefJSON(ref interface {
	GetEntityId() string
	GetEntityType() string
	GetDisplayName() string
	GetDefinitionId() string
}) map[string]any {
	fields := make(map[string]any)
	appendStringField(fields, "entity_id", ref.GetEntityId())
	appendStringField(fields, "entity_type", ref.GetEntityType())
	appendStringField(fields, "display_name", ref.GetDisplayName())
	appendStringField(fields, "definition_id", ref.GetDefinitionId())
	return fields
}

func renderJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	if string(data) == "null" {
		return "{}"
	}
	return string(data)
}
