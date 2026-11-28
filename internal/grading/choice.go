package grading

import (
	"encoding/json"
	"sort"
	"strings"
)

// checkChoice сравнивает выбранные варианты с правильным набором как множества.
// Порядок вариантов не важен, а правильный набор не раскрывается в результате.
func checkChoice(parsed spec, answer string) Result {
	actual, ok := parseOptionIDs(answer)
	if !ok {
		return mismatch(nil, strings.TrimSpace(answer))
	}

	expected := make([]string, 0, len(parsed.OptionIDs))
	seen := make(map[string]struct{}, len(parsed.OptionIDs))
	for _, id := range parsed.OptionIDs {
		id = strings.TrimSpace(id)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		expected = append(expected, id)
	}
	sort.Strings(expected)

	normalized := make([]string, 0, len(actual))
	for _, id := range actual {
		if _, ok := seen[id]; !ok {
			return mismatch(nil, strings.Join(actual, ","))
		}
		normalized = append(normalized, id)
	}
	sort.Strings(normalized)

	if len(normalized) != len(expected) {
		return mismatch(nil, strings.Join(normalized, ","))
	}
	for index := range expected {
		if normalized[index] != expected[index] {
			return mismatch(nil, strings.Join(normalized, ","))
		}
	}

	return matched(nil, strings.Join(normalized, ","))
}

func parseOptionIDs(answer string) ([]string, bool) {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil, false
	}

	if strings.HasPrefix(answer, "[") {
		var values []string
		if err := json.Unmarshal([]byte(answer), &values); err != nil {
			return nil, false
		}

		return cleanOptionIDs(values), true
	}

	parts := strings.FieldsFunc(answer, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})

	return cleanOptionIDs(parts), true
}

func cleanOptionIDs(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}

	return result
}
