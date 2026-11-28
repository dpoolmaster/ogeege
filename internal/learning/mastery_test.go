package learning

import "testing"

func TestNextMastery(t *testing.T) {
	tests := []struct {
		name       string
		old        float64
		correct    bool
		difficulty string
		want       float64
	}{
		{name: "первый верный easy", old: 0, correct: true, difficulty: "easy", want: 0.3},
		{name: "первый верный medium", old: 0, correct: true, difficulty: "medium", want: 0.33},
		{name: "первый верный hard", old: 0, correct: true, difficulty: "hard", want: 0.36},
		{name: "накопление", old: 0.3, correct: true, difficulty: "easy", want: 0.51},
		{name: "неверный ответ снижает", old: 0.5, correct: false, difficulty: "easy", want: 0.35},
		{name: "верхняя граница", old: 1, correct: true, difficulty: "hard", want: 1},
		{name: "нижняя граница", old: 0, correct: false, difficulty: "hard", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextMastery(tt.old, tt.correct, tt.difficulty)
			if diff := got - tt.want; diff > 0.0001 || diff < -0.0001 {
				t.Fatalf("NextMastery() = %f, ожидалось %f", got, tt.want)
			}
		})
	}
}

func TestLowConfidence(t *testing.T) {
	if !LowConfidence(2) {
		t.Fatal("две попытки должны считаться недостоверными")
	}
	if LowConfidence(3) {
		t.Fatal("три попытки должны считаться достоверными")
	}
}
