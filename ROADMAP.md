# Roadmap

World Is Agent is an experimental MVP0 project. This public roadmap uses Now / Next / Later so the current direction stays readable while detailed phase plans continue to live under `docs/`.

## Now

Complete the Phase7 Context Subsystem for the Runtime + Stardew vertical slice.

Focus:

- Freeze the Context contracts before implementation expands.
- Add runtime-owned Game Definition and Agent Definition sources.
- Build scoped Context projection from event, observation, memory, transcript, definitions, and tools.
- Replace process-global environment tool exposure with EnvironmentSession-scoped Tool View snapshots.
- Make context selection, budget limits, and build diagnostics deterministic and testable.
- Validate the full path with real Stardew NPC dialogue.

Exit signals:

- A model request clearly shows the right game definition, agent definition, event facts, observation, memory, transcript, and tool view.
- Two Stardew NPCs can load different definitions without adapter-side prompt assembly.
- Tool exposure and tool execution use the same immutable Turn Tool View.
- Phase5 multi-step and Phase6 async action behavior remain stable.
- Current capabilities and limits stay captured in `docs/STATUS.md`.

## Next

Add Persistent Recent Memory, then environment recovery.

Focus:

- Schema version, compatible reads, migration, and explicit reset behavior for recent memory.
- Bounded retention and idempotent writes for per-AgentSession memory.
- Adapter reconnect and EnvironmentSession recovery.
- Heartbeat and liveness semantics.
- Capability registry scoping across reconnects.
- Disconnect, late result, and idempotency behavior.
- Durable continuation strategy for async actions.

Exit signals:

- Agent memory can survive process restart for a selected backend.
- Runtime restart and adapter reconnect behavior are specified and tested.
- Async action outcomes have clear recovery semantics.

## Later

Grow WIA from one validated adapter into a reusable multi-world agent harness.

Focus:

- Scenario evaluation and regression suites.
- Fault injection for runtime, adapter, and provider boundaries.
- Adapter conformance tests.
- A second real adapter outside Stardew Valley.
- Versioned releases, migration policy, and trace retention/export.

Exit signals:

- New adapters can be built against documented conformance checks.
- Runtime behavior is measurable through scenario evaluation.
- Public releases have stable install, upgrade, and compatibility notes.

## Detailed Plans

Detailed phase plans, ADRs, and acceptance records are kept under [docs/](docs/README.md).
