package context_test

import (
	"errors"
	"strings"
	"testing"

	protocolv1alpha2 "gameagent/protocol/gen/go/gameagent/protocol/v1alpha2"
	agentcontext "gameagent/runtime/internal/context"
	"gameagent/runtime/internal/definition"
	"gameagent/runtime/internal/session"
	"gameagent/runtime/internal/tool"
)

func TestEngineBuildCreatesContextProjectionFromValidatedInput(t *testing.T) {
	engine := agentcontext.NewEngine(agentcontext.EngineConfig{})
	input := validEngineInput(t)

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
	if projection.Event.GetEventId() != "event-1" {
		t.Fatalf("Event.EventId = %q, want event-1", projection.Event.GetEventId())
	}
	if projection.Observation.GetEntityId() != "npc:Abigail" {
		t.Fatalf("Observation.EntityId = %q, want npc:Abigail", projection.Observation.GetEntityId())
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
