package llmengine

import "time"

// SetPause сокращает паузу между попытками, чтобы тест повтора не ждал секунды.
func (e *Engine) SetPause(d time.Duration) { e.client.Pause = d }

// NewYandexAt — NewYandex с адресом тестового сервера.
var NewYandexAt = newYandex //nolint:gochecknoglobals // экспорт для внешнего теста, только в _test.go
