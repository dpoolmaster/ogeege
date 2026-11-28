package learning

// difficultyFactor задаёт коэффициент сложности задания.
func difficultyFactor(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 1.0
	case "hard":
		return 1.2
	default:
		return 1.1
	}
}

// Clamp01 ограничивает значение диапазоном [0, 1].
func Clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}

	return value
}

// NextMastery пересчитывает освоение по формуле design §6.
func NextMastery(old float64, correct bool, difficulty string) float64 {
	next := old * (1 - 0.3)
	if correct {
		next += 0.3 * difficultyFactor(difficulty)
	}

	return Clamp01(next)
}

// LowConfidence сообщает, что освоение недостоверно при малом числе попыток.
func LowConfidence(attempts int) bool {
	return attempts < 3
}
