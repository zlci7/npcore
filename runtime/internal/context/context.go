package context

import (
	"errors"
	"fmt"
	"strings"

	protocolv1alpha2 "gameagent/protocol/gen/go/gameagent/protocol/v1alpha2"
	"gameagent/runtime/internal/definition"
	"gameagent/runtime/internal/memory"
	"gameagent/runtime/internal/model"
	"gameagent/runtime/internal/session"
	"gameagent/runtime/internal/tool"
)

var ErrInvalidInput = errors.New("invalid agent context input")

type AgentContext = ContextProjection

type EngineConfig struct {
	MemoryContextSizeLimit        int
	MaxToolResultOutputBytes      int
	MaxToolResultOutputDepth      int
	MaxToolResultOutputFields     int
	MaxToolResultOutputArrayItems int
}

type Engine struct {
	config EngineConfig
}

type ContextProjection struct {
	SessionKey session.AgentSessionKey

	CanonicalTarget *protocolv1alpha2.EntityRef

	AgentDescriptor definition.AgentInstanceDescriptor
	GameDefinition  *definition.GameDefinition
	AgentDefinition *definition.AgentDefinition

	RuntimePolicy string

	RecentMemories []memory.Record

	Event       *protocolv1alpha2.GameEvent
	Observation *protocolv1alpha2.Observation

	CurrentEvent             EventProjection
	CurrentEventContextFacts []ContextFactProjection

	RecentMemory []MemoryProjection

	Tools []model.ToolDefinition

	Transcript []model.Message
}

type EventProjection struct {
	EventID         string                      `json:"event_id,omitempty"`
	EventType       string                      `json:"event_type,omitempty"`
	WorldID         string                      `json:"world_id,omitempty"`
	TargetEntityID  string                      `json:"target_entity_id,omitempty"`
	Sequence        uint64                      `json:"sequence,omitempty"`
	GameTime        *protocolv1alpha2.GameTime  `json:"game_time,omitempty"`
	CanonicalTarget *protocolv1alpha2.EntityRef `json:"canonical_target,omitempty"`
	Payload         map[string]any              `json:"payload,omitempty"`
}

type ContextFactProjection struct {
	Kind           string         `json:"kind,omitempty"`
	ActorEntityID  string         `json:"actor_entity_id,omitempty"`
	TargetEntityID string         `json:"target_entity_id,omitempty"`
	ScopeID        string         `json:"scope_id,omitempty"`
	Text           string         `json:"text,omitempty"`
	Label          string         `json:"label,omitempty"`
	Attributes     map[string]any `json:"attributes,omitempty"`
}

type MemoryProjection struct {
	MemoryID     string   `json:"memory_id,omitempty"`
	TimeRelation string   `json:"time_relation,omitempty"`
	Summaries    []string `json:"summaries,omitempty"`
}

type BuildInput struct {
	SessionKey      session.AgentSessionKey
	CanonicalTarget *protocolv1alpha2.EntityRef
	AgentDescriptor definition.AgentInstanceDescriptor
	GameDefinition  *definition.GameDefinition
	AgentDefinition *definition.AgentDefinition

	RuntimePolicy string

	RecentMemories []memory.Record

	Event       *protocolv1alpha2.GameEvent
	Observation *protocolv1alpha2.Observation

	TurnToolView tool.TurnToolView
	Tools        []model.ToolDefinition

	Transcript []model.Message
}

func NewEngine(config EngineConfig) Engine {
	return Engine{config: config}
}

func (e Engine) Build(input BuildInput) (ContextProjection, error) {
	if err := validateEngineInput(input); err != nil {
		return ContextProjection{}, err
	}

	return ContextProjection{
		SessionKey:               input.SessionKey,
		CanonicalTarget:          input.CanonicalTarget,
		AgentDescriptor:          input.AgentDescriptor,
		GameDefinition:           copyGameDefinition(input.GameDefinition),
		AgentDefinition:          copyAgentDefinition(input.AgentDefinition),
		RuntimePolicy:            input.RuntimePolicy,
		RecentMemories:           append([]memory.Record(nil), input.RecentMemories...),
		Event:                    input.Event,
		Observation:              input.Observation,
		CurrentEvent:             projectCurrentEvent(input.Event, input.CanonicalTarget),
		CurrentEventContextFacts: projectCurrentEventContextFacts(input.Event.GetContextFacts()),
		RecentMemory:             projectRecentMemories(input.RecentMemories, e.config.MemoryContextSizeLimit, currentGameTimeFromEventObservation(input.Event, input.Observation)),
		Tools:                    input.TurnToolView.Available(),
		Transcript:               copyMessages(input.Transcript),
	}, nil
}

type Builder struct{}

// NewBuilder 创建 AgentContext Builder。
// Builder 本身无状态，便于在 Loop 中长期复用。
func NewBuilder() Builder {
	return Builder{}
}

func validateEngineInput(input BuildInput) error {
	if input.Event == nil {
		return fmt.Errorf("%w: event is required", ErrInvalidInput)
	}
	if input.Observation == nil {
		return fmt.Errorf("%w: observation is required", ErrInvalidInput)
	}
	if input.CanonicalTarget == nil {
		return fmt.Errorf("%w: canonical target is required", ErrInvalidInput)
	}
	if input.RuntimePolicy == "" {
		return fmt.Errorf("%w: runtime policy is required", ErrInvalidInput)
	}
	if input.SessionKey.GameID == "" {
		return fmt.Errorf("%w: session key game_id is required", ErrInvalidInput)
	}
	if input.SessionKey.WorldID == "" {
		return fmt.Errorf("%w: session key world_id is required", ErrInvalidInput)
	}
	if input.SessionKey.EntityID == "" {
		return fmt.Errorf("%w: session key entity_id is required", ErrInvalidInput)
	}
	if input.CanonicalTarget.GetEntityId() != input.SessionKey.EntityID {
		return fmt.Errorf("%w: canonical target entity_id does not match session key", ErrInvalidInput)
	}
	if input.AgentDescriptor.SessionKey != input.SessionKey {
		return fmt.Errorf("%w: agent descriptor session key does not match session key", ErrInvalidInput)
	}
	if input.AgentDescriptor.DefinitionID != input.CanonicalTarget.GetDefinitionId() {
		return fmt.Errorf("%w: agent descriptor definition_id does not match canonical target", ErrInvalidInput)
	}
	if input.Event.GetWorldId() != input.SessionKey.WorldID {
		return fmt.Errorf("%w: event world_id does not match session key", ErrInvalidInput)
	}
	if input.Event.GetTargetEntityId() != input.SessionKey.EntityID {
		return fmt.Errorf("%w: event target_entity_id does not match session key", ErrInvalidInput)
	}
	if input.Observation.GetWorldId() != input.SessionKey.WorldID {
		return fmt.Errorf("%w: observation world_id does not match session key", ErrInvalidInput)
	}
	if input.Observation.GetEntityId() != input.SessionKey.EntityID {
		return fmt.Errorf("%w: observation entity_id does not match session key", ErrInvalidInput)
	}
	if input.GameDefinition != nil && input.GameDefinition.GameID != input.SessionKey.GameID {
		return fmt.Errorf("%w: game definition game_id does not match session key", ErrInvalidInput)
	}
	if input.CanonicalTarget.GetDefinitionId() == "" && input.AgentDefinition != nil {
		return fmt.Errorf("%w: agent definition must be nil when canonical target definition_id is empty", ErrInvalidInput)
	}
	if input.AgentDefinition != nil {
		if input.AgentDefinition.GameID != input.SessionKey.GameID {
			return fmt.Errorf("%w: agent definition game_id does not match session key", ErrInvalidInput)
		}
		if input.AgentDefinition.DefinitionID != input.AgentDescriptor.DefinitionID {
			return fmt.Errorf("%w: agent definition definition_id does not match agent descriptor", ErrInvalidInput)
		}
		if input.AgentDefinition.DefinitionID != input.CanonicalTarget.GetDefinitionId() {
			return fmt.Errorf("%w: agent definition definition_id does not match canonical target", ErrInvalidInput)
		}
	}
	return nil
}

func projectCurrentEvent(event *protocolv1alpha2.GameEvent, canonicalTarget *protocolv1alpha2.EntityRef) EventProjection {
	payload := map[string]any(nil)
	if event.GetPayload() != nil {
		payload = copyMap(event.GetPayload().AsMap())
	}

	return EventProjection{
		EventID:         event.GetEventId(),
		EventType:       event.GetEventType(),
		WorldID:         event.GetWorldId(),
		TargetEntityID:  event.GetTargetEntityId(),
		Sequence:        event.GetSequence(),
		GameTime:        event.GetGameTime(),
		CanonicalTarget: canonicalTarget,
		Payload:         payload,
	}
}

func projectCurrentEventContextFacts(facts []*protocolv1alpha2.ContextFact) []ContextFactProjection {
	if len(facts) == 0 {
		return nil
	}

	out := make([]ContextFactProjection, 0, len(facts))
	for _, fact := range facts {
		if fact == nil {
			continue
		}
		attributes := map[string]any(nil)
		if fact.GetAttributes() != nil {
			attributes = copyMap(fact.GetAttributes().AsMap())
		}
		out = append(out, ContextFactProjection{
			Kind:           strings.TrimSpace(fact.GetKind()),
			ActorEntityID:  strings.TrimSpace(fact.GetActorEntityId()),
			TargetEntityID: strings.TrimSpace(fact.GetTargetEntityId()),
			ScopeID:        strings.TrimSpace(fact.GetScopeId()),
			Text:           strings.TrimSpace(fact.GetText()),
			Label:          strings.TrimSpace(fact.GetLabel()),
			Attributes:     attributes,
		})
	}
	return out
}

// Build 负责建立 AgentContext 边界。
// 它只做结构化组装与必要校验，不负责 prompt 文本渲染。
func (Builder) Build(input BuildInput) (AgentContext, error) {
	if input.Event == nil {
		return AgentContext{}, fmt.Errorf("%w: event is required", ErrInvalidInput)
	}
	if input.Observation == nil {
		return AgentContext{}, fmt.Errorf("%w: observation is required", ErrInvalidInput)
	}
	if input.SessionKey.GameID == "" || input.SessionKey.WorldID == "" || input.SessionKey.EntityID == "" {
		return AgentContext{}, fmt.Errorf("%w: session key is required", ErrInvalidInput)
	}

	descriptor := input.AgentDescriptor
	descriptor.SessionKey = input.SessionKey

	return AgentContext{
		SessionKey:      input.SessionKey,
		AgentDescriptor: descriptor,
		GameDefinition:  copyGameDefinition(input.GameDefinition),
		AgentDefinition: copyAgentDefinition(input.AgentDefinition),
		RuntimePolicy:   input.RuntimePolicy,
		RecentMemories:  append([]memory.Record(nil), input.RecentMemories...),
		Event:           input.Event,
		Observation:     input.Observation,
		Tools:           append([]model.ToolDefinition(nil), input.Tools...),
		Transcript:      copyMessages(input.Transcript),
	}, nil
}

func copyGameDefinition(game *definition.GameDefinition) *definition.GameDefinition {
	if game == nil {
		return nil
	}
	out := *game
	out.WorldRules = append([]string(nil), game.WorldRules...)
	out.Lore = append([]string(nil), game.Lore...)
	out.NarrativeConstraints = append([]string(nil), game.NarrativeConstraints...)
	return &out
}

func copyAgentDefinition(agent *definition.AgentDefinition) *definition.AgentDefinition {
	if agent == nil {
		return nil
	}
	out := *agent
	out.Personality = append([]string(nil), agent.Personality...)
	out.SpeechStyle = append([]string(nil), agent.SpeechStyle...)
	out.Preferences = append([]string(nil), agent.Preferences...)
	out.BehaviorGuidelines = append([]string(nil), agent.BehaviorGuidelines...)
	return &out
}

func copyMessages(messages []model.Message) []model.Message {
	if len(messages) == 0 {
		return nil
	}

	out := make([]model.Message, len(messages))
	for i, message := range messages {
		out[i] = message
		out[i].ToolCalls = copyToolCalls(message.ToolCalls)
		out[i].ToolResults = copyToolResults(message.ToolResults)
	}
	return out
}

func copyToolCalls(calls []model.ToolCall) []model.ToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]model.ToolCall, len(calls))
	for i, call := range calls {
		out[i] = call
		out[i].Arguments = copyMap(call.Arguments)
	}
	return out
}

func copyToolResults(results []model.ToolResult) []model.ToolResult {
	if len(results) == 0 {
		return nil
	}

	out := make([]model.ToolResult, len(results))
	for i, result := range results {
		out[i] = result
		out[i].Output = copyMap(result.Output)
	}
	return out
}

func copyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}

	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
