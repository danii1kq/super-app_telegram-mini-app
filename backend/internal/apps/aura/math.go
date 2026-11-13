package aura

import (
	"fmt"
	"math/rand"
)

// Topic is one kind of exercise. The list grows with the player's level, so a
// beginner never sees an equation and a veteran never sees 2+3.
type Topic string

const (
	TopicAdd      Topic = "add"
	TopicSub      Topic = "sub"
	TopicMul      Topic = "mul"
	TopicDiv      Topic = "div"
	TopicEquation Topic = "equation"
	TopicFraction Topic = "fraction"
	TopicPercent  Topic = "percent"
	TopicPower    Topic = "power"
	TopicSquare   Topic = "square"
	TopicSqrt     Topic = "sqrt"
)

// topicNames is the human label shown above the question.
var topicNames = map[Topic]string{
	TopicAdd:      "сложение",
	TopicSub:      "вычитание",
	TopicMul:      "умножение",
	TopicDiv:      "деление",
	TopicEquation: "уравнения",
	TopicFraction: "дроби",
	TopicPercent:  "проценты",
	TopicPower:    "степени",
	TopicSquare:   "квадраты",
	TopicSqrt:     "корни",
}

// Question is one exercise with its four options already shuffled.
type Question struct {
	Text      string `json:"text"`
	Answer    int    `json:"answer"`
	Options   []int  `json:"options"`
	Topic     Topic  `json:"topic"`
	TopicName string `json:"topic_name"`
}

// topicsFor picks the pool for a level. The bands are the same as the original
// bot: arithmetic first, then expressions, then algebra and roots.
func topicsFor(level int) []Topic {
	switch {
	case level < 5:
		return []Topic{TopicAdd, TopicSub}
	case level < 10:
		return []Topic{TopicAdd, TopicSub, TopicMul, TopicDiv}
	case level < 20:
		return []Topic{TopicMul, TopicDiv, TopicPercent, TopicFraction, TopicSquare}
	case level < 35:
		return []Topic{TopicEquation, TopicPercent, TopicPower, TopicSquare, TopicSqrt, TopicFraction}
	default:
		return []Topic{TopicEquation, TopicPercent, TopicPower, TopicSquare, TopicSqrt, TopicFraction, TopicDiv}
	}
}

// Generate builds one question for a level. rnd is passed in so tests can be
// deterministic and the handler can use its own source.
func Generate(rnd *rand.Rand, level int, topic Topic) Question {
	if level < 1 {
		level = 1
	}
	chosen := topic
	if _, ok := topicNames[topic]; !ok {
		pool := topicsFor(level)
		chosen = pool[rnd.Intn(len(pool))]
	}

	text, answer := build(rnd, chosen, level)
	return Question{
		Text:      text,
		Answer:    answer,
		Options:   options(rnd, answer),
		Topic:     chosen,
		TopicName: topicNames[chosen],
	}
}

// options returns four distinct values including the answer. Distractors are
// close to the answer, so a guess is not free.
func options(rnd *rand.Rand, answer int) []int {
	deltas := []int{-10, -5, -3, -2, -1, 1, 2, 3, 5, 10}
	seen := map[int]bool{answer: true}
	out := []int{answer}
	for len(out) < 4 {
		delta := deltas[rnd.Intn(len(deltas))]
		candidate := answer + delta
		if answer >= 0 && candidate < 0 {
			continue
		}
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	rnd.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func build(rnd *rand.Rand, topic Topic, level int) (string, int) {
	switch topic {
	case TopicAdd:
		a := rnd.Intn(30+level*5) + 10
		b := rnd.Intn(30+level*5) + 10
		return fmt.Sprintf("%d + %d", a, b), a + b
	case TopicSub:
		a := rnd.Intn(40+level*5) + 30
		b := rnd.Intn(a-1) + 1
		return fmt.Sprintf("%d − %d", a, b), a - b
	case TopicMul:
		highA := min(25, 10+level/10)
		highB := min(12, 4+level/15)
		a := rnd.Intn(highA-6+1) + 6
		b := rnd.Intn(highB-3+1) + 3
		return fmt.Sprintf("%d × %d", a, b), a * b
	case TopicDiv:
		highB := min(20, 6+level/10)
		highQ := min(20, 5+level/15)
		b := rnd.Intn(highB-4+1) + 4
		q := rnd.Intn(highQ-3+1) + 3
		return fmt.Sprintf("%d ÷ %d", b*q, b), q
	case TopicEquation:
		highA := min(12, 5+level/15)
		highX := min(30, 5+level/10)
		a := rnd.Intn(highA-2+1) + 2
		x := rnd.Intn(highX-2+1) + 2
		if rnd.Intn(2) == 0 {
			b := rnd.Intn(min(60, 10+level/2)) + 1
			return fmt.Sprintf("%dx + %d = %d", a, b, a*x+b), x
		}
		b := rnd.Intn(a*x-1) + 1
		return fmt.Sprintf("%dx − %d = %d", a, b, a*x-b), x
	case TopicSquare:
		high := min(25, 10+level/5)
		a := rnd.Intn(high-5+1) + 5
		return fmt.Sprintf("%d²", a), a * a
	case TopicSqrt:
		high := min(25, 8+level/5)
		a := rnd.Intn(high-4+1) + 4
		return fmt.Sprintf("√%d", a*a), a
	case TopicFraction:
		denoms := []int{2, 4, 5, 10}
		q := denoms[rnd.Intn(len(denoms))]
		p := rnd.Intn(q-1) + 1
		high := min(100, 20+level)
		base := rnd.Intn(high-10+1) + 10
		n := base * q
		return fmt.Sprintf("%d/%d от %d", p, q, n), n * p / q
	case TopicPercent:
		percents := []int{10, 20, 25, 50}
		p := percents[rnd.Intn(len(percents))]
		high := min(100, 20+level)
		base := rnd.Intn(high-10+1) + 10
		mult := 10
		if p == 25 {
			mult = 4
		}
		n := base * mult
		return fmt.Sprintf("%d%% от %d", p, n), n * p / 100
	case TopicPower:
		bases := []int{2, 3}
		if level >= 50 {
			bases = []int{2, 3, 4}
		}
		base := bases[rnd.Intn(len(bases))]
		highExp := min(7, 2+level/20)
		exp := rnd.Intn(highExp-2+1) + 2
		value := 1
		for i := 0; i < exp; i++ {
			value *= base
		}
		return fmt.Sprintf("%d^%d", base, exp), value
	}
	// Unreachable: topicsFor only returns known topics.
	return "2 + 2", 4
}

// hint explains a wrong answer. The text is feedback, not a lesson: it names
// the rule to check, which is what a player actually needs mid-run.
func hint(topic Topic) string {
	switch topic {
	case TopicAdd, TopicSub:
		return "Проверь разряды: сначала единицы, потом десятки."
	case TopicMul:
		return "Разбей умножение на части и проверь по таблице."
	case TopicDiv:
		return "Проверь деление обратным умножением."
	case TopicEquation:
		return "Перенеси свободный член и раздели на коэффициент."
	case TopicPercent:
		return "Процент это сотая доля: умножь число на p/100."
	case TopicFraction:
		return "Раздели число на знаменатель и умножь на числитель."
	case TopicPower:
		return "Проверь основание и количество множителей."
	case TopicSquare:
		return "Квадрат это число, умноженное само на себя."
	case TopicSqrt:
		return "Корень это число, которое в квадрате даёт подкоренное."
	}
	return "Проверь порядок действий."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
