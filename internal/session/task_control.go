package session

import (
	"strings"

	"rm_ai_agent/internal/planner"
)

type taskController struct {
	userText        string
	taskMode        string
	phase           string
	intentSummary   string
	doneWhen        string
	maxRounds       int
	maxPageScans    int
	listScanRounds  int
	artifactRounds  int
	seenSignatures  map[string]int
	preferPageScan  bool
	allowDetailDive bool
}

func newTaskController(userText string, plan planner.Plan) *taskController {
	mode := strings.TrimSpace(plan.TaskMode)
	phase := strings.TrimSpace(plan.Phase)
	preferPageScan := wantsBroaderExploration(userText) || mode == "exhaustive"
	controller := &taskController{
		userText:        userText,
		taskMode:        mode,
		phase:           phase,
		intentSummary:   strings.TrimSpace(plan.IntentSummary),
		doneWhen:        strings.TrimSpace(plan.DoneWhen),
		maxRounds:       maxRoundsForTaskMode(mode),
		maxPageScans:    maxPageScansForMode(mode, userText),
		seenSignatures:  make(map[string]int),
		preferPageScan:  preferPageScan,
		allowDetailDive: !preferPageScan,
	}
	if controller.phase == "" {
		controller.phase = inferredControllerPhase(mode, plan.ToolCalls, preferPageScan)
	}
	return controller
}

func (c *taskController) CountSignature(signature string) int {
	c.seenSignatures[signature]++
	return c.seenSignatures[signature]
}

func (c *taskController) ObserveRound(calls []planner.ToolCall) {
	if allThreatListCalls(calls) {
		c.listScanRounds++
		if c.preferPageScan && c.listScanRounds >= c.maxPageScans {
			c.allowDetailDive = true
		}
		if c.phase == "" || c.phase == "overview" {
			c.phase = "scan_pages"
		}
		return
	}
	if allArtifactExplorationCalls(calls) {
		c.artifactRounds++
		c.phase = "drill_down"
		return
	}
	if containsDetailCalls(calls) {
		c.allowDetailDive = true
		c.phase = "drill_down"
	}
}

func (c *taskController) AdoptPlan(plan planner.Plan) {
	if strings.TrimSpace(plan.TaskMode) != "" {
		c.taskMode = strings.TrimSpace(plan.TaskMode)
		c.maxRounds = maxRoundsForTaskMode(c.taskMode)
		if c.maxPageScans == 0 {
			c.maxPageScans = maxPageScansForMode(c.taskMode, c.userText)
		}
	}
	if strings.TrimSpace(plan.Phase) != "" {
		c.phase = strings.TrimSpace(plan.Phase)
	}
	if strings.TrimSpace(plan.IntentSummary) != "" {
		c.intentSummary = strings.TrimSpace(plan.IntentSummary)
	}
	if strings.TrimSpace(plan.DoneWhen) != "" {
		c.doneWhen = strings.TrimSpace(plan.DoneWhen)
	}
	if c.preferPageScan && (c.phase == "pick_target" || c.phase == "drill_down") && c.listScanRounds < c.maxPageScans {
		c.allowDetailDive = false
	}
}

func (c *taskController) ShouldStopByBudget(rounds int) bool {
	return rounds > c.maxRounds
}

func (c *taskController) ShouldStopArtifact(currentCalls []planner.ToolCall, nextCalls []planner.ToolCall) bool {
	if c.taskMode == "exhaustive" || c.artifactRounds < 3 || len(nextCalls) == 0 {
		return false
	}
	if !allArtifactExplorationCalls(currentCalls) || !allArtifactExplorationCalls(nextCalls) {
		return false
	}
	return samePrimaryArtifact(currentCalls, nextCalls)
}

func (c *taskController) AdjustNextCalls(currentCalls []planner.ToolCall, nextPlan planner.Plan) ([]planner.ToolCall, string) {
	if !c.preferPageScan {
		return nextPlan.ToolCalls, ""
	}
	if !allThreatListCalls(currentCalls) {
		return nextPlan.ToolCalls, ""
	}
	if c.allowDetailDive {
		return nextPlan.ToolCalls, ""
	}
	if c.listScanRounds >= c.maxPageScans {
		c.allowDetailDive = true
		return nextPlan.ToolCalls, ""
	}
	if len(nextPlan.ToolCalls) == 0 || containsDetailCalls(nextPlan.ToolCalls) {
		c.phase = "scan_pages"
		return synthesizeNextPageCalls(currentCalls), "我先继续按你的目标多看几页，把候选范围扫开，再决定要不要深入某一条。"
	}
	if allThreatListCalls(nextPlan.ToolCalls) {
		c.phase = "scan_pages"
		return ensureProgressivePagination(currentCalls, nextPlan.ToolCalls), ""
	}
	return nextPlan.ToolCalls, ""
}

func inferredControllerPhase(taskMode string, calls []planner.ToolCall, preferPageScan bool) string {
	if preferPageScan && allThreatListCalls(calls) {
		return "scan_pages"
	}
	if containsDetailCalls(calls) || allArtifactExplorationCalls(calls) {
		return "drill_down"
	}
	if allThreatListCalls(calls) {
		return "overview"
	}
	if strings.TrimSpace(taskMode) == "action" {
		return "answer"
	}
	return "overview"
}

func maxPageScansForMode(taskMode string, userText string) int {
	if wantsBroaderExploration(userText) {
		return 3
	}
	switch strings.TrimSpace(taskMode) {
	case "exhaustive":
		return 5
	case "overview_drilldown":
		return 2
	default:
		return 1
	}
}

func wantsBroaderExploration(text string) bool {
	plain := strings.ToLower(strings.TrimSpace(text))
	keywords := []string{"多看几页", "再看几页", "多翻几页", "继续往后", "往后翻", "继续看更多", "更多", "多看看", "其他", "别的", "不同类型", "看看其他", "多找几个"}
	for _, keyword := range keywords {
		if strings.Contains(plain, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func containsDetailCalls(calls []planner.ToolCall) bool {
	for _, call := range calls {
		switch call.Name {
		case "edr_incident_view", "edr_detection_view", "artifact_outline", "artifact_search", "artifact_read":
			return true
		}
	}
	return false
}

func synthesizeNextPageCalls(currentCalls []planner.ToolCall) []planner.ToolCall {
	next := make([]planner.ToolCall, 0, len(currentCalls))
	for _, call := range currentCalls {
		updated := call
		updated.Page = positiveOr(call.Page, 1) + 1
		next = append(next, updated)
	}
	return next
}

func ensureProgressivePagination(currentCalls []planner.ToolCall, nextCalls []planner.ToolCall) []planner.ToolCall {
	updated := make([]planner.ToolCall, 0, len(nextCalls))
	for _, next := range nextCalls {
		bestPage := positiveOr(next.Page, 1)
		for _, current := range currentCalls {
			if sameThreatListQuery(next, current) {
				if bestPage <= positiveOr(current.Page, 1) {
					bestPage = positiveOr(current.Page, 1) + 1
				}
				break
			}
		}
		next.Page = bestPage
		updated = append(updated, next)
	}
	return updated
}
