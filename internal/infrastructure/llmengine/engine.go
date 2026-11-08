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
	"github.com/lllypuk/llm/yandex"

	"family-budget-service/internal/recognize"
)

const (
	task          = "receipt.screenshot"
	attempts      = 2
	maxRetryAfter = 10 * time.Second

	outcomeBadAnswer = "bad_answer"

	yandexEndpoint = "https://ai.api.cloud.yandex.net/v1"
	// yandexMaxOutputTokens — запас на рассуждения qwen3.6: с умолчанием ответ обрезается (`length`), а это RetryNever.
	yandexMaxOutputTokens = 32768
)

// Engine распознаёт скриншоты одним вызовом модели; повторы — только транспортные, внутри llm.Client.
type Engine struct {
	client *llm.Client
	model  string
	opts   llm.Options
	logger *slog.Logger
}

// NewOllama собирает движок над демоном Ollama; obs может быть nil.
func NewOllama(host, model string, timeout time.Duration, obs llm.Observer, logger *slog.Logger) (*Engine, error) {
	return newEngine(ollama.New(host), model, timeout, llm.Options{}, obs, logger)
}

// NewYandex собирает движок над Yandex AI Studio с ключом API; obs может быть nil.
func NewYandex(
	folder, apiKey, model string,
	timeout time.Duration,
	obs llm.Observer,
	logger *slog.Logger,
) (*Engine, error) {
	return newYandex(yandexEndpoint, folder, apiKey, model, timeout, obs, logger)
}

func newYandex(
	endpoint, folder, apiKey, model string, timeout time.Duration, obs llm.Observer, logger *slog.Logger,
) (*Engine, error) {
	p, err := yandex.New(yandex.Config{Endpoint: endpoint, Folder: folder, Credentials: yandex.APIKey(apiKey)})
	if err != nil {
		return nil, fmt.Errorf("llmengine: %w", err)
	}

	return newEngine(p, model, timeout, llm.Options{MaxOutputTokens: yandexMaxOutputTokens}, obs, logger)
}

// newEngine отбивает модель, которой профиль плеча не обещает кадров: иначе отказ пришёл бы только на вызове.
func newEngine(
	p llm.Provider, model string, timeout time.Duration, opts llm.Options, obs llm.Observer, logger *slog.Logger,
) (*Engine, error) {
	if caps, ok := p.Capabilities(model); !ok || !caps.Vision {
		return nil, fmt.Errorf("llmengine: %s model %q does not accept images", p.Name(), model)
	}

	client := llm.New(p, timeout)
	client.Attempts = attempts
	client.MaxRetryAfter = maxRetryAfter
	client.Observe = obs

	opts.Temperature = llm.Ptr(0.0)

	return &Engine{client: client, model: model, opts: opts, logger: logger}, nil
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
		Task:     task,
		Model:    e.model,
		Messages: messages,
		Output:   llm.Output{Mode: llm.ModeJSON},
		Options:  e.opts,
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
		slog.Int("input_tokens", report.Usage.InputTokens()),
		slog.Int("output_tokens", report.Usage.OutputTokens()),
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
