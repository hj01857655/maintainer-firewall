package service

import "strings"

type SuggestedAction struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Reason  string `json:"reason"`
	Matched string `json:"matched"`
}

type RuleEngine struct {
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{}
}

func (e *RuleEngine) Evaluate(eventType string, payload map[string]any) []SuggestedAction {
	if eventType != "issues" && eventType != "pull_request" {
		return nil
	}

	text := strings.ToLower(extractText(payload))
	if strings.TrimSpace(text) == "" {
		return nil
	}

	rules := []struct {
		keyword string
		label   string
		reply   string
		reason  string
	}{
		{
			keyword: "duplicate",
			label:   "needs-triage",
			reply:   "Thanks! This may be a duplicate. Please reference related issue links.",
			reason:  "contains duplicate keyword",
		},
		{
			keyword: "help wanted",
			label:   "help-wanted",
			reply:   "Maintainers marked this as help wanted candidate.",
			reason:  "contains help wanted keyword",
		},
		{
			keyword: "urgent",
			label:   "priority-high",
			reply:   "Marked for fast triage due to urgency signal.",
			reason:  "contains urgent keyword",
		},
	}

	result := make([]SuggestedAction, 0, 4)
	for _, rule := range rules {
		if strings.Contains(text, rule.keyword) {
			result = append(result,
				SuggestedAction{Type: "label", Value: rule.label, Reason: rule.reason, Matched: rule.keyword},
				SuggestedAction{Type: "comment", Value: rule.reply, Reason: rule.reason, Matched: rule.keyword},
			)
		}
	}

	return dedupeActions(result)
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
		key := a.Type + "|" + a.Value
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, a)
	}
	return out
}
