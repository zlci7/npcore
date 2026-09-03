# Phase7.1 Definition Sources Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development to implement each task. The controller runs a subagent code review after every M-step. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete Phase7.1 so Runtime can resolve a canonical target entity, load scoped Game / Agent Definitions, build an Agent Instance Descriptor, and expose those structures to the current Context path.

**Architecture:** Keep Phase7.1 as a small Runtime-owned layer. Definitions load once into a read-only Catalog, Gateway resolves one canonical Target EntityRef before AgentTurn admission, and AgentLoop passes Definition + Descriptor data into the existing Context Builder / Renderer without introducing a general Context Engine or Tool View lifecycle.

**Tech Stack:** Go, protobuf-generated protocol types, existing Runtime config, existing `go test ./runtime/...` test suite.

**Spec:** `docs/phase7/GameAgent MVP0 Phase7.1 技术开发与验收方案.md`

## Global Constraints

- Do not add protocol fields.
- Do not add a Definition database, hot reload, multi-backend Resolver, Repository, Report, or lifecycle framework.
- Runtime owns Definition loading and Context composition; Adapter does not assemble final prompts.
- `AgentSessionKey = game_id + world_id + entity_id` remains the Memory and Agent identity scope.
- `AgentDefinition` lookup uses `game_id + definition_id`; never look up by `definition_id` alone.
- Missing configured data falls back only when the Catalog is absent or a key is missing; configured unreadable, malformed, unsupported, duplicated, or scope-mismatched files fail at startup/load time.
- ID comparison trims leading/trailing whitespace and then uses exact, case-sensitive equality; Runtime performs no lowercase conversion, alias mapping, or other identity normalization.
- `definition_id` missing stays missing and uses fallback; Runtime must not infer `definition_id = entity_id`.

---

### Task M1: Definition Model, Catalog, And Static Loader

**Files:**
- Create: `runtime/internal/definition/definition.go`
- Create: `runtime/internal/definition/catalog.go`
- Create: `runtime/internal/definition/loader.go`
- Create: `runtime/internal/definition/catalog_test.go`

**Interfaces:**
- Produces: `definition.GameDefinition`, `definition.AgentDefinition`, `definition.AgentInstanceDescriptor`, `definition.Catalog`, `definition.NewCatalog`, `definition.LoadCatalogFromDir(root string) (Catalog, error)`, `Catalog.FindGame(gameID string) (GameDefinition, bool)`, `Catalog.FindAgent(gameID, definitionID string) (AgentDefinition, bool)`.
- Consumes: no earlier task output.

- [ ] Write failing tests for valid Catalog lookup, duplicate keys, malformed JSON, unsupported `schema_version`, missing required identity fields, scope mismatch between path game_id and file game_id, and missing directory empty Catalog fallback.
- [ ] Run `go test ./runtime/internal/definition` and verify the tests fail because the package does not exist.
- [ ] Implement the minimal definition package with immutable map copies, trim-based required field validation, static JSON loading, supported `schema_version`, and duplicate detection.
- [ ] Run `go test ./runtime/internal/definition` and verify it passes.
- [ ] Run `go test ./runtime/...` and verify existing packages still pass.
- [ ] Commit M1 and dispatch subagent review for the M1 diff.

### Task M2: Gateway Canonical Target EntityRef

**Files:**
- Modify: `runtime/internal/gateway/gateway.go`
- Modify: `runtime/internal/gateway/gateway_integration_test.go` or adjacent gateway tests.
- Modify if interface requires it: `runtime/internal/agent/loop.go`

**Interfaces:**
- Consumes: Phase7.1 target rules from the spec.
- Produces: Gateway pre-turn validation that returns both `session.AgentSessionKey` and one canonical `*protocolv1alpha2.EntityRef`.

- [ ] Write failing Gateway tests for target missing, duplicate identical target accepted, duplicate `definition_id` conflict rejected, duplicate `display_name` conflict rejected, duplicate `entity_type` conflict rejected, and `definition_id != entity_id` accepted.
- [ ] Run the focused gateway tests and verify they fail on current first-match behavior.
- [ ] Replace the existing target existence check with canonical target resolution using trim-then-exact ID equality and field conflict detection.
- [ ] Pass the canonical target into AgentLoop without making Context Builder rescan `GameEvent.entities`.
- [ ] Run focused gateway tests and `go test ./runtime/...`.
- [ ] Commit M2 and dispatch subagent review for the M2 diff.

### Task M3: Agent Instance Descriptor And AgentLoop / Context Wiring

**Files:**
- Modify: `runtime/internal/agent/loop.go`
- Modify: `runtime/internal/agent/config.go`
- Modify: `runtime/cmd/server/main.go`
- Modify: `runtime/internal/context/context.go`
- Modify: `runtime/internal/context/renderer.go`
- Modify: `runtime/internal/context/builder_test.go`
- Modify if needed: `runtime/internal/agent/loop_test.go`, `runtime/internal/gateway/gateway_integration_test.go`

**Interfaces:**
- Consumes: `definition.Catalog`, canonical Target EntityRef from M2.
- Produces: Context Builder input fields for Game Definition, Agent Definition, and Agent Instance Descriptor.

- [ ] Write failing Context Builder / Renderer tests showing Descriptor includes the session scope plus target fields, `definition_id != entity_id` loads by definition ID, shared definition IDs do not overwrite entity identity, and fallback does not render fabricated Definition content.
- [ ] Write failing AgentLoop wiring tests or update existing tests so `buildModelRequest` receives definitions from an injected Catalog.
- [ ] Add optional Definition Catalog config with a default disabled state and load configured definitions during server startup.
- [ ] Extend `Loop` construction and event handling to retain the canonical target, look up scoped definitions, and pass structured data into `context.BuildInput`.
- [ ] Render minimal Definition and Descriptor sections in the existing Renderer while preserving the current Observation-over-Memory instruction.
- [ ] Run focused context/agent/gateway tests and `go test ./runtime/...`.
- [ ] Commit M3 and dispatch subagent review for the M3 diff.

### Task M4: Stardew Fixtures, Minimal Projection Verification, And Full Tests

**Files:**
- Create: `runtime/config/games/stardew-valley/definitions/game.json`
- Create: `runtime/config/games/stardew-valley/definitions/npc-abigail.json`
- Create: `runtime/config/games/stardew-valley/definitions/npc-linus.json`
- Create: `runtime/config/games/stardew-valley/definitions/archetype-town-villager.json`
- Modify: focused loader/context/gateway tests as needed.

**Interfaces:**
- Consumes: M1 loader and M3 Runtime wiring.
- Produces: real Stardew definition fixtures that load at startup and support renderer verification.

- [ ] Write failing fixture-loading tests that load the Stardew definitions directory and assert `npc:Abigail`, `npc:Linus`, and `archetype:town_villager` are scoped to `stardew-valley`.
- [ ] Add the minimal Stardew Game and Agent Definition JSON fixtures.
- [ ] Add or update renderer tests that verify the model input contains different Agent Definition projections for two definition IDs and omits Definition text under fallback.
- [ ] Run focused fixture/context tests and then `go test ./runtime/...`.
- [ ] Commit M4 and dispatch subagent review for the M4 diff.
- [ ] Run final whole-branch review package through a subagent before handing the branch to the user for code review.
