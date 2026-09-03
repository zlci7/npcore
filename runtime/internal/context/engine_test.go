package context_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	protocolv1alpha2 "gameagent/protocol/gen/go/gameagent/protocol/v1alpha2"
	agentcontext "gameagent/runtime/internal/context"
	"gameagent/runtime/internal/definition"
	"gameagent/runtime/internal/memory"
	"gameagent/runtime/internal/model"
	"gameagent/runtime/internal/session"
	"gameagent/runtime/internal/tool"
)

func TestEngineBuildCreatesContextProjectionFromValidatedInput(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
	input := validEngineInput(t)
	input.Observation.State = mustStruct(t, map[string]any{"weather": "rain"})

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if projection.SessionKey != input.SessionKey {
		t.Fatalf("SessionKey = %+v, want %+v", projection.SessionKey, input.SessionKey)
	}
	if projection.CanonicalTarget.GetEntityId() != "npc:Abigail" {
		t.Fatalf("CanonicalTarget entity_id = %q, want npc:Abigail", projection.CanonicalTarget.GetEntityId())
	}
	if projection.AgentDescriptor.DefinitionID != "npc/abigail" {
		t.Fatalf("AgentDescriptor.DefinitionID = %q, want npc/abigail", projection.AgentDescriptor.DefinitionID)
	}
	if projection.GameDefinition == nil || projection.GameDefinition.GameID != "stardew-valley" {
		t.Fatalf("GameDefinition = %+v, want stardew-valley definition", projection.GameDefinition)
	}
	if projection.AgentDefinition == nil || projection.AgentDefinition.DefinitionID != "npc/abigail" {
		t.Fatalf("AgentDefinition = %+v, want npc/abigail definition", projection.AgentDefinition)
	}
	if projection.RuntimePolicy != "runtime policy" {
		t.Fatalf("RuntimePolicy = %q, want runtime policy", projection.RuntimePolicy)
	}
	if projection.CurrentEvent.EventID != "event-1" {
		t.Fatalf("CurrentEvent.EventID = %q, want event-1", projection.CurrentEvent.EventID)
	}
	if projection.CurrentObservation.EntityID != "npc:Abigail" || projection.CurrentObservation.WorldID != "world-a" {
		t.Fatalf("CurrentObservation scope = %+v, want world-a/npc:Abigail", projection.CurrentObservation)
	}
	if got := projection.CurrentObservation.State["weather"]; got != "rain" {
		t.Fatalf("CurrentObservation.State[weather] = %#v, want rain", got)
	}
	if got := len(projection.Tools); got != 1 {
		t.Fatalf("Tools length = %d, want 1", got)
	}
	if projection.Tools[0].Name != "speak" {
		t.Fatalf("Tools[0].Name = %q, want speak", projection.Tools[0].Name)
	}
}

func TestEngineBuildAllowsDefinitionFallback(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
	input := validEngineInput(t)
	input.GameDefinition = nil
	input.AgentDefinition = nil

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if projection.GameDefinition != nil {
		t.Fatalf("GameDefinition = %+v, want nil fallback", projection.GameDefinition)
	}
	if projection.AgentDefinition != nil {
		t.Fatalf("AgentDefinition = %+v, want nil fallback", projection.AgentDefinition)
	}
	if projection.AgentDescriptor.DefinitionID != "npc/abigail" {
		t.Fatalf("AgentDescriptor.DefinitionID = %q, want original definition id", projection.AgentDescriptor.DefinitionID)
	}
}

func TestEngineBuildRejectsMissingRequiredInputs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*agentcontext.BuildInput)
		wantErr string
	}{
		{
			name: "missing event",
			mutate: func(input *agentcontext.BuildInput) {
				input.Event = nil
			},
			wantErr: "event is required",
		},
		{
			name: "missing observation",
			mutate: func(input *agentcontext.BuildInput) {
				input.Observation = nil
			},
			wantErr: "observation is required",
		},
		{
			name: "missing canonical target",
			mutate: func(input *agentcontext.BuildInput) {
				input.CanonicalTarget = nil
			},
			wantErr: "canonical target is required",
		},
		{
			name: "missing runtime policy",
			mutate: func(input *agentcontext.BuildInput) {
				input.RuntimePolicy = ""
			},
			wantErr: "runtime policy is required",
		},
		{
			name: "missing session game",
			mutate: func(input *agentcontext.BuildInput) {
				input.SessionKey.GameID = ""
			},
			wantErr: "session key game_id is required",
		},
		{
			name: "missing session world",
			mutate: func(input *agentcontext.BuildInput) {
				input.SessionKey.WorldID = ""
			},
			wantErr: "session key world_id is required",
		},
		{
			name: "missing session entity",
			mutate: func(input *agentcontext.BuildInput) {
				input.SessionKey.EntityID = ""
			},
			wantErr: "session key entity_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
			input := validEngineInput(t)
			tt.mutate(&input)

			_, err := engine.Build(input)
			assertInvalidInputError(t, err, tt.wantErr)
		})
	}
}

func TestEngineBuildRejectsScopeMismatches(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*agentcontext.BuildInput)
		wantErr string
	}{
		{
			name: "canonical target entity mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.CanonicalTarget.EntityId = "npc:Haley"
			},
			wantErr: "canonical target entity_id does not match session key",
		},
		{
			name: "descriptor session mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.AgentDescriptor.SessionKey.EntityID = "npc:Haley"
			},
			wantErr: "agent descriptor session key does not match session key",
		},
		{
			name: "descriptor definition mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.AgentDescriptor.DefinitionID = "npc/haley"
			},
			wantErr: "agent descriptor definition_id does not match canonical target",
		},
		{
			name: "event world mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.Event.WorldId = "world-b"
			},
			wantErr: "event world_id does not match session key",
		},
		{
			name: "event target mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.Event.TargetEntityId = "npc:Haley"
			},
			wantErr: "event target_entity_id does not match session key",
		},
		{
			name: "observation world mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.Observation.WorldId = "world-b"
			},
			wantErr: "observation world_id does not match session key",
		},
		{
			name: "observation entity mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.Observation.EntityId = "npc:Haley"
			},
			wantErr: "observation entity_id does not match session key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
			input := validEngineInput(t)
			tt.mutate(&input)

			_, err := engine.Build(input)
			assertInvalidInputError(t, err, tt.wantErr)
		})
	}
}

func TestEngineBuildRejectsDefinitionBindingMismatches(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*agentcontext.BuildInput)
		wantErr string
	}{
		{
			name: "game definition game mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.GameDefinition.GameID = "other-game"
			},
			wantErr: "game definition game_id does not match session key",
		},
		{
			name: "agent definition game mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.AgentDefinition.GameID = "other-game"
			},
			wantErr: "agent definition game_id does not match session key",
		},
		{
			name: "agent definition descriptor mismatch",
			mutate: func(input *agentcontext.BuildInput) {
				input.AgentDefinition.DefinitionID = "npc/haley"
			},
			wantErr: "agent definition definition_id does not match agent descriptor",
		},
		{
			name: "empty target definition cannot have agent definition",
			mutate: func(input *agentcontext.BuildInput) {
				input.CanonicalTarget.DefinitionId = ""
				input.AgentDescriptor.DefinitionID = ""
			},
			wantErr: "agent definition must be nil when canonical target definition_id is empty",
		},
		{
			name: "empty target definition cannot have descriptor definition",
			mutate: func(input *agentcontext.BuildInput) {
				input.CanonicalTarget.DefinitionId = ""
				input.AgentDefinition = nil
			},
			wantErr: "agent descriptor definition_id does not match canonical target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
			input := validEngineInput(t)
			tt.mutate(&input)

			_, err := engine.Build(input)
			assertInvalidInputError(t, err, tt.wantErr)
		})
	}
}

func TestEngineBuildProjectsCurrentEventAndContextFactsSeparately(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
	input := validEngineInput(t)
	input.Event.Sequence = 42
	input.Event.GameTime = &protocolv1alpha2.GameTime{
		Year:   ptrInt32(1),
		Season: ptrInt32(2),
		Day:    ptrInt32(3),
		Hour:   ptrInt32(6),
		Minute: ptrInt32(20),
		Tick:   ptrInt64(9001),
	}
	input.Event.Payload = mustStruct(t, map[string]any{
		"dialogue_id": "intro",
		"nested": map[string]any{
			"mood": "curious",
		},
	})
	input.Event.ContextFacts = []*protocolv1alpha2.ContextFact{
		{
			Kind:           "utterance",
			ActorEntityId:  "player:local",
			TargetEntityId: "npc:Abigail",
			ScopeId:        "conversation:1",
			Text:           "  Want to explore the mines?  ",
			Label:          "  player invite  ",
			Attributes: mustStruct(t, map[string]any{
				"tone": "friendly",
			}),
		},
		{
			Kind:  "scene",
			Label: "rainy day",
		},
	}

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	event := projection.CurrentEvent
	if event.EventID != "event-1" || event.EventType != "player_interacted_with_npc" {
		t.Fatalf("CurrentEvent identity = %+v, want event-1/player_interacted_with_npc", event)
	}
	if event.WorldID != "world-a" || event.TargetEntityID != "npc:Abigail" || event.Sequence != 42 {
		t.Fatalf("CurrentEvent scope = %+v, want world-a/npc:Abigail/42", event)
	}
	if event.GameTime == nil || event.GameTime.GetTick() != 9001 {
		t.Fatalf("CurrentEvent.GameTime = %+v, want copied game time", event.GameTime)
	}
	if event.CanonicalTarget.GetDefinitionId() != "npc/abigail" {
		t.Fatalf("CurrentEvent.CanonicalTarget = %+v, want canonical target", event.CanonicalTarget)
	}
	if got := event.Payload["dialogue_id"]; got != "intro" {
		t.Fatalf("CurrentEvent.Payload[dialogue_id] = %#v, want intro", got)
	}
	eventJSON := mustMarshalJSONBytes(t, event)
	if strings.Contains(string(eventJSON), "context_facts") || strings.Contains(string(eventJSON), "ContextFacts") {
		t.Fatalf("CurrentEvent unexpectedly contains context facts: %s", eventJSON)
	}

	facts := projection.CurrentEventContextFacts
	if got, want := len(facts), 2; got != want {
		t.Fatalf("CurrentEventContextFacts length = %d, want %d", got, want)
	}
	if facts[0].Kind != "utterance" ||
		facts[0].ActorEntityID != "player:local" ||
		facts[0].TargetEntityID != "npc:Abigail" ||
		facts[0].ScopeID != "conversation:1" ||
		facts[0].Text != "Want to explore the mines?" ||
		facts[0].Label != "player invite" {
		t.Fatalf("first fact projection = %+v, want trimmed utterance fact", facts[0])
	}
	if got, want := facts[0].Attributes, map[string]any{"tone": "friendly"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first fact attributes = %#v, want %#v", got, want)
	}
	if facts[1].Kind != "scene" || facts[1].Label != "rainy day" {
		t.Fatalf("second fact projection = %+v, want scene/rainy day", facts[1])
	}
}

func TestEngineBuildDoesNotInferContextFactsFromPayloadOrObservationState(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
	input := validEngineInput(t)
	input.Event.Payload = mustStruct(t, map[string]any{
		"context_facts": []any{
			map[string]any{"kind": "utterance", "text": "payload should not become a fact"},
		},
	})
	input.Observation.State = mustStruct(t, map[string]any{
		"context_facts": []any{
			map[string]any{"kind": "scene", "label": "state should not become a fact"},
		},
	})

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if got := len(projection.CurrentEventContextFacts); got != 0 {
		t.Fatalf("CurrentEventContextFacts length = %d, want 0", got)
	}
	if got := projection.CurrentEvent.Payload["context_facts"]; got == nil {
		t.Fatal("CurrentEvent payload lost context_facts key; want payload preserved as generic JSON")
	}
}

func TestEngineBuildProjectsRecentMemorySelection(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{MemoryContextSizeLimit: 4096})
	input := validEngineInput(t)
	input.Event.GameTime = &protocolv1alpha2.GameTime{
		Year:   ptrInt32(1),
		Season: ptrInt32(1),
		Day:    ptrInt32(2),
		Hour:   ptrInt32(6),
		Minute: ptrInt32(20),
	}
	gameTime := &memory.GameTimeSnapshot{Year: 1, Season: 1, Day: 2, Hour: 6, Minute: 20}
	input.RecentMemories = []memory.Record{
		{
			MemoryID:            "memory-seq-2",
			GameTime:            gameTime,
			SourceEventSequence: 2,
			Outcomes: []memory.TurnOutcome{{
				ToolName: "present_dialogue",
				ToolArguments: map[string]any{
					"text":            "I brought snacks.",
					"allow_free_text": true,
				},
				ActionStatus: "ACTION_STATUS_SUCCEEDED",
			}},
		},
		{
			MemoryID:            "memory-seq-1",
			GameTime:            gameTime,
			SourceEventSequence: 1,
			SourceContextFacts: []memory.SourceContextFact{{
				Kind:          "utterance",
				ActorEntityID: "player:local",
				Text:          "Meet me at the mine.",
			}},
		},
		{
			MemoryID: "memory-future",
			GameTime: &memory.GameTimeSnapshot{Year: 1, Season: 1, Day: 2, Hour: 8, Minute: 0},
			SourceContextFacts: []memory.SourceContextFact{{
				Kind:          "utterance",
				ActorEntityID: "player:local",
				Text:          "This future fact must be filtered.",
			}},
		},
	}

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if got, want := len(projection.RecentMemory), 2; got != want {
		t.Fatalf("RecentMemory length = %d, want %d", got, want)
	}
	if projection.RecentMemory[0].MemoryID != "memory-seq-1" || projection.RecentMemory[1].MemoryID != "memory-seq-2" {
		t.Fatalf("RecentMemory order = %+v, want sequence order memory-seq-1 then memory-seq-2", projection.RecentMemory)
	}
	if got, want := projection.RecentMemory[0].TimeRelation, "today 06:20"; got != want {
		t.Fatalf("first TimeRelation = %q, want %q", got, want)
	}
	if got, want := projection.RecentMemory[0].Summaries, []string{`player:local said "Meet me at the mine."`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first summaries = %#v, want %#v", got, want)
	}
	wantOutcome := `tool "present_dialogue" status "ACTION_STATUS_SUCCEEDED" arguments {"allow_free_text":true,"text":"I brought snacks."}`
	if got, want := projection.RecentMemory[1].Summaries, []string{wantOutcome}; !reflect.DeepEqual(got, want) {
		t.Fatalf("second summaries = %#v, want %#v", got, want)
	}
}

func TestEngineBuildAppliesRecentMemorySoftLimit(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{MemoryContextSizeLimit: 95})
	input := validEngineInput(t)
	input.RecentMemories = []memory.Record{
		{
			MemoryID: "old",
			SourceContextFacts: []memory.SourceContextFact{{
				Kind:          "utterance",
				ActorEntityID: "player:local",
				Text:          "older memory should be trimmed by the soft limit",
			}},
		},
		{
			MemoryID: "new",
			SourceContextFacts: []memory.SourceContextFact{{
				Kind:          "utterance",
				ActorEntityID: "player:local",
				Text:          "newest memory is retained",
			}},
		},
	}

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if got, want := len(projection.RecentMemory), 1; got != want {
		t.Fatalf("RecentMemory length = %d, want %d", got, want)
	}
	if projection.RecentMemory[0].MemoryID != "new" {
		t.Fatalf("retained memory = %q, want newest memory", projection.RecentMemory[0].MemoryID)
	}
}

func TestEngineBuildProjectsBoundedRecentMemoryToolArguments(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{
		MemoryContextSizeLimit:        4096,
		MaxToolResultOutputBytes:      300,
		MaxToolResultOutputDepth:      2,
		MaxToolResultOutputFields:     2,
		MaxToolResultOutputArrayItems: 2,
	})
	input := validEngineInput(t)
	input.RecentMemories = []memory.Record{{
		Outcomes: []memory.TurnOutcome{{
			ToolName: "inspect",
			ToolArguments: map[string]any{
				"a": []any{"one", "two", "three"},
				"b": map[string]any{"nested": map[string]any{"leaf": "too deep"}},
				"c": "extra field",
			},
			ActionStatus: "ACTION_STATUS_SUCCEEDED",
		}},
	}}

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	summary := projection.RecentMemory[0].Summaries[0]
	assertStringContainsAll(t, summary, `tool "inspect"`, `status "ACTION_STATUS_SUCCEEDED"`, `"a":["one","two","_truncated_items:1"]`, "_truncated")
	for _, unwanted := range []string{"three", "extra field", "too deep"} {
		if strings.Contains(summary, unwanted) {
			t.Fatalf("bounded memory arguments leaked %q:\n%s", unwanted, summary)
		}
	}
}

func TestEngineBuildProjectsCurrentTurnTranscriptWithBounds(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{
		MaxToolResultOutputBytes:      300,
		MaxToolResultOutputDepth:      2,
		MaxToolResultOutputFields:     2,
		MaxToolResultOutputArrayItems: 2,
	})
	input := validEngineInput(t)
	input.Transcript = []model.Message{
		{
			Role: model.RoleAssistant,
			ToolCalls: []model.ToolCall{{
				ID:   "call_1",
				Name: "inspect",
				Arguments: map[string]any{
					"a": []any{"one", "two", "three"},
					"b": map[string]any{"nested": map[string]any{"leaf": "too deep"}},
					"c": "extra field",
				},
			}},
		},
		{
			Role: model.RoleTool,
			ToolResults: []model.ToolResult{{
				ToolCallID: "call_1",
				Name:       "inspect",
				Status:     "rejected",
				Code:       "adapter_rejected",
				Message:    "adapter rejected request\nstack trace line\n{\"raw\":\"json\",\"action_id\":\"runtime-action-123\"}" + strings.Repeat("x", 180),
				Output: map[string]any{
					"a": []any{"one", "two", "three"},
					"b": map[string]any{"nested": map[string]any{"leaf": "too deep"}},
					"c": "extra field",
				},
			}},
		},
	}

	projection, err := engine.Build(input)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if got, want := len(projection.CurrentTurnTranscript), 2; got != want {
		t.Fatalf("CurrentTurnTranscript length = %d, want %d", got, want)
	}
	callArgs := projection.CurrentTurnTranscript[0].ToolCalls[0].Arguments
	if _, ok := callArgs["c"]; ok {
		t.Fatalf("projected tool call arguments leaked extra field: %#v", callArgs)
	}
	if got := callArgs["a"]; !reflect.DeepEqual(got, []any{"one", "two", "_truncated_items:1"}) {
		t.Fatalf("projected tool call arguments[a] = %#v", got)
	}
	result := projection.CurrentTurnTranscript[1].ToolResults[0]
	if strings.Contains(result.Message, "stack trace") || strings.Contains(result.Message, "runtime-action-123") || strings.Contains(result.Message, `{"raw"`) {
		t.Fatalf("projected tool result message leaked raw diagnostic: %q", result.Message)
	}
	if len(result.Message) > 120 {
		t.Fatalf("projected tool result message length = %d, want <= 120", len(result.Message))
	}
	if _, ok := result.Output["c"]; ok {
		t.Fatalf("projected tool result output leaked extra field: %#v", result.Output)
	}
	if got := result.Output["a"]; !reflect.DeepEqual(got, []any{"one", "two", "_truncated_items:1"}) {
		t.Fatalf("projected tool result output[a] = %#v", got)
	}
}

func validEngineInput(t *testing.T) agentcontext.BuildInput {
	t.Helper()

	key := session.AgentSessionKey{GameID: "stardew-valley", WorldID: "world-a", EntityID: "npc:Abigail"}
	target := &protocolv1alpha2.EntityRef{
		EntityId:     key.EntityID,
		EntityType:   "npc",
		DisplayName:  "Abigail",
		DefinitionId: "npc/abigail",
	}

	return agentcontext.BuildInput{
		SessionKey:      key,
		CanonicalTarget: target,
		AgentDescriptor: definition.NewAgentInstanceDescriptor(key, target),
		GameDefinition: &definition.GameDefinition{
			SchemaVersion: definition.SchemaVersionV1Alpha1,
			GameID:        key.GameID,
			Title:         "Stardew Valley",
		},
		AgentDefinition: &definition.AgentDefinition{
			SchemaVersion: definition.SchemaVersionV1Alpha1,
			GameID:        key.GameID,
			DefinitionID:  "npc/abigail",
			Identity:      "A purple-haired villager.",
		},
		RuntimePolicy: "runtime policy",
		Event: &protocolv1alpha2.GameEvent{
			EventId:        "event-1",
			EventType:      "player_interacted_with_npc",
			WorldId:        key.WorldID,
			TargetEntityId: key.EntityID,
		},
		Observation: &protocolv1alpha2.Observation{
			WorldId:  key.WorldID,
			EntityId: key.EntityID,
		},
		TurnToolView: singleToolView(t, "speak"),
	}
}

func singleToolView(t *testing.T, name string) tool.TurnToolView {
	t.Helper()

	catalog, _, err := tool.BuildEnvironmentToolCatalog(&protocolv1alpha2.CapabilityList{
		Capabilities: []*protocolv1alpha2.Capability{{
			Name:            name,
			Description:     "Run " + name,
			InputSchemaJson: `{"type":"object"}`,
		}},
	})
	if err != nil {
		t.Fatalf("BuildEnvironmentToolCatalog returned error: %v", err)
	}
	return catalog.Snapshot()
}

func assertInvalidInputError(t *testing.T, err error, want string) {
	t.Helper()

	if !errors.Is(err, agentcontext.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want message containing %q", err, want)
	}
}

func assertStringContainsAll(t *testing.T, content string, values ...string) {
	t.Helper()

	for _, want := range values {
		if !strings.Contains(content, want) {
			t.Fatalf("content missing %q:\n%s", want, content)
		}
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}
