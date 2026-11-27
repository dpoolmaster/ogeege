package grading

func checkText(parsed spec, answer string) Result {
	actual := normalizeText(answer, parsed.CaseSensitive, parsed.IgnoreYo, parsed.IgnoreHyphen)
	if actual == "" {
		return mismatch(parsed.Accepted, actual)
	}

	for _, item := range parsed.Accepted {
		expected := normalizeText(item, parsed.CaseSensitive, parsed.IgnoreYo, parsed.IgnoreHyphen)
		if actual == expected {
			return matched(parsed.Accepted, actual)
		}
	}

	return mismatch(parsed.Accepted, actual)
}
