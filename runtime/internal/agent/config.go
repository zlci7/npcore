package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	agentcontext "gameagent/runtime/internal/context"
	"log"
	"os"
	"strings"
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
	MaxSteps                      int
	MaxToolCallsPerStep           int
	MaxToolCallsPerTurn           int
	MaxAsyncActionsPerTurn        int
	MaxParallelToolCalls          int
	MaxRequestTokens              int
	MaxSystemTokens               int
	MaxUserMessageTokens          int
	MaxDefinitionTokens           int
	MaxObservationTokens          int
	MaxEventTokens                int
	MaxContextFactsTokens         int
	MaxRecentMemoryTokens         int
	MaxTranscriptTokens           int
	MaxToolCount                  int
	MaxToolDescriptionTokens      int
	MaxToolSchemaTokens           int
	MaxTotalToolSchemaTokens      int
	MaxToolResultOutputTokens     int
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
	MaxRequestTokens              int          `json:"max_request_tokens"`
	MaxSystemTokens               int          `json:"max_system_tokens"`
	MaxUserMessageTokens          int          `json:"max_user_message_tokens"`
	MaxDefinitionTokens           int          `json:"max_definition_tokens"`
	MaxObservationTokens          int          `json:"max_observation_tokens"`
	MaxEventTokens                int          `json:"max_event_tokens"`
	MaxContextFactsTokens         int          `json:"max_context_facts_tokens"`
	MaxRecentMemoryTokens         int          `json:"max_recent_memory_tokens"`
	MaxTranscriptTokens           int          `json:"max_transcript_tokens"`
	MaxToolCount                  int          `json:"max_tool_count"`
	MaxToolDescriptionTokens      int          `json:"max_tool_description_tokens"`
	MaxToolSchemaTokens           int          `json:"max_tool_schema_tokens"`
	MaxTotalToolSchemaTokens      int          `json:"max_total_tool_schema_tokens"`
	MaxToolResultOutputTokens     int          `json:"max_tool_result_output_tokens"`
	MaxRequestBytes               int          `json:"max_request_bytes"`
	MaxSystemBytes                int          `json:"max_system_bytes"`
	MaxUserMessageBytes           int          `json:"max_user_message_bytes"`
	MaxDefinitionBytes            int          `json:"max_definition_bytes"`
	MaxObservationBytes           int          `json:"max_observation_bytes"`
	MaxEventBytes                 int          `json:"max_event_bytes"`
	MaxContextFactsBytes          int          `json:"max_context_facts_bytes"`
	MaxRecentMemoryBytes          int          `json:"max_recent_memory_bytes"`
	MaxTranscriptBytes            int          `json:"max_transcript_bytes"`
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
		MaxSteps:                      3,
		MaxToolCallsPerStep:           4,
		MaxToolCallsPerTurn:           6,
		MaxAsyncActionsPerTurn:        1,
		MaxParallelToolCalls:          4,
		MaxRequestTokens:              budget.MaxRequestTokens,
		MaxSystemTokens:               budget.MaxSystemTokens,
		MaxUserMessageTokens:          budget.MaxUserMessageTokens,
		MaxDefinitionTokens:           budget.MaxDefinitionTokens,
		MaxObservationTokens:          budget.MaxObservationTokens,
		MaxEventTokens:                budget.MaxEventTokens,
		MaxContextFactsTokens:         budget.MaxContextFactsTokens,
		MaxRecentMemoryTokens:         budget.MaxRecentMemoryTokens,
		MaxTranscriptTokens:           budget.MaxTranscriptTokens,
		MaxToolCount:                  budget.MaxToolCount,
		MaxToolDescriptionTokens:      budget.MaxToolDescriptionTokens,
		MaxToolSchemaTokens:           budget.MaxToolSchemaTokens,
		MaxTotalToolSchemaTokens:      budget.MaxTotalToolSchemaTokens,
		MaxToolResultOutputTokens:     budget.MaxToolResultOutputTokens,
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

	legacyBudgetFields := raw.legacyBudgetFields()
	if len(legacyBudgetFields) > 0 {
		log.Printf("agent config uses deprecated byte budget fields; migrate to *_tokens: %s", strings.Join(legacyBudgetFields, ", "))
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
		MaxSteps:                      raw.MaxSteps,
		MaxToolCallsPerStep:           raw.MaxToolCallsPerStep,
		MaxToolCallsPerTurn:           raw.MaxToolCallsPerTurn,
		MaxAsyncActionsPerTurn:        raw.MaxAsyncActionsPerTurn,
		MaxParallelToolCalls:          raw.MaxParallelToolCalls,
		MaxRequestTokens:              budgetTokenValue(raw.MaxRequestTokens, raw.MaxRequestBytes),
		MaxSystemTokens:               budgetTokenValue(raw.MaxSystemTokens, raw.MaxSystemBytes),
		MaxUserMessageTokens:          budgetTokenValue(raw.MaxUserMessageTokens, raw.MaxUserMessageBytes),
		MaxDefinitionTokens:           budgetTokenValue(raw.MaxDefinitionTokens, raw.MaxDefinitionBytes),
		MaxObservationTokens:          budgetTokenValue(raw.MaxObservationTokens, raw.MaxObservationBytes),
		MaxEventTokens:                budgetTokenValue(raw.MaxEventTokens, raw.MaxEventBytes),
		MaxContextFactsTokens:         budgetTokenValue(raw.MaxContextFactsTokens, raw.MaxContextFactsBytes),
		MaxRecentMemoryTokens:         recentMemoryTokenValue(raw),
		MaxTranscriptTokens:           budgetTokenValue(raw.MaxTranscriptTokens, raw.MaxTranscriptBytes),
		MaxToolCount:                  raw.MaxToolCount,
		MaxToolDescriptionTokens:      budgetTokenValue(raw.MaxToolDescriptionTokens, raw.MaxToolDescriptionBytes),
		MaxToolSchemaTokens:           budgetTokenValue(raw.MaxToolSchemaTokens, raw.MaxToolSchemaBytes),
		MaxTotalToolSchemaTokens:      budgetTokenValue(raw.MaxTotalToolSchemaTokens, raw.MaxTotalToolSchemaBytes),
		MaxToolResultOutputTokens:     budgetTokenValue(raw.MaxToolResultOutputTokens, raw.MaxToolResultOutputBytes),
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
	if c.MaxRequestTokens <= 0 {
		c.MaxRequestTokens = defaults.MaxRequestTokens
	}
	if c.MaxSystemTokens <= 0 {
		c.MaxSystemTokens = defaults.MaxSystemTokens
	}
	if c.MaxUserMessageTokens <= 0 {
		c.MaxUserMessageTokens = defaults.MaxUserMessageTokens
	}
	if c.MaxDefinitionTokens <= 0 {
		c.MaxDefinitionTokens = defaults.MaxDefinitionTokens
	}
	if c.MaxObservationTokens <= 0 {
		c.MaxObservationTokens = defaults.MaxObservationTokens
	}
	if c.MaxEventTokens <= 0 {
		c.MaxEventTokens = defaults.MaxEventTokens
	}
	if c.MaxContextFactsTokens <= 0 {
		c.MaxContextFactsTokens = defaults.MaxContextFactsTokens
	}
	if c.MaxRecentMemoryTokens <= 0 {
		c.MaxRecentMemoryTokens = defaults.MaxRecentMemoryTokens
	}
	if c.MaxTranscriptTokens <= 0 {
		c.MaxTranscriptTokens = defaults.MaxTranscriptTokens
	}
	if c.MaxToolCount <= 0 {
		c.MaxToolCount = defaults.MaxToolCount
	}
	if c.MaxToolDescriptionTokens <= 0 {
		c.MaxToolDescriptionTokens = defaults.MaxToolDescriptionTokens
	}
	if c.MaxToolSchemaTokens <= 0 {
		c.MaxToolSchemaTokens = defaults.MaxToolSchemaTokens
	}
	if c.MaxTotalToolSchemaTokens <= 0 {
		c.MaxTotalToolSchemaTokens = defaults.MaxTotalToolSchemaTokens
	}
	if c.MaxToolResultOutputTokens <= 0 {
		c.MaxToolResultOutputTokens = defaults.MaxToolResultOutputTokens
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

func budgetTokenValue(tokenValue int, legacyBytes int) int {
	if tokenValue > 0 {
		return tokenValue
	}
	if legacyBytes > 0 {
		return legacyBytesToEstimatedTokens(legacyBytes)
	}
	return 0
}

func recentMemoryTokenValue(raw fileConfig) int {
	if raw.MaxRecentMemoryTokens > 0 {
		return raw.MaxRecentMemoryTokens
	}
	if raw.MaxRecentMemoryBytes > 0 {
		return legacyBytesToEstimatedTokens(raw.MaxRecentMemoryBytes)
	}
	if raw.MemoryContextSizeLimit > 0 {
		return legacyBytesToEstimatedTokens(raw.MemoryContextSizeLimit)
	}
	return 0
}

func legacyBytesToEstimatedTokens(bytes int) int {
	if bytes <= 0 {
		return 0
	}
	return (bytes + 3) / 4
}

func (c fileConfig) legacyBudgetFields() []string {
	fields := make([]string, 0)
	add := func(name string, value int) {
		if value > 0 {
			fields = append(fields, name)
		}
	}
	add("memory_context_size_limit", c.MemoryContextSizeLimit)
	add("max_request_bytes", c.MaxRequestBytes)
	add("max_system_bytes", c.MaxSystemBytes)
	add("max_user_message_bytes", c.MaxUserMessageBytes)
	add("max_definition_bytes", c.MaxDefinitionBytes)
	add("max_observation_bytes", c.MaxObservationBytes)
	add("max_event_bytes", c.MaxEventBytes)
	add("max_context_facts_bytes", c.MaxContextFactsBytes)
	add("max_recent_memory_bytes", c.MaxRecentMemoryBytes)
	add("max_transcript_bytes", c.MaxTranscriptBytes)
	add("max_tool_description_bytes", c.MaxToolDescriptionBytes)
	add("max_tool_schema_bytes", c.MaxToolSchemaBytes)
	add("max_total_tool_schema_bytes", c.MaxTotalToolSchemaBytes)
	add("max_tool_result_output_bytes", c.MaxToolResultOutputBytes)
	return fields
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
