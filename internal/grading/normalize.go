package grading

import (
	"strings"
	"unicode"
)

// normalizeBase приводит минусы, неразрывные пробелы и внешние пробелы к единому виду.
func normalizeBase(value string) string {
	value = strings.TrimSpace(value)

	return strings.Map(func(r rune) rune {
		switch r {
		case '\u2212', '\u2010', '\u2011', '\u2012', '\u2013', '\u2014', '\u2015', '\uFE63', '\uFF0D':
			return '-'
		case '\u00A0', '\u202F':
			return ' '
		default:
			return r
		}
	}, value)
}

// collapseSpaces заменяет последовательности пробелов одним пробелом.
func collapseSpaces(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// normalizeText применяет настройки текстовой проверки.
func normalizeText(value string, caseSensitive, ignoreYo, ignoreHyphen bool) string {
	value = collapseSpaces(normalizeBase(value))
	if !caseSensitive {
		value = strings.ToLower(value)
	}
	if ignoreYo {
		value = strings.NewReplacer("ё", "е", "Ё", "Е").Replace(value)
	}
	if ignoreHyphen {
		value = strings.ReplaceAll(value, "-", " ")
		value = collapseSpaces(value)
	}

	return value
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func onlyDigits(value string) (string, bool) {
	value = normalizeBase(value)
	var builder strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsSpace(r):
			continue
		case isDigit(r):
			builder.WriteRune(r)
		default:
			return "", false
		}
	}

	return builder.String(), true
}
