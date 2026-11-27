package grading

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const numberEpsilon = 1e-9

func checkNumber(parsed spec, answer string) Result {
	actual, ok := parseNumber(answer, parsed.Units)
	if !ok {
		return mismatch(parsed.Accepted, strings.TrimSpace(answer))
	}

	for _, item := range parsed.Accepted {
		expected, ok := parseNumber(item, parsed.Units)
		if !ok {
			continue
		}
		if numbersEqual(actual, expected, parsed.Tolerance) {
			return matched(parsed.Accepted, formatNumber(actual))
		}
	}

	return mismatch(parsed.Accepted, formatNumber(actual))
}

func numbersEqual(actual, expected float64, tol *tolerance) bool {
	if math.Abs(actual-expected) <= numberEpsilon {
		return true
	}
	if tol == nil || tol.Value < 0 {
		return false
	}

	delta := math.Abs(actual - expected)
	switch tol.Type {
	case "relative":
		if math.Abs(expected) <= numberEpsilon {
			return delta <= tol.Value
		}
		return delta <= tol.Value*math.Abs(expected)
	default:
		return delta <= tol.Value
	}
}

func parseNumber(value string, units []string) (float64, bool) {
	value = normalizeBase(value)
	value = stripUnits(value, units)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "'", "")
	value = strings.ReplaceAll(value, "\u2009", "")
	if value == "" {
		return 0, false
	}

	if numerator, denominator, found := strings.Cut(value, "/"); found {
		top, ok := parseDecimal(numerator)
		if !ok {
			return 0, false
		}
		bottom, ok := parseDecimal(denominator)
		if !ok || math.Abs(bottom) <= numberEpsilon {
			return 0, false
		}

		return top / bottom, true
	}

	return parseDecimal(value)
}

func parseDecimal(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	comma := strings.LastIndex(value, ",")
	dot := strings.LastIndex(value, ".")
	switch {
	case comma >= 0 && dot >= 0:
		if comma > dot {
			value = strings.ReplaceAll(value, ".", "")
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	case comma >= 0:
		if strings.Count(value, ",") > 1 {
			value = strings.ReplaceAll(value, ",", "")
		} else {
			value = strings.ReplaceAll(value, ",", ".")
		}
	case dot >= 0:
		if strings.Count(value, ".") > 1 {
			last := strings.LastIndex(value, ".")
			value = strings.ReplaceAll(value[:last], ".", "") + value[last:]
		}
	}

	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}

	return result, true
}

func stripUnits(value string, units []string) string {
	value = strings.TrimSpace(value)
	for _, unit := range units {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			continue
		}
		if len(value) >= len(unit) && strings.EqualFold(value[len(value)-len(unit):], unit) {
			value = strings.TrimSpace(value[:len(value)-len(unit)])
		}
		if len(value) >= len(unit) && strings.EqualFold(value[:len(unit)], unit) {
			value = strings.TrimSpace(value[len(unit):])
		}
	}

	return value
}

func formatNumber(value float64) string {
	if math.Abs(value-math.Round(value)) <= numberEpsilon {
		return strconv.FormatInt(int64(math.Round(value)), 10)
	}

	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatInterval(lower, upper float64) string {
	return fmt.Sprintf("[%s;%s]", formatNumber(lower), formatNumber(upper))
}
