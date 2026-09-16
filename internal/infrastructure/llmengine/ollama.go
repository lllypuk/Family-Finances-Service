// Package llmengine — плечо распознавания поверх github.com/lllypuk/llm; промпт и разбор живут в internal/recognize.
package llmengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lllypuk/llm"
	"github.com/lllypuk/llm/ollama"

	"family-budget-service/internal/recognize"
)

const (
	task          = "receipt.screenshot"
	attempts      = 2
	maxRetryAfter = 10 * time.Second

	outcomeBadAnswer = "bad_answer"
)

// Engine распознаёт скриншоты одним вызовом модели; повторы — только транспортные, внутри llm.Client.
type Engine struct {
	client *llm.Client
	model  string
	logger *slog.Logger
}

// New собирает движок над демоном Ollama; obs может быть nil.
func New(host, model string, timeout time.Duration, obs llm.Observer, logger *slog.Logger) *Engine {
	client := llm.New(ollama.New(host), timeout)
	client.Attempts = attempts
	client.MaxRetryAfter = maxRetryAfter
	client.Observe = obs

	return &Engine{client: client, model: model, logger: logger}
}

// Budget — худший срок вызова со всеми попытками и паузами.
func (e *Engine) Budget() time.Duration {
	return e.client.Budget()
}

// Recognize зовёт модель и нормализует ответ; неразобранный ответ — recognize.ErrBadAnswer без второго вызова.
func (e *Engine) Recognize(ctx context.Context, input recognize.Input) (recognize.Result, error) {
	res, err := e.client.Chat(ctx, e.request(input))
	if err != nil {
		var call *llm.CallError
		if !errors.As(err, &call) {
			return recognize.Result{}, fmt.Errorf("llmengine: %w", err)
		}

		e.log(ctx, call.Report, string(call.Class), -1)

		if call.Class == llm.RetryNever {
			return recognize.Result{}, fmt.Errorf("llmengine: %w", err)
		}

		return recognize.Result{}, &recognize.UnavailableError{RetryAfter: call.RetryAfter, Err: err}
	}

	answer, err := recognize.Parse(res.Text)
	if err != nil {
		report := res.Report
		report.Outcome = outcomeBadAnswer
		e.log(ctx, report, "", -1)

		return recognize.Result{}, err
	}

	result := recognize.Normalize(answer, input)
	result.Model = res.Model
	e.log(ctx, res.Report, "", len(result.Items))

	return result, nil
}

func (e *Engine) request(input recognize.Input) llm.Request {
	messages := make([]llm.Message, 0, len(input.Images)+1)
	messages = append(messages, llm.Message{Role: llm.RoleSystem, Text: recognize.System(input)})

	for i, img := range input.Images {
		messages = append(messages, llm.Message{
			Role:   llm.RoleUser,
			Text:   recognize.UserText(i),
			Images: []llm.Image{{MIME: img.MIME, Data: img.Data}},
		})
	}

	return llm.Request{
		Task:        task,
		Model:       e.model,
		Messages:    messages,
		Output:      llm.Output{Mode: llm.ModeJSON},
		Temperature: llm.Ptr(0.0),
	}
}

// log — одна запись на вызов; ни байты картинок, ни текст ответа в неё не попадают. items < 0 — строк нет.
func (e *Engine) log(ctx context.Context, report llm.CallReport, class string, items int) {
	attrs := []slog.Attr{
		slog.String("task", report.Task),
		slog.String("provider", report.Provider),
		slog.String("model", report.RequestedModel),
		slog.String("outcome", report.Outcome),
		slog.Int("attempts", report.Attempts),
		slog.Duration("latency", report.Duration),
		slog.Int("input_tokens", report.Usage.InputTokens),
		slog.Int("output_tokens", report.Usage.OutputTokens),
		slog.Bool("usage_known", report.Usage.Known),
	}
	if class != "" {
		attrs = append(attrs, slog.String("class", class))
	}
	if items >= 0 {
		attrs = append(attrs, slog.Int("items", items))
	}

	level := slog.LevelInfo
	if report.Outcome != llm.OutcomeOK {
		level = slog.LevelWarn
	}

	e.logger.LogAttrs(ctx, level, "llm recognize call", attrs...)
}
