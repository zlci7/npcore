package context

import (
	"fmt"

	"gameagent/runtime/internal/model"
	"gameagent/runtime/internal/tool"
)

type requestMessageForSizing struct {
	Role        model.Role         `json:"role,omitempty"`
	Content     string             `json:"content,omitempty"`
	ToolCalls   []model.ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []model.ToolResult `json:"tool_results,omitempty"`
}

func MeasureRequest(req model.Request) RequestSizeSummary {
	summary := RequestSizeSummary{
		SystemBytes:   len([]byte(req.System)),
		ToolsBytes:    sectionProxyBytes(req.Tools),
		ControlsBytes: sectionProxyBytes(req.Controls),
	}
	summary.MessagesBytes, summary.UserMessageBytes = measureMessages(req.Messages)
	summary.TotalBytes = summary.SystemBytes + summary.MessagesBytes + summary.ToolsBytes + summary.ControlsBytes
	return summary
}

func measureMessages(messages []model.Message) (int, int) {
	if len(messages) == 0 {
		return 0, 0
	}
	items := make([]requestMessageForSizing, 0, len(messages))
	userBytes := 0
	for _, message := range messages {
		items = append(items, requestMessageForSizing{
			Role:        message.Role,
			Content:     message.Content,
			ToolCalls:   message.ToolCalls,
			ToolResults: message.ToolResults,
		})
		if message.Role == model.RoleUser {
			userBytes += len([]byte(message.Content))
		}
	}
	return sectionProxyBytes(items), userBytes
}

func RequestSizeExceedsBudget(summary RequestSizeSummary, budget BudgetConfig) bool {
	if budget.MaxSystemBytes > 0 && summary.SystemBytes > budget.MaxSystemBytes {
		return true
	}
	if budget.MaxUserMessageBytes > 0 && summary.UserMessageBytes > budget.MaxUserMessageBytes {
		return true
	}
	if budget.MaxRequestBytes > 0 && summary.TotalBytes > budget.MaxRequestBytes {
		return true
	}
	return false
}

func RequiredRequestSectionExceedsBudget(summary RequestSizeSummary, budget BudgetConfig) bool {
	return budget.MaxSystemBytes > 0 && summary.SystemBytes > budget.MaxSystemBytes
}

func RequestSizeBudgetError(summary RequestSizeSummary, budget BudgetConfig) error {
	return fmt.Errorf(
		"%w: final model request exceeds byte budget (total=%d max_request=%d system=%d max_system=%d user=%d max_user=%d)",
		ErrBudgetExceeded,
		summary.TotalBytes,
		budget.MaxRequestBytes,
		summary.SystemBytes,
		budget.MaxSystemBytes,
		summary.UserMessageBytes,
		budget.MaxUserMessageBytes,
	)
}

func ToolAdmissionSummaryFromReport(report tool.ToolAdmissionReport) ToolAdmissionSummary {
	return ToolAdmissionSummary{
		AcceptedToolCount: report.AcceptedToolCount,
		AcceptedToolNames: append([]string(nil), report.AcceptedToolNames...),
		DroppedToolCount:  report.DroppedToolCount,
		DroppedToolNames:  append([]string(nil), report.DroppedToolNames...),
		DroppedTools:      append([]tool.ToolAdmissionDrop(nil), report.DroppedTools...),
		TotalSchemaBytes:  report.TotalSchemaBytes,
	}
}

func (r ContextBuildReport) WithToolAdmission(summary ToolAdmissionSummary) ContextBuildReport {
	r.ToolAdmission = summary
	return r
}

func (r ContextBuildReport) WithFinalRequestSize(summary RequestSizeSummary) ContextBuildReport {
	r.FinalRequestSize = summary
	return r
}

func (r ContextBuildReport) WithReason(reason string) ContextBuildReport {
	r.addReason(reason)
	return r
}
