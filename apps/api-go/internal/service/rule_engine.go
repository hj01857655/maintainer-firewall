package service

import "strings"

type SuggestedAction struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Reason  string `json:"reason"`
	Matched string `json:"matched"`
}

type RuleDefinition struct {
	EventType       string
	Keyword         string
	SuggestionType  string
	SuggestionValue string
	Reason          string
}

type RuleEngine struct {
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{}
}

func (e *RuleEngine) Evaluate(eventType string, payload map[string]any) []SuggestedAction {
	return e.EvaluateWithRules(eventType, payload, defaultRules())
}

func (e *RuleEngine) EvaluateWithRules(eventType string, payload map[string]any, rules []RuleDefinition) []SuggestedAction {
	if eventType != "issues" && eventType != "pull_request" {
		return nil
	}

	text := strings.ToLower(extractText(payload))
	if strings.TrimSpace(text) == "" {
		return nil
	}

	result := make([]SuggestedAction, 0, 4)
	for _, rule := range rules {
		if strings.TrimSpace(rule.Keyword) == "" {
			continue
		}
		if rule.EventType != "" && rule.EventType != eventType {
			continue
		}
		keyword := strings.ToLower(rule.Keyword)
		if strings.Contains(text, keyword) {
			result = append(result, SuggestedAction{
				Type:    rule.SuggestionType,
				Value:   rule.SuggestionValue,
				Reason:  rule.Reason,
				Matched: rule.Keyword,
			})
		}
	}

	return dedupeActions(result)
}

func defaultRules() []RuleDefinition {
	return []RuleDefinition{
		{EventType: "issues", Keyword: "duplicate", SuggestionType: "label", SuggestionValue: "needs-triage", Reason: "contains duplicate keyword"},
		{EventType: "issues", Keyword: "duplicate", SuggestionType: "comment", SuggestionValue: "Thanks! This may be a duplicate. Please reference related issue links.", Reason: "contains duplicate keyword"},
		{EventType: "issues", Keyword: "help wanted", SuggestionType: "label", SuggestionValue: "help-wanted", Reason: "contains help wanted keyword"},
		{EventType: "issues", Keyword: "help wanted", SuggestionType: "comment", SuggestionValue: "Maintainers marked this as help wanted candidate.", Reason: "contains help wanted keyword"},
		{EventType: "issues", Keyword: "urgent", SuggestionType: "label", SuggestionValue: "priority-high", Reason: "contains urgent keyword"},
		{EventType: "issues", Keyword: "urgent", SuggestionType: "comment", SuggestionValue: "Marked for fast triage due to urgency signal.", Reason: "contains urgent keyword"},
		{EventType: "pull_request", Keyword: "duplicate", SuggestionType: "label", SuggestionValue: "needs-triage", Reason: "contains duplicate keyword"},
		{EventType: "pull_request", Keyword: "duplicate", SuggestionType: "comment", SuggestionValue: "Thanks! This may be a duplicate. Please reference related PR links.", Reason: "contains duplicate keyword"},
		{EventType: "pull_request", Keyword: "help wanted", SuggestionType: "label", SuggestionValue: "help-wanted", Reason: "contains help wanted keyword"},
		{EventType: "pull_request", Keyword: "help wanted", SuggestionType: "comment", SuggestionValue: "Maintainers marked this as help wanted candidate.", Reason: "contains help wanted keyword"},
		{EventType: "pull_request", Keyword: "urgent", SuggestionType: "label", SuggestionValue: "priority-high", Reason: "contains urgent keyword"},
		{EventType: "pull_request", Keyword: "urgent", SuggestionType: "comment", SuggestionValue: "Marked for fast triage due to urgency signal.", Reason: "contains urgent keyword"},
	}
}

func extractText(payload map[string]any) string {
	parts := []string{}

	if issue, ok := payload["issue"].(map[string]any); ok {
		if t, _ := issue["title"].(string); t != "" {
			parts = append(parts, t)
		}
		if b, _ := issue["body"].(string); b != "" {
			parts = append(parts, b)
		}
	}

	if pr, ok := payload["pull_request"].(map[string]any); ok {
		if t, _ := pr["title"].(string); t != "" {
			parts = append(parts, t)
		}
		if b, _ := pr["body"].(string); b != "" {
			parts = append(parts, b)
		}
	}

	return strings.Join(parts, "\n")
}

func dedupeActions(in []SuggestedAction) []SuggestedAction {
	if len(in) == 0 {
		return in
	}
	seen := map[string]struct{}{}
	out := make([]SuggestedAction, 0, len(in))
	for _, a := range in {
		key := a.Type + "|" + a.Value + "|" + a.Matched
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, a)
	}
	return out
}
