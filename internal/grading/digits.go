package grading

import "sort"

func checkDigits(parsed spec, answer string) Result {
	actual, ok := onlyDigits(answer)
	if !ok {
		return mismatch(parsed.Accepted, answer)
	}
	if parsed.Length > 0 && len(actual) != parsed.Length {
		return mismatch(parsed.Accepted, actual)
	}

	orderMatters := true
	if parsed.OrderMatters != nil {
		orderMatters = *parsed.OrderMatters
	}
	if !orderMatters {
		actual = sortDigits(actual)
	}

	for _, item := range parsed.Accepted {
		expected, ok := onlyDigits(item)
		if !ok {
			continue
		}
		if parsed.Length > 0 && len(expected) != parsed.Length {
			continue
		}
		if !orderMatters {
			expected = sortDigits(expected)
		}
		if actual == expected {
			return matched(parsed.Accepted, actual)
		}
	}

	return mismatch(parsed.Accepted, actual)
}

func sortDigits(value string) string {
	runes := []rune(value)
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })

	return string(runes)
}
