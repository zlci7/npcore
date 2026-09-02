package definition_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gameagent/runtime/internal/definition"
)

func TestNewCatalogFindsScopedDefinitionsWithTrimmedKeys(t *testing.T) {
	catalog, err := definition.NewCatalog(
		[]definition.GameDefinition{
			{
				SchemaVersion:        " v1alpha1 ",
				GameID:               " stardew-valley ",
				Title:                "Stardew Valley",
				Summary:              "A farming life sim.",
				WorldRules:           []string{"Each season has 28 days."},
				Lore:                 []string{"Pelican Town is a small rural town."},
				NarrativeConstraints: []string{"Stay in character."},
				SourceVersion:        "fixture",
			},
		},
		[]definition.AgentDefinition{
			{
				SchemaVersion:      "v1alpha1",
				GameID:             "stardew-valley",
				DefinitionID:       " npc:Abigail ",
				Identity:           "Abigail is a Stardew Valley villager.",
				Personality:        []string{"adventurous"},
				SpeechStyle:        []string{"direct"},
				Preferences:        []string{"amethyst"},
				BehaviorGuidelines: []string{"Respond briefly."},
				SourceVersion:      "fixture",
			},
		},
	)
	if err != nil {
		t.Fatalf("NewCatalog returned error: %v", err)
	}

	game, ok := catalog.FindGame(" stardew-valley ")
	if !ok {
		t.Fatal("FindGame did not find stardew-valley")
	}
	if game.GameID != "stardew-valley" {
		t.Fatalf("GameID = %q, want stardew-valley", game.GameID)
	}
	if len(game.WorldRules) != 1 || game.WorldRules[0] != "Each season has 28 days." {
		t.Fatalf("WorldRules = %+v", game.WorldRules)
	}

	agent, ok := catalog.FindAgent("stardew-valley", " npc:Abigail ")
	if !ok {
		t.Fatal("FindAgent did not find npc:Abigail")
	}
	if agent.DefinitionID != "npc:Abigail" {
		t.Fatalf("DefinitionID = %q, want npc:Abigail", agent.DefinitionID)
	}
	if len(agent.BehaviorGuidelines) != 1 || agent.BehaviorGuidelines[0] != "Respond briefly." {
		t.Fatalf("BehaviorGuidelines = %+v", agent.BehaviorGuidelines)
	}
}

func TestNewCatalogRejectsDuplicateGameID(t *testing.T) {
	_, err := definition.NewCatalog(
		[]definition.GameDefinition{
			{SchemaVersion: "v1alpha1", GameID: "stardew-valley"},
			{SchemaVersion: "v1alpha1", GameID: " stardew-valley "},
		},
		nil,
	)
	if err == nil {
		t.Fatal("NewCatalog returned nil error, want duplicate game_id error")
	}
	if !strings.Contains(err.Error(), "duplicate game_id") {
		t.Fatalf("error = %v, want duplicate game_id", err)
	}
}

func TestNewCatalogRejectsDuplicateAgentDefinitionWithinGame(t *testing.T) {
	_, err := definition.NewCatalog(
		nil,
		[]definition.AgentDefinition{
			{SchemaVersion: "v1alpha1", GameID: "game-a", DefinitionID: "npc:Abigail"},
			{SchemaVersion: "v1alpha1", GameID: "game-a", DefinitionID: " npc:Abigail "},
		},
	)
	if err == nil {
		t.Fatal("NewCatalog returned nil error, want duplicate agent definition error")
	}
	if !strings.Contains(err.Error(), "duplicate agent definition") {
		t.Fatalf("error = %v, want duplicate agent definition", err)
	}
}

func TestNewCatalogAllowsSameDefinitionIDAcrossGames(t *testing.T) {
	catalog, err := definition.NewCatalog(
		nil,
		[]definition.AgentDefinition{
			{SchemaVersion: "v1alpha1", GameID: "game-a", DefinitionID: "npc:Guide", Identity: "Guide A"},
			{SchemaVersion: "v1alpha1", GameID: "game-b", DefinitionID: "npc:Guide", Identity: "Guide B"},
		},
	)
	if err != nil {
		t.Fatalf("NewCatalog returned error: %v", err)
	}

	agentA, ok := catalog.FindAgent("game-a", "npc:Guide")
	if !ok {
		t.Fatal("FindAgent did not find game-a npc:Guide")
	}
	agentB, ok := catalog.FindAgent("game-b", "npc:Guide")
	if !ok {
		t.Fatal("FindAgent did not find game-b npc:Guide")
	}
	if agentA.Identity != "Guide A" || agentB.Identity != "Guide B" {
		t.Fatalf("agents crossed game scope: game-a=%q game-b=%q", agentA.Identity, agentB.Identity)
	}
}

func TestNewCatalogRejectsMissingRequiredIdentityFields(t *testing.T) {
	tests := []struct {
		name   string
		games  []definition.GameDefinition
		agents []definition.AgentDefinition
		want   string
	}{
		{
			name:  "game schema_version",
			games: []definition.GameDefinition{{GameID: "game-a"}},
			want:  "schema_version",
		},
		{
			name:  "game game_id",
			games: []definition.GameDefinition{{SchemaVersion: "v1alpha1"}},
			want:  "game_id",
		},
		{
			name:   "agent schema_version",
			agents: []definition.AgentDefinition{{GameID: "game-a", DefinitionID: "npc:Abigail"}},
			want:   "schema_version",
		},
		{
			name:   "agent game_id",
			agents: []definition.AgentDefinition{{SchemaVersion: "v1alpha1", DefinitionID: "npc:Abigail"}},
			want:   "game_id",
		},
		{
			name:   "agent definition_id",
			agents: []definition.AgentDefinition{{SchemaVersion: "v1alpha1", GameID: "game-a"}},
			want:   "definition_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := definition.NewCatalog(tt.games, tt.agents)
			if err == nil {
				t.Fatal("NewCatalog returned nil error, want validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestNewCatalogRejectsUnsupportedSchemaVersion(t *testing.T) {
	_, err := definition.NewCatalog(
		[]definition.GameDefinition{{SchemaVersion: "v9", GameID: "game-a"}},
		nil,
	)
	if err == nil {
		t.Fatal("NewCatalog returned nil error, want unsupported schema error")
	}
	if !strings.Contains(err.Error(), "unsupported schema_version") {
		t.Fatalf("error = %v, want unsupported schema_version", err)
	}
}

func TestLoadCatalogFromDirLoadsStaticDefinitionFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "stardew-valley", "game.json"), `{
  "schema_version": "v1alpha1",
  "game_id": "stardew-valley",
  "title": "Stardew Valley",
  "summary": "A farming life sim.",
  "world_rules": ["Each season has 28 days."],
  "lore": ["Pelican Town is a small rural town."],
  "narrative_constraints": ["Stay grounded in Stardew Valley."],
  "source_version": "test"
}`)
	writeFile(t, filepath.Join(root, "stardew-valley", "agents", "abigail.json"), `{
  "schema_version": "v1alpha1",
  "game_id": "stardew-valley",
  "definition_id": "npc:Abigail",
  "identity": "Abigail is a villager in Pelican Town.",
  "personality": ["adventurous"],
  "speech_style": ["brief"],
  "preferences": ["amethyst"],
  "behavior_guidelines": ["Stay in character."],
  "source_version": "test"
}`)

	catalog, err := definition.LoadCatalogFromDir(root)
	if err != nil {
		t.Fatalf("LoadCatalogFromDir returned error: %v", err)
	}
	if _, ok := catalog.FindGame("stardew-valley"); !ok {
		t.Fatal("FindGame did not find loaded game definition")
	}
	if _, ok := catalog.FindAgent("stardew-valley", "npc:Abigail"); !ok {
		t.Fatal("FindAgent did not find loaded agent definition")
	}
}

func TestLoadCatalogFromDirMissingDirectoryReturnsEmptyCatalog(t *testing.T) {
	catalog, err := definition.LoadCatalogFromDir(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("LoadCatalogFromDir returned error: %v", err)
	}
	if _, ok := catalog.FindGame("stardew-valley"); ok {
		t.Fatal("FindGame found a game in empty fallback catalog")
	}
	if _, ok := catalog.FindAgent("stardew-valley", "npc:Abigail"); ok {
		t.Fatal("FindAgent found an agent in empty fallback catalog")
	}
}

func TestLoadCatalogFromDirRejectsMalformedJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "stardew-valley", "game.json"), `{`)

	_, err := definition.LoadCatalogFromDir(root)
	if err == nil {
		t.Fatal("LoadCatalogFromDir returned nil error, want malformed JSON error")
	}
	if !strings.Contains(err.Error(), "parse game definition") {
		t.Fatalf("error = %v, want parse game definition", err)
	}
}

func TestLoadCatalogFromDirRejectsPathScopeMismatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "stardew-valley", "game.json"), `{
  "schema_version": "v1alpha1",
  "game_id": "other-game"
}`)

	_, err := definition.LoadCatalogFromDir(root)
	if err == nil {
		t.Fatal("LoadCatalogFromDir returned nil error, want scope mismatch error")
	}
	if !strings.Contains(err.Error(), "scope mismatch") {
		t.Fatalf("error = %v, want scope mismatch", err)
	}
}

func TestLoadCatalogFromDirRejectsUnreadableAgentPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "game-a", "game.json"), `{
  "schema_version": "v1alpha1",
  "game_id": "game-a"
}`)
	agentsPath := filepath.Join(root, "game-a", "agents")
	if err := os.WriteFile(agentsPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write agents path: %v", err)
	}

	_, err := definition.LoadCatalogFromDir(root)
	if err == nil {
		t.Fatal("LoadCatalogFromDir returned nil error, want unreadable agents path error")
	}
	if !strings.Contains(err.Error(), "read agent definitions") {
		t.Fatalf("error = %v, want read agent definitions", err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("make dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
