package context

import (
	"encoding/json"
	"fmt"
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

type Renderer struct{}

// NewRenderer 创建 ContextProjection Renderer。
// Renderer 负责把结构化上下文变成 Provider 可以消费的模型请求。
// config 保留为构造兼容参数；投影选择和本地上限由 Engine 处理。
func NewRenderer(config RendererConfig) Renderer {
	return Renderer{}
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
	messages = append(messages, r.renderTranscript(projection.CurrentTurnTranscript)...)

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
			ToolResults: copyToolResults(message.ToolResults),
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
