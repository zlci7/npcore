# GameAgent MVP0 Phase7.1 技术开发与验收方案

> **Status:** Implementation Plan Draft
> **Date:** 2026-09-02
> **Phase:** Phase7.1 Definition Sources 与 Instance Descriptor
> **Roadmap Baseline:** [GameAgent 阶段规划](../summary/GameAgent%20阶段规划.md) v1.9
> **Previous Gate:** [GameAgent MVP0 Phase7.0 技术开发与验收方案](GameAgent%20MVP0%20Phase7.0%20技术开发与验收方案.md) Accepted
> **Code Baseline:** `main` @ `e4f3490`

---

# 1. 阶段目标

Phase7.1 让 Game Definition、Agent Definition、Agent Instance Descriptor 和 canonical Target EntityRef 进入 Runtime 可测试链路。

本阶段主要证明：

```text
Runtime 可以根据一次 GameEvent 的目标实体，稳定拿到：

1. 唯一、规范化、已校验的 Target EntityRef；
2. 当前 game_id 对应的 Game Definition；
3. 当前 game_id + definition_id 对应的 Agent Definition；
4. 当前 Agent 实例自己的 Descriptor。
```

这一阶段不追求完整 Context Engine。它先把“角色定义真的生效”的地基打稳，让后续 7.2 到 7.5 可以在同一套身份和定义输入上继续开发。

---

# 2. 非目标

Phase7.1 不做：

```text
Environment-scoped Tool View
Turn Tool View snapshot
完整 Context Engine
Context Projection 分层选择
最终 Context Budget
BuildReport
综合 diagnostics
Persistent Memory backend
Vector retrieval
Semantic compression
Stardew Adapter 行为改造
Stardew 实机对话验收
最终 Model Request 段落硬验收
```

本阶段可以临时把 Definition 内容接入现有 `AgentContext` 和 Renderer，以便测试模型输入能看到角色定义；但不把最终 prompt 结构作为验收硬标准。

---

# 3. 当前代码事实

## 3.1 协议已有 EntityRef

当前协议已有：

```text
protocol/proto/gameagent.proto

EntityRef:
    entity_id
    entity_type
    display_name
    definition_id

GameEvent:
    entities[]
    world_id
    target_entity_id

Observation:
    entity_id
    world_id
    nearby_entities[]
```

Phase7.1 不新增协议字段。Runtime 继续复用现有 `EntityRef`。

## 3.2 Gateway 已有基础 target 校验

当前 Gateway 在 `runtime/internal/gateway/gateway.go` 的 `resolveAgentSessionKey` 中已经校验：

```text
event_type 非空
target_entity_id 非空
target_entity_id 出现在 GameEvent.entities 中
game_id + world_id + target_entity_id 可以解析为 AgentSessionKey
```

Phase7.1 在这个边界上继续收敛 canonical Target EntityRef，不把 target 解析放到 Adapter prompt 拼接里。

## 3.3 Context Builder 已有临时 Descriptor

当前 `runtime/internal/context/context.go` 中：

```text
AgentDescriptor:
    EntityID
    DefinitionID
```

`Builder` 会从 `GameEvent.entities` 中寻找 `target_entity_id` 对应的 `definition_id`。

Phase7.1 需要把这里升级为更完整的 Agent Instance Descriptor，并接入加载出来的 Game / Agent Definition。

## 3.4 Runtime 配置还没有 Definition Source

当前配置入口：

```text
runtime/config/agent.json
runtime/config/games/stardew-valley/agent.json
runtime/internal/agent/config.go
runtime/cmd/server/main.go
```

这些配置只覆盖 Runtime 运行参数和 prompt 默认值，还没有 Definition 目录、Definition Source 或 Catalog 加载。

---

# 4. 设计范围

## 4.1 Canonical Target EntityRef

Phase7.1 在 Gateway / pre-turn validation 边界产出 canonical Target EntityRef。

最小规则：

```text
输入：
    GameEvent.target_entity_id
    GameEvent.entities[]

输出：
    一个确定的 EntityRef

规则：
    target_entity_id 必须非空
    entities 中必须存在目标 entity
    目标 entity_id 归一化后必须与 target_entity_id 一致
    如果 entities 中重复出现同一目标 entity，字段完全一致时可接受
    如果重复目标 entity 的 entity_type / display_name / definition_id 冲突，拒绝事件
```

验收重点：

```text
不存在目标 entity -> REJECTED
重复且一致 -> ACCEPTED
重复但冲突 -> REJECTED
```

canonical Target EntityRef 只表示“这次事件的目标实体是谁”。它不替代 `AgentSessionKey`，也不改变 Memory 的 scope。

## 4.2 Game Definition

Runtime 按 `game_id` 加载 Game Definition。

最小字段：

```text
schema_version
game_id
title
summary
world_rules[]
lore[]
narrative_constraints[]
source_version
```

读取规则：

```text
game_id 对应一个 Game Definition
同一个 Catalog 中重复 game_id 视为配置错误
文件存在但格式损坏或必填字段缺失时启动 fail-fast
game_id 没有配置 Definition 时允许使用 empty fallback
```

empty fallback 表示 Runtime 仍可运行，但模型不会获得额外游戏定义内容。

## 4.3 Agent Definition

Runtime 按 `game_id + definition_id` 加载 Agent Definition。

最小字段：

```text
schema_version
game_id
definition_id
identity
personality[]
speech_style[]
preferences[]
behavior_rules[]
relationship_hints[]
source_version
```

读取规则：

```text
definition_id 来自 canonical Target EntityRef
definition_id 可以不同于 entity_id
多个 entity 可以共享同一个 definition_id
同一个 game_id 下重复 definition_id 视为配置错误
文件存在但格式损坏或必填字段缺失时启动 fail-fast
entity 没有 definition_id 或 definition_id 未命中时允许使用 fallback
```

fallback 继续使用现有 `prompt.npc_style` 作为全局默认语气，不把它写回某个 Agent Definition。

## 4.4 Agent Instance Descriptor

Agent Instance Descriptor 描述当前被控制的实体实例，而不是角色定义本身。

最小字段：

```text
game_id
world_id
entity_id
entity_type
display_name
definition_id
session_key
definition_status
```

其中：

```text
game_id / world_id / entity_id 来自 AgentSessionKey
entity_type / display_name / definition_id 来自 canonical Target EntityRef
session_key 复用现有 AgentSessionKey
definition_status 记录 loaded / fallback
```

多实体共享 `definition_id` 时：

```text
npc:Abigail#world-a 与 npc:Abigail#world-b 的 Memory 仍按 AgentSessionKey 分开
npc:Abigail 与 npc:Abigail_clone 可以共享同一 Agent Definition
Descriptor 必须保留各自 entity_id / world_id / display_name
```

## 4.5 Definition Source

Phase7.1 提供最小 Definition Source 能力。

建议形态：

```text
GameDefinitionSource
    LoadGameDefinition(game_id)

AgentDefinitionSource
    LoadAgentDefinition(game_id, definition_id)
```

生产路径使用静态文件或启动时 Catalog。测试路径可以使用内存实现。

最小静态目录建议：

```text
runtime/config/games/stardew-valley/definitions/game.json
runtime/config/games/stardew-valley/definitions/agents/*.json
```

这里的目录是 Phase7.1 候选方案，Review 通过后再进入代码实现。

## 4.6 最小 Stardew Definition fixtures

Phase7.1 只需要少量 fixtures 证明加载链路成立。

候选内容：

```text
stardew-valley Game Definition

Agent Definitions:
    npc:Abigail
    npc:Linus
    archetype:town_villager
```

验收用例覆盖：

```text
definition_id == entity_id
definition_id != entity_id
多个 entity 共享 definition_id
missing definition fallback
malformed configured definition fail-fast
```

fixtures 的内容只用于加载和最小渲染验证，不在本阶段追求完整 Stardew 人设校对。

---

# 5. 预计代码改动

## 5.1 Runtime Definition 包

新增候选包：

```text
runtime/internal/definition
```

职责：

```text
定义 GameDefinition / AgentDefinition / AgentInstanceDescriptor
提供 Source 接口
提供内存 Source
提供静态文件 Catalog loader
完成字段校验、重复检查和 scope 检查
```

## 5.2 Gateway target 解析

预计修改：

```text
runtime/internal/gateway/gateway.go
runtime/internal/gateway/*_test.go
```

职责：

```text
在 pre-turn validation 中解析 canonical Target EntityRef
保留现有 AgentSessionKey 解析
对重复一致和重复冲突给出确定行为
```

Gateway 可以继续把原始 `GameEvent` 传入 AgentLoop。是否额外传递 canonical Target EntityRef，由 Phase7.1 implementation 方案 Review 后确定。

## 5.3 Agent Loop 接线

预计修改：

```text
runtime/internal/agent/loop.go
runtime/internal/agent/config.go
runtime/cmd/server/main.go
```

职责：

```text
启动时加载 Definition Catalog
在 buildModelRequest 前按 game_id / definition_id 取 Definition
把 Definition 与 Descriptor 作为结构化输入交给 Context Builder
```

如果 Definition Source 不可用，Runtime 使用 empty fallback；如果配置文件损坏，启动失败。

## 5.4 Context Builder / Renderer 临时接入

预计修改：

```text
runtime/internal/context/context.go
runtime/internal/context/renderer.go
runtime/internal/context/builder_test.go
```

职责：

```text
扩展 BuildInput / AgentContext，携带 Game Definition、Agent Definition 和 Agent Instance Descriptor
Renderer 临时渲染 Definition 内容，证明模型输入可以看到这些信息
保留 Current Observation 优先于 Recent Memory 的语义
```

最终 Context Projection 的分层结构属于 Phase7.3。

## 5.5 Stardew fixtures

预计新增：

```text
runtime/config/games/stardew-valley/definitions/game.json
runtime/config/games/stardew-valley/definitions/agents/*.json
```

职责：

```text
提供最小可加载内容
证明 Stardew 的 game_id 与 Agent Definition scope 匹配
覆盖共享 definition_id 的测试场景
```

---

# 6. 错误与 fallback

## 6.1 Event 级错误

以下错误在 GameEvent admission 阶段返回 `REJECTED`：

```text
target_entity_id 缺失
target_entity_id 不在 entities 中
目标 EntityRef 重复且字段冲突
game_id / world_id / entity_id 无法形成 AgentSessionKey
```

## 6.2 启动级错误

以下错误启动 fail-fast：

```text
已配置的 Definition 文件不是合法 JSON
已配置的 Definition 缺少必填字段
Game Definition 的 game_id 与所在 game scope 不一致
Agent Definition 的 game_id 与所在 game scope 不一致
同一 Catalog 中存在重复 game_id
同一 game_id 下存在重复 definition_id
```

## 6.3 Turn 级 fallback

以下情况进入 fallback：

```text
game_id 没有配置 Game Definition
target EntityRef 没有 definition_id
definition_id 没有命中 Agent Definition
```

fallback 不阻塞 Turn。它只让本次模型输入缺少对应 Definition 内容，并记录最小状态，方便测试断言。

---

# 7. 测试计划

## 7.1 Gateway / target tests

覆盖：

```text
target_entity_id 不存在于 entities -> REJECTED
target EntityRef 重复且字段一致 -> ACCEPTED
target EntityRef 重复但 definition_id 冲突 -> REJECTED
target EntityRef 重复但 display_name 冲突 -> REJECTED
target EntityRef 中 definition_id 与 entity_id 不同 -> ACCEPTED
```

## 7.2 Definition loader tests

覆盖：

```text
按 game_id 加载 Game Definition
按 game_id + definition_id 加载 Agent Definition
missing Game Definition fallback
missing Agent Definition fallback
malformed JSON fail-fast
必填字段缺失 fail-fast
scope mismatch fail-fast
重复 game_id fail-fast
重复 game_id + definition_id fail-fast
```

## 7.3 Context Builder / Renderer tests

覆盖：

```text
Agent Instance Descriptor 包含 game_id / world_id / entity_id / entity_type / display_name / definition_id
definition_id != entity_id 时，模型输入使用 definition_id 加载出的 Agent Definition
多个 entity 共享 definition_id 时，Descriptor 不混淆 entity_id 和 session_key
fallback 时 Renderer 不输出伪造 Definition
Current Observation 仍然优先于 Recent Memory
```

## 7.4 Scope isolation tests

覆盖：

```text
同一 entity_id 在不同 world_id 下使用不同 AgentSessionKey
同一 definition_id 被多个 entity 共享时 Memory 不串线
不同 game_id 下同名 definition_id 不串线
```

## 7.5 回归测试

建议运行：

```powershell
go test ./runtime/...
```

如 Phase7.1 代码修改触碰协议生成或 Adapter fixture，再补充对应静态检查。Phase7.1 文档草稿阶段不运行 Go 测试。

---

# 8. 验收条件

Phase7.1 Accepted 前必须满足：

1. Phase7.1 技术方案 Review 通过。
2. Game Definition / Agent Definition 最小模型已定稿。
3. Definition Source 的生产路径和测试路径已定稿。
4. canonical Target EntityRef 的重复一致、重复冲突和缺失行为已定稿。
5. Agent Instance Descriptor 字段已定稿。
6. fallback 与 fail-fast 边界已定稿。
7. Stardew 最小 fixtures 路径和覆盖目标已定稿。
8. 预计改动文件和测试清单已明确。

Phase7.1 代码开发完成后必须满足：

1. Runtime 可按 `game_id` 加载 Game Definition。
2. Runtime 可按 `game_id + definition_id` 加载 Agent Definition。
3. `definition_id != entity_id` 有测试覆盖。
4. 多个 entity 共享 `definition_id` 时，Descriptor 和 Memory 不串线。
5. missing definition 走 fallback。
6. malformed configured definition 启动 fail-fast。
7. scope mismatch fail-fast。
8. Context Builder 能接收 Definition 和 Descriptor。
9. Renderer 临时输出 Definition 内容用于可测试验证。
10. `go test ./runtime/...` 通过。

本阶段不以最终 Model Request 段落作为硬验收。最终 Context Projection 和稳定渲染结构归 Phase7.3。

---

# 9. Review 关注点

Review Phase7.1 时重点看：

1. target entity 是否在 Runtime 边界被唯一确定。
2. `definition_id` 是否已经和 `entity_id` 解耦。
3. Definition loading 是否按 `game_id` 隔离。
4. 多实体共享 Definition 时，实例身份和 Memory scope 是否仍然独立。
5. fallback 是否只处理 missing，不吞掉损坏配置。
6. 是否把 Tool View、Budget、BuildReport 或 Stardew 实机验收提前放进本阶段。

---

# 10. 下一阶段衔接

Phase7.1 完成后进入 Phase7.2 技术方案编写。

Phase7.2 只接收 Phase7.1 的结果：

```text
canonical Target EntityRef
Game Definition
Agent Definition
Agent Instance Descriptor
Definition loading / fallback 状态
```

Phase7.2 不重新定义 Definition Source，不改变 AgentSessionKey 和 Memory scope。
