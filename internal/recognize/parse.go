package recognize

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Answer — ответ модели как есть, до нормализации.
type Answer struct {
	Items      []AnswerItem
	Incomplete bool
}

// AnswerItem — строка ответа модели; Source и Amount — string или json.Number.
type AnswerItem struct {
	Source      any     `json:"source"`
	Amount      any     `json:"amount"`
	Currency    *string `json:"currency"`
	Type        string  `json:"type"`
	Date        *string `json:"date"`
	DateText    *string `json:"date_text"`
	YearPresent *bool   `json:"year_present"`
	Description string  `json:"description"`
	Category    *string `json:"category"`
}

type answerEnvelope struct {
	Items      *[]AnswerItem `json:"items"`
	Incomplete bool          `json:"incomplete"`
}

// Parse снимает ограду из бэктиков и разбирает объект; лишние поля терпятся, всё прочее — ErrBadAnswer.
func Parse(text string) (Answer, error) {
	dec := json.NewDecoder(strings.NewReader(stripFence(text)))
	dec.UseNumber()

	var env answerEnvelope
	if err := dec.Decode(&env); err != nil {
		return Answer{}, fmt.Errorf("%w: %w", ErrBadAnswer, err)
	}

	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Answer{}, fmt.Errorf("%w: data after the JSON object", ErrBadAnswer)
	}

	if env.Items == nil {
		return Answer{}, fmt.Errorf("%w: items is not an array", ErrBadAnswer)
	}

	return Answer{Items: *env.Items, Incomplete: env.Incomplete}, nil
}

func stripFence(text string) string {
	s := strings.TrimSpace(text)
	if !strings.HasPrefix(s, "```") {
		return s
	}

	_, body, found := strings.Cut(s, "\n")
	if !found {
		return s
	}

	return strings.TrimSuffix(strings.TrimSpace(body), "```")
}
