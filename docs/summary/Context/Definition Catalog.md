# Definition Catalog

> Status: Phase7.1 Runtime Contract
> Scope: Static Game Definition / Agent Definition catalog loaded by Runtime startup.

Definition Catalog 是 Runtime 的静态定义目录。它回答：

```text
这个 game_id 对应什么游戏基础定义？
这个 game_id + definition_id 对应什么 Agent / Archetype 定义？
```

它不回答当前 world 发生了什么，也不保存 Agent 的经历、关系、位置、任务或运行时状态。

---

# 1. Directory Layout

Runtime 通过 `agent.json` 中的 `definition_catalog_root` 配置 Definition Catalog 根目录。

当前约定：

```text
runtime/config/games/
└── <game_id>/
    ├── agent.json
    └── definitions/
        ├── game.json
        ├── <agent-definition>.json
        └── <agent-archetype>.json
```

示例：

```text
runtime/config/games/stardew-valley/definitions/game.json
runtime/config/games/stardew-valley/definitions/npc-abigail.json
runtime/config/games/stardew-valley/definitions/npc-linus.json
runtime/config/games/stardew-valley/definitions/archetype-town-villager.json
```

`definitions/game.json` 是当前 game scope 的 Game Definition。

`definitions/*.json` 中除 `game.json` 以外的 JSON 文件，是当前 game scope 下的 Agent Definition / Archetype Definition。

Phase7.1 Runtime 不递归读取子目录；`definitions/agents/*.json` 没有 Catalog 语义。

---

# 2. Loading Model

Runtime 启动时加载 Definition Catalog，一次加载后在运行期间保持静态。

```text
definition_catalog_root unset
    -> empty catalog fallback

definition_catalog_root set
    -> startup load catalog
```

加载规则：

1. `definition_catalog_root` 已配置时，根目录必须存在且可读取。
2. 根目录下每个一级目录表示一个 `game_id` scope。
3. `<game_id>/definitions/` 缺失时，该 game 没有 Game Definition 或 Agent Definition，Turn 继续 fallback。
4. `<game_id>/definitions/game.json` 缺失时，该 game 没有 Game Definition，Turn 继续 fallback。
5. `<game_id>/definitions/*.json` 中除 `game.json` 以外没有其他 JSON 文件时，该 game 没有 Agent Definition，Turn 继续 fallback。
6. 已存在的 Definition 文件必须是合法 JSON。
7. 已存在的 Definition 文件必须使用受支持的 `schema_version`。
8. 已存在的 Definition 文件必须包含对应类型的 required identity fields。
9. Definition 文件中的 `game_id` 必须与所在 game scope 一致。
10. Catalog 内不能出现重复 `game_id`。
11. 同一 `game_id` 下不能出现重复 `definition_id`。

运行期间不做动态刷新，不从 Adapter 拉取 Definition，不把缺失 Definition 当作配置损坏。

---

# 3. Identity Semantics

## 3.1 game_id

`game_id` 标识一个游戏类型或游戏适配域。

比较语义：

```text
trim leading/trailing whitespace
case-sensitive exact match
no lowercase normalization
no alias resolution
```

`stardew-valley` 与 `Stardew-Valley` 是不同 ID。

## 3.2 entity_id

`entity_id` 标识某个 world 内的具体游戏实体。

它参与 Agent 实例身份：

```text
AgentSessionKey = game_id + world_id + entity_id
```

Memory、Trace 和 per-agent execution lane 使用 `AgentSessionKey` 隔离。

## 3.3 definition_id

`definition_id` 标识可复用 Agent Definition / Archetype Definition。

它参与 Definition lookup：

```text
AgentDefinition lookup = game_id + definition_id
```

比较语义：

```text
trim leading/trailing whitespace
case-sensitive exact match
no lowercase normalization
no alias resolution
```

`definition_id` 来自 canonical Target `EntityRef.definition_id`。

`definition_id` 可以等于 `entity_id`，也可以不同于 `entity_id`。

固定 NPC 可以使用：

```text
entity_id = npc:Abigail
definition_id = npc:Abigail
```

动态实体或可复用模板可以使用：

```text
entity_id = villager:alpha
definition_id = villager/farmer

entity_id = villager:beta
definition_id = villager/farmer
```

这两个实体共享同一个 Agent Definition，但 Memory 仍按各自 `AgentSessionKey` 隔离。

---

# 4. Game Definition

Game Definition 描述一个游戏的静态基础设定、世界规则和叙事边界。

Scope：

```text
game_id
```

JSON fields：

| Field | Required | Model Visible | Notes |
| --- | --- | --- | --- |
| `schema_version` | yes | no | 当前支持 `v1alpha1` |
| `game_id` | yes | no | 必须与目录 scope 一致 |
| `title` | no | yes | 游戏展示名称 |
| `summary` | no | yes | 游戏整体摘要 |
| `world_rules` | no | yes | 稳定世界规则 |
| `lore` | no | yes | 基础世界观信息 |
| `narrative_constraints` | no | yes | 叙事和角色行为边界 |
| `source_version` | no | no | 内容来源或版本备注 |

最小合法示例：

```json
{
  "schema_version": "v1alpha1",
  "game_id": "stardew-valley"
}
```

带内容示例：

```json
{
  "schema_version": "v1alpha1",
  "game_id": "stardew-valley",
  "title": "Stardew Valley",
  "summary": "A rural life simulation game centered on farming, community, exploration, and daily routines.",
  "world_rules": [
    "A year has four seasons.",
    "Each season has 28 days."
  ],
  "lore": [
    "Pelican Town is a small rural town with recurring local residents."
  ],
  "narrative_constraints": [
    "Keep responses grounded in the game's cozy rural setting."
  ],
  "source_version": "mvp0"
}
```

Game Definition 不包含当前存档事实，例如当天日期、天气、玩家位置、NPC 当前坐标或好感度。

---

# 5. Agent Definition

Agent Definition 描述一个可复用 Agent / Archetype 的静态身份、语气和行为约束。

Scope：

```text
game_id + definition_id
```

JSON fields：

| Field | Required | Model Visible | Notes |
| --- | --- | --- | --- |
| `schema_version` | yes | no | 当前支持 `v1alpha1` |
| `game_id` | yes | no | 必须与目录 scope 一致 |
| `definition_id` | yes | no | 在同一 `game_id` 下唯一 |
| `identity` | no | yes | 角色或 archetype 身份摘要 |
| `personality` | no | yes | 稳定人格特征 |
| `speech_style` | no | yes | 说话风格 |
| `preferences` | no | yes | 稳定偏好 |
| `behavior_guidelines` | no | yes | 行为约束和互动原则 |
| `source_version` | no | no | 内容来源或版本备注 |

最小合法示例：

```json
{
  "schema_version": "v1alpha1",
  "game_id": "stardew-valley",
  "definition_id": "npc:Abigail"
}
```

固定 NPC 示例：

```json
{
  "schema_version": "v1alpha1",
  "game_id": "stardew-valley",
  "definition_id": "npc:Abigail",
  "identity": "Abigail is a young resident of Pelican Town.",
  "personality": [
    "curious",
    "adventurous",
    "independent"
  ],
  "speech_style": [
    "casual",
    "direct",
    "slightly mischievous"
  ],
  "preferences": [
    "adventure",
    "games",
    "unusual discoveries"
  ],
  "behavior_guidelines": [
    "Respond as the character, not as a system assistant.",
    "Keep everyday dialogue concise."
  ],
  "source_version": "mvp0"
}
```

共享 archetype 示例：

```json
{
  "schema_version": "v1alpha1",
  "game_id": "stardew-valley",
  "definition_id": "archetype:town_villager",
  "identity": "A reusable archetype for ordinary town residents.",
  "personality": [
    "neighborly",
    "routine-oriented"
  ],
  "speech_style": [
    "plain",
    "local",
    "brief"
  ],
  "behavior_guidelines": [
    "Treat the current EntityRef display name as the concrete character name.",
    "Do not assume memories or relationships that are not present in Runtime context."
  ],
  "source_version": "mvp0"
}
```

Agent Definition 不包含当前记忆、最近对话、实时任务、当前地点、当前情绪状态或临时关系变化。

---

# 6. Runtime Data Flow

Phase7.1 的数据流：

```text
GameEvent
  -> Gateway resolves canonical Target EntityRef
  -> AgentSessionKey = game_id + world_id + target.entity_id
  -> AgentInstanceDescriptor = AgentSessionKey + canonical Target fields
  -> Game Definition lookup by game_id
  -> Agent Definition lookup by game_id + descriptor.definition_id
  -> Context Builder
  -> Renderer
  -> Model Request
```

`Context Builder` 不重新扫描 `GameEvent.entities` 来推断目标实体。

`Observation` 和 `ContextFact` 不承载 Definition 模板内容，也不替代 `EntityRef.definition_id`。

Renderer 在 Phase7.1 临时输出 `[Game Definition]`、`[Agent Definition]` 和 `[Agent Instance Descriptor]`，用于测试验证。最终稳定 Context Projection 结构归 Phase7.3。

---

# 7. Fallback And Fail-Fast

## 7.1 Startup fail-fast

以下情况是配置错误，Runtime 启动失败：

```text
configured definition_catalog_root missing or unreadable
configured Definition file unreadable
configured Definition file malformed JSON
unsupported schema_version
required identity field missing
Game Definition game_id mismatches directory scope
Agent Definition game_id mismatches directory scope
duplicate game_id
duplicate game_id + definition_id
```

## 7.2 Turn fallback

以下情况不是配置损坏，Turn 继续：

```text
definition_catalog_root unset
game_id has no Game Definition
target EntityRef has no definition_id
definition_id misses Agent Definition
```

fallback 只表示模型本轮拿不到对应 Definition 内容。Runtime 不伪造 Game Definition 或 Agent Definition。

---

# 8. Authoring Rules

1. 把稳定、跨存档复用的内容写入 Definition。
2. 把当前存档、当前场景、当前对话、当前关系和短期事件留给 Observation、ContextFact、Memory 或后续 Context Source。
3. `game.json` 写游戏规则、世界观和叙事边界。
4. Agent Definition 写角色或 archetype 的稳定身份、语气、偏好和行为约束。
5. 固定 NPC 可以让 `definition_id == entity_id`。
6. 可复用模板应让多个实体共享同一个 `definition_id`。
7. 文件名只服务于组织和人工阅读；Catalog 身份以 JSON 字段为准。
8. 不用大小写变化表达 alias。
9. 不把 `prompt.npc_style` 写回某个 Agent Definition；它仍是全局 fallback 语气。

---

# 9. Verification

Definition Catalog 相关 Runtime 验证：

```powershell
go test ./runtime/internal/definition
go test ./runtime/internal/agent
go test ./runtime/internal/gateway
go test ./runtime/...
```

Stardew 实机 smoke 可以验证配置路径、启动方式和 Adapter 到 Runtime 的真实链路，但 Phase7.1 的核心泛化语义由 Runtime 测试覆盖。
