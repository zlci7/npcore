package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	agentcontext "gameagent/runtime/internal/context"
	"os"
	"time"
)

const (
	defaultConfigPath = "runtime/config/agent.json"
	configPathEnv     = "GAMEAGENT_AGENT_CONFIG"

	defaultMemoryEnabled = true
)

type Config struct {
	TurnTimeout                   time.Duration
	LLMTimeout                    time.Duration
	ObserveTimeout                time.Duration
	ActionTimeout                 time.Duration
	ActionStartTimeout            time.Duration
	AsyncActionTimeout            time.Duration
	MemoryEnabled                 *bool
	RecentMemoryLimit             int
	MemoryContextSizeLimit        int
	MaxSteps                      int
	MaxToolCallsPerStep           int
	MaxToolCallsPerTurn           int
	MaxAsyncActionsPerTurn        int
	MaxParallelToolCalls          int
	MaxRequestBytes               int
	MaxSystemBytes                int
	MaxUserMessageBytes           int
	MaxDefinitionBytes            int
	MaxObservationBytes           int
	MaxEventBytes                 int
	MaxContextFactsBytes          int
	MaxRecentMemoryBytes          int
	MaxTranscriptBytes            int
	MaxToolCount                  int
	MaxToolDescriptionBytes       int
	MaxToolSchemaBytes            int
	MaxTotalToolSchemaBytes       int
	MaxToolResultOutputBytes      int
	MaxToolResultOutputDepth      int
	MaxToolResultOutputFields     int
	MaxToolResultOutputArrayItems int
	DefinitionCatalogRoot         string
	Prompt                        PromptConfig
}

type fileConfig struct {
	TurnTimeoutMS                 int64        `json:"turn_timeout_ms"`
	LLMTimeoutMS                  int64        `json:"llm_timeout_ms"`
	ObserveTimeoutMS              int64        `json:"observe_timeout_ms"`
	ActionTimeoutMS               int64        `json:"action_timeout_ms"`
	ActionStartTimeoutMS          int64        `json:"action_start_timeout_ms"`
	AsyncActionTimeoutMS          int64        `json:"async_action_timeout_ms"`
	MemoryEnabled                 *bool        `json:"memory_enabled"`
	RecentMemoryLimit             int          `json:"recent_memory_limit"`
	MemoryContextSizeLimit        int          `json:"memory_context_size_limit"`
	MaxSteps                      int          `json:"max_steps"`
	MaxToolCallsPerStep           int          `json:"max_tool_calls_per_step"`
	MaxToolCallsPerTurn           int          `json:"max_tool_calls_per_turn"`
	MaxAsyncActionsPerTurn        int          `json:"max_async_actions_per_turn"`
	MaxParallelToolCalls          int          `json:"max_parallel_tool_calls"`
	MaxRequestBytes               int          `json:"max_request_bytes"`
	MaxSystemBytes                int          `json:"max_system_bytes"`
	MaxUserMessageBytes           int          `json:"max_user_message_bytes"`
	MaxDefinitionBytes            int          `json:"max_definition_bytes"`
	MaxObservationBytes           int          `json:"max_observation_bytes"`
	MaxEventBytes                 int          `json:"max_event_bytes"`
	MaxContextFactsBytes          int          `json:"max_context_facts_bytes"`
	MaxRecentMemoryBytes          int          `json:"max_recent_memory_bytes"`
	MaxTranscriptBytes            int          `json:"max_transcript_bytes"`
	MaxToolCount                  int          `json:"max_tool_count"`
	MaxToolDescriptionBytes       int          `json:"max_tool_description_bytes"`
	MaxToolSchemaBytes            int          `json:"max_tool_schema_bytes"`
	MaxTotalToolSchemaBytes       int          `json:"max_total_tool_schema_bytes"`
	MaxToolResultOutputBytes      int          `json:"max_tool_result_output_bytes"`
	MaxToolResultOutputDepth      int          `json:"max_tool_result_output_depth"`
	MaxToolResultOutputFields     int          `json:"max_tool_result_output_fields"`
	MaxToolResultOutputArrayItems int          `json:"max_tool_result_output_array_items"`
	DefinitionCatalogRoot         string       `json:"definition_catalog_root"`
	Prompt                        PromptConfig `json:"prompt"`
}

type PromptConfig struct {
	Language        string `json:"language"`
	NPCStyle        string `json:"npc_style"`
	MaxSpeakChars   int    `json:"max_speak_chars"`
	ToolInstruction string `json:"tool_instruction"`
}

// DefaultConfig 返回 Agent Runtime 的默认运行配置。
// 默认开启短期 Memory，并加载当前 Runtime 预算上限。
func DefaultConfig() Config {
	budget := agentcontext.DefaultBudgetConfig()
	return Config{
		TurnTimeout:                   90 * time.Second,
		LLMTimeout:                    8 * time.Second,
		ObserveTimeout:                3 * time.Second,
		ActionTimeout:                 3 * time.Second,
		ActionStartTimeout:            3 * time.Second,
		AsyncActionTimeout:            45 * time.Second,
		MemoryEnabled:                 boolPtr(defaultMemoryEnabled),
		RecentMemoryLimit:             5,
		MemoryContextSizeLimit:        4096,
		MaxSteps:                      3,
		MaxToolCallsPerStep:           4,
		MaxToolCallsPerTurn:           6,
		MaxAsyncActionsPerTurn:        1,
		MaxParallelToolCalls:          4,
		MaxRequestBytes:               budget.MaxRequestBytes,
		MaxSystemBytes:                budget.MaxSystemBytes,
		MaxUserMessageBytes:           budget.MaxUserMessageBytes,
		MaxDefinitionBytes:            budget.MaxDefinitionBytes,
		MaxObservationBytes:           budget.MaxObservationBytes,
		MaxEventBytes:                 budget.MaxEventBytes,
		MaxContextFactsBytes:          budget.MaxContextFactsBytes,
		MaxRecentMemoryBytes:          budget.MaxRecentMemoryBytes,
		MaxTranscriptBytes:            budget.MaxTranscriptBytes,
		MaxToolCount:                  budget.MaxToolCount,
		MaxToolDescriptionBytes:       budget.MaxToolDescriptionBytes,
		MaxToolSchemaBytes:            budget.MaxToolSchemaBytes,
		MaxTotalToolSchemaBytes:       budget.MaxTotalToolSchemaBytes,
		MaxToolResultOutputBytes:      budget.MaxToolResultOutputBytes,
		MaxToolResultOutputDepth:      budget.MaxToolResultOutputDepth,
		MaxToolResultOutputFields:     budget.MaxToolResultOutputFields,
		MaxToolResultOutputArrayItems: budget.MaxToolResultOutputArrayItems,
		DefinitionCatalogRoot:         "",
		Prompt: PromptConfig{
			Language:        "Simplified Chinese",
			NPCStyle:        "自然、简短、符合当前游戏 NPC 的语气",
			MaxSpeakChars:   60,
			ToolInstruction: "Use available tools only when the NPC should take an environment action. Choose tools from their descriptions and input schemas. If a tool result reports rejected, failed, invalid, cancelled, or interrupted, use the next step to adjust or settle. Use settle when no more environment action is needed.",
		},
	}
}

// ConfigPathFromEnv 解析 Agent 配置文件路径。
// GAMEAGENT_AGENT_CONFIG 用于本地覆盖默认配置，未设置时读取 runtime/config/agent.json。
func ConfigPathFromEnv() string {
	if path := os.Getenv(configPathEnv); path != "" {
		return path
	}
	return defaultConfigPath
}

// LoadConfigFile 读取并解析 Agent 配置文件。
// 配置文件不存在时使用默认值，避免本地最小启动流程依赖额外文件。
func LoadConfigFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read agent config: %w", err)
	}

	var raw fileConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse agent config: %w", err)
	}

	cfg := Config{
		MemoryEnabled:                 raw.MemoryEnabled,
		TurnTimeout:                   durationMS(raw.TurnTimeoutMS),
		LLMTimeout:                    durationMS(raw.LLMTimeoutMS),
		ObserveTimeout:                durationMS(raw.ObserveTimeoutMS),
		ActionTimeout:                 durationMS(raw.ActionTimeoutMS),
		ActionStartTimeout:            durationMS(raw.ActionStartTimeoutMS),
		AsyncActionTimeout:            durationMS(raw.AsyncActionTimeoutMS),
		RecentMemoryLimit:             raw.RecentMemoryLimit,
		MemoryContextSizeLimit:        raw.MemoryContextSizeLimit,
		MaxSteps:                      raw.MaxSteps,
		MaxToolCallsPerStep:           raw.MaxToolCallsPerStep,
		MaxToolCallsPerTurn:           raw.MaxToolCallsPerTurn,
		MaxAsyncActionsPerTurn:        raw.MaxAsyncActionsPerTurn,
		MaxParallelToolCalls:          raw.MaxParallelToolCalls,
		MaxRequestBytes:               raw.MaxRequestBytes,
		MaxSystemBytes:                raw.MaxSystemBytes,
		MaxUserMessageBytes:           raw.MaxUserMessageBytes,
		MaxDefinitionBytes:            raw.MaxDefinitionBytes,
		MaxObservationBytes:           raw.MaxObservationBytes,
		MaxEventBytes:                 raw.MaxEventBytes,
		MaxContextFactsBytes:          raw.MaxContextFactsBytes,
		MaxRecentMemoryBytes:          raw.MaxRecentMemoryBytes,
		MaxTranscriptBytes:            raw.MaxTranscriptBytes,
		MaxToolCount:                  raw.MaxToolCount,
		MaxToolDescriptionBytes:       raw.MaxToolDescriptionBytes,
		MaxToolSchemaBytes:            raw.MaxToolSchemaBytes,
		MaxTotalToolSchemaBytes:       raw.MaxTotalToolSchemaBytes,
		MaxToolResultOutputBytes:      raw.MaxToolResultOutputBytes,
		MaxToolResultOutputDepth:      raw.MaxToolResultOutputDepth,
		MaxToolResultOutputFields:     raw.MaxToolResultOutputFields,
		MaxToolResultOutputArrayItems: raw.MaxToolResultOutputArrayItems,
		DefinitionCatalogRoot:         raw.DefinitionCatalogRoot,
		Prompt:                        raw.Prompt,
	}.WithDefaults()

	return cfg, nil
}

// MemoryEnabledValue 返回最终生效的 MemoryEnabled。
// 使用指针是为了区分“配置里没写”和“显式写 false”。
func (c Config) MemoryEnabledValue() bool {
	if c.MemoryEnabled == nil {
		return defaultMemoryEnabled
	}
	return *c.MemoryEnabled
}

// WithDefaults 为 PromptConfig 补齐缺省字段。
// 这样配置文件可以只覆盖关心的 prompt 片段。
func (p PromptConfig) WithDefaults() PromptConfig {
	defaults := DefaultConfig().Prompt

	if p.Language == "" {
		p.Language = defaults.Language
	}
	if p.NPCStyle == "" {
		p.NPCStyle = defaults.NPCStyle
	}
	if p.MaxSpeakChars <= 0 {
		p.MaxSpeakChars = defaults.MaxSpeakChars
	}
	if p.ToolInstruction == "" {
		p.ToolInstruction = defaults.ToolInstruction
	}

	return p
}

// WithDefaults 为 Agent Config 补齐缺省字段。
// Memory 配置和 Runtime 预算在这里统一归一化，避免 Loop 初始化时处理零值分支。
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.MemoryEnabled == nil {
		c.MemoryEnabled = boolPtr(defaults.MemoryEnabledValue())
	}

	if c.TurnTimeout <= 0 {
		c.TurnTimeout = defaults.TurnTimeout
	}
	if c.LLMTimeout <= 0 {
		c.LLMTimeout = defaults.LLMTimeout
	}
	if c.ObserveTimeout <= 0 {
		c.ObserveTimeout = defaults.ObserveTimeout
	}
	if c.ActionTimeout <= 0 {
		c.ActionTimeout = defaults.ActionTimeout
	}
	if c.ActionStartTimeout <= 0 {
		c.ActionStartTimeout = defaults.ActionStartTimeout
	}
	if c.AsyncActionTimeout <= 0 {
		c.AsyncActionTimeout = defaults.AsyncActionTimeout
	}
	if c.RecentMemoryLimit <= 0 {
		c.RecentMemoryLimit = defaults.RecentMemoryLimit
	}
	if c.MemoryContextSizeLimit <= 0 {
		c.MemoryContextSizeLimit = defaults.MemoryContextSizeLimit
	}
	if c.MaxSteps <= 0 {
		c.MaxSteps = defaults.MaxSteps
	}
	if c.MaxToolCallsPerStep <= 0 {
		c.MaxToolCallsPerStep = defaults.MaxToolCallsPerStep
	}
	if c.MaxToolCallsPerTurn <= 0 {
		c.MaxToolCallsPerTurn = defaults.MaxToolCallsPerTurn
	}
	if c.MaxAsyncActionsPerTurn <= 0 {
		c.MaxAsyncActionsPerTurn = defaults.MaxAsyncActionsPerTurn
	}
	if c.MaxParallelToolCalls <= 0 {
		c.MaxParallelToolCalls = defaults.MaxParallelToolCalls
	}
	if c.MaxRecentMemoryBytes <= 0 && c.MemoryContextSizeLimit > 0 {
		c.MaxRecentMemoryBytes = c.MemoryContextSizeLimit
	}
	if c.MaxRequestBytes <= 0 {
		c.MaxRequestBytes = defaults.MaxRequestBytes
	}
	if c.MaxSystemBytes <= 0 {
		c.MaxSystemBytes = defaults.MaxSystemBytes
	}
	if c.MaxUserMessageBytes <= 0 {
		c.MaxUserMessageBytes = defaults.MaxUserMessageBytes
	}
	if c.MaxDefinitionBytes <= 0 {
		c.MaxDefinitionBytes = defaults.MaxDefinitionBytes
	}
	if c.MaxObservationBytes <= 0 {
		c.MaxObservationBytes = defaults.MaxObservationBytes
	}
	if c.MaxEventBytes <= 0 {
		c.MaxEventBytes = defaults.MaxEventBytes
	}
	if c.MaxContextFactsBytes <= 0 {
		c.MaxContextFactsBytes = defaults.MaxContextFactsBytes
	}
	if c.MaxRecentMemoryBytes <= 0 {
		c.MaxRecentMemoryBytes = defaults.MaxRecentMemoryBytes
	}
	if c.MemoryContextSizeLimit <= 0 {
		c.MemoryContextSizeLimit = c.MaxRecentMemoryBytes
	}
	if c.MaxTranscriptBytes <= 0 {
		c.MaxTranscriptBytes = defaults.MaxTranscriptBytes
	}
	if c.MaxToolCount <= 0 {
		c.MaxToolCount = defaults.MaxToolCount
	}
	if c.MaxToolDescriptionBytes <= 0 {
		c.MaxToolDescriptionBytes = defaults.MaxToolDescriptionBytes
	}
	if c.MaxToolSchemaBytes <= 0 {
		c.MaxToolSchemaBytes = defaults.MaxToolSchemaBytes
	}
	if c.MaxTotalToolSchemaBytes <= 0 {
		c.MaxTotalToolSchemaBytes = defaults.MaxTotalToolSchemaBytes
	}
	if c.MaxToolResultOutputBytes <= 0 {
		c.MaxToolResultOutputBytes = defaults.MaxToolResultOutputBytes
	}
	if c.MaxToolResultOutputDepth <= 0 {
		c.MaxToolResultOutputDepth = defaults.MaxToolResultOutputDepth
	}
	if c.MaxToolResultOutputFields <= 0 {
		c.MaxToolResultOutputFields = defaults.MaxToolResultOutputFields
	}
	if c.MaxToolResultOutputArrayItems <= 0 {
		c.MaxToolResultOutputArrayItems = defaults.MaxToolResultOutputArrayItems
	}
	c.Prompt = c.Prompt.WithDefaults()
	return c
}

// durationMS 将配置文件中的毫秒值转换成 time.Duration。
func durationMS(v int64) time.Duration {
	if v <= 0 {
		return 0
	}
	return time.Duration(v) * time.Millisecond
}

// boolPtr 帮助构造可区分“未配置/显式配置”的 bool 指针。
func boolPtr(v bool) *bool {
	return &v
}
