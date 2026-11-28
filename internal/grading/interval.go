package grading

import (
	"math"
	"sort"
	"strings"
)

type intervalValue struct {
	lower      float64
	upper      float64
	hasBracket bool
	lowerInc   bool
	upperInc   bool
}

func checkInterval(parsed spec, answer string) Result {
	if actual, ok := parseSet(answer); ok {
		for _, item := range parsed.Accepted {
			expected, ok := parseSet(item)
			if !ok {
				continue
			}
			if setsEqual(actual, expected) {
				return matched(parsed.Accepted, formatSet(actual))
			}
		}

		return mismatch(parsed.Accepted, formatSet(actual))
	}

	actual, ok := parseInterval(answer)
	if !ok {
		normalized := normalizeIntervalString(answer)
		for _, item := range parsed.Accepted {
			if normalized != "" && normalized == normalizeIntervalString(item) {
				return matched(parsed.Accepted, normalized)
			}
		}

		return mismatch(parsed.Accepted, normalized)
	}

	for _, item := range parsed.Accepted {
		expected, ok := parseInterval(item)
		if !ok {
			continue
		}
		if intervalsEqual(actual, expected) {
			return matched(parsed.Accepted, formatInterval(actual.lower, actual.upper))
		}
	}

	return mismatch(parsed.Accepted, formatInterval(actual.lower, actual.upper))
}

func parseInterval(value string) (intervalValue, bool) {
	value = strings.ReplaceAll(normalizeBase(value), " ", "")
	if value == "" {
		return intervalValue{}, false
	}

	var result intervalValue
	if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "(") {
		result.hasBracket = true
		result.lowerInc = value[0] == '['
		value = value[1:]
	}
	if strings.HasSuffix(value, "]") || strings.HasSuffix(value, ")") {
		result.hasBracket = true
		result.upperInc = value[len(value)-1] == ']'
		value = value[:len(value)-1]
	} else {
		result.upperInc = true
	}
	if !result.hasBracket {
		result.lowerInc = true
		result.upperInc = true
	}

	separator := ";"
	if !strings.Contains(value, ";") {
		separator = ","
	}
	parts := strings.Split(value, separator)
	if len(parts) != 2 {
		return intervalValue{}, false
	}

	lower, ok := parseNumber(parts[0], nil)
	if !ok {
		return intervalValue{}, false
	}
	upper, ok := parseNumber(parts[1], nil)
	if !ok {
		return intervalValue{}, false
	}

	result.lower = lower
	result.upper = upper
	if result.lower > result.upper {
		result.lower, result.upper = result.upper, result.lower
		result.lowerInc, result.upperInc = result.upperInc, result.lowerInc
	}

	return result, true
}

func intervalsEqual(actual, expected intervalValue) bool {
	if math.Abs(actual.lower-expected.lower) > numberEpsilon ||
		math.Abs(actual.upper-expected.upper) > numberEpsilon {
		return false
	}
	if actual.hasBracket && expected.hasBracket {
		return actual.lowerInc == expected.lowerInc && actual.upperInc == expected.upperInc
	}

	return true
}

func parseSet(value string) ([]float64, bool) {
	value = strings.ReplaceAll(normalizeBase(value), " ", "")
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return nil, false
	}
	value = strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}")
	if value == "" {
		return nil, false
	}

	separator := ";"
	if !strings.Contains(value, ";") {
		separator = ","
	}
	parts := strings.Split(value, separator)
	items := make([]float64, 0, len(parts))
	for _, part := range parts {
		number, ok := parseNumber(part, nil)
		if !ok {
			return nil, false
		}
		items = append(items, number)
	}
	sort.Float64s(items)

	return items, true
}

func setsEqual(actual, expected []float64) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if math.Abs(actual[index]-expected[index]) > numberEpsilon {
			return false
		}
	}

	return true
}

func formatSet(items []float64) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, formatNumber(item))
	}

	return "{" + strings.Join(parts, ";") + "}"
}

func normalizeIntervalString(value string) string {
	value = strings.ReplaceAll(normalizeBase(value), " ", "")

	return strings.ReplaceAll(value, ",", ";")
}
