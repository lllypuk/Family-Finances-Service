package llmengine

import "time"

// SetPause сокращает паузу между попытками, чтобы тест повтора не ждал секунды.
func (e *Engine) SetPause(d time.Duration) { e.client.Pause = d }
