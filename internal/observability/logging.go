package observability

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// LogConfig конфигурация для логгера
type LogConfig struct {
	Level  string `json:"level" default:"info"`
	Format string `json:"format" default:"json"` // json или text
}

// NewLogger создает новый structured logger
func NewLogger(config LogConfig) *slog.Logger {
	var level slog.Level
	switch config.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Форматируем время в ISO 8601
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}
	
	var handler slog.Handler
	if config.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	
	return slog.New(handler)
}

// BusinessLogger содержит логгеры для различных доменов
type BusinessLogger struct {
	Logger *slog.Logger
}

// NewBusinessLogger создает логгер для бизнес-логики
func NewBusinessLogger(logger *slog.Logger) *BusinessLogger {
	return &BusinessLogger{
		Logger: logger,
	}
}

// LogUserAction логирует действия пользователя
func (bl *BusinessLogger) LogUserAction(ctx context.Context, userID, familyID, action string, details map[string]interface{}) {
	logArgs := []any{
		slog.String("domain", "user"),
		slog.String("user_id", userID),
		slog.String("family_id", familyID),
		slog.String("action", action),
	}
	
	// Добавляем дополнительные детали
	for key, value := range details {
		logArgs = append(logArgs, slog.Any(key, value))
	}
	
	bl.Logger.InfoContext(ctx, "User action performed", logArgs...)
}

// LogTransactionEvent логирует события транзакций
func (bl *BusinessLogger) LogTransactionEvent(ctx context.Context, transactionID, userID, familyID, eventType string, amount float64, currency string) {
	bl.Logger.InfoContext(ctx, "Transaction event",
		slog.String("domain", "transaction"),
		slog.String("transaction_id", transactionID),
		slog.String("user_id", userID),
		slog.String("family_id", familyID),
		slog.String("event_type", eventType),
		slog.Float64("amount", amount),
		slog.String("currency", currency),
	)
}

// LogBudgetEvent логирует события бюджетов
func (bl *BusinessLogger) LogBudgetEvent(ctx context.Context, budgetID, userID, familyID, eventType string, details map[string]interface{}) {
	logArgs := []any{
		slog.String("domain", "budget"),
		slog.String("budget_id", budgetID),
		slog.String("user_id", userID),
		slog.String("family_id", familyID),
		slog.String("event_type", eventType),
	}
	
	for key, value := range details {
		logArgs = append(logArgs, slog.Any(key, value))
	}
	
	bl.Logger.InfoContext(ctx, "Budget event", logArgs...)
}

// LogSecurityEvent логирует события безопасности
func (bl *BusinessLogger) LogSecurityEvent(ctx context.Context, eventType, userID, ip, userAgent string, success bool, details map[string]interface{}) {
	logArgs := []any{
		slog.String("domain", "security"),
		slog.String("event_type", eventType),
		slog.String("user_id", userID),
		slog.String("ip_address", ip),
		slog.String("user_agent", userAgent),
		slog.Bool("success", success),
	}
	
	for key, value := range details {
		logArgs = append(logArgs, slog.Any(key, value))
	}
	
	level := slog.LevelInfo
	if !success {
		level = slog.LevelWarn
	}
	
	bl.Logger.Log(ctx, level, "Security event", logArgs...)
}

// LogDatabaseOperation логирует операции с базой данных
func (bl *BusinessLogger) LogDatabaseOperation(ctx context.Context, operation, collection, query string, duration time.Duration, success bool, err error) {
	logArgs := []any{
		slog.String("domain", "database"),
		slog.String("operation", operation),
		slog.String("collection", collection),
		slog.String("query", query),
		slog.Duration("duration", duration),
		slog.Bool("success", success),
	}
	
	if err != nil {
		logArgs = append(logArgs, slog.String("error", err.Error()))
		bl.Logger.ErrorContext(ctx, "Database operation failed", logArgs...)
	} else {
		bl.Logger.DebugContext(ctx, "Database operation completed", logArgs...)
	}
}

// LogAPIError логирует ошибки API
func (bl *BusinessLogger) LogAPIError(ctx context.Context, endpoint, method, errorType, message string, statusCode int, userID string) {
	bl.Logger.ErrorContext(ctx, "API error occurred",
		slog.String("domain", "api"),
		slog.String("endpoint", endpoint),
		slog.String("method", method),
		slog.String("error_type", errorType),
		slog.String("message", message),
		slog.Int("status_code", statusCode),
		slog.String("user_id", userID),
	)
}

// LogPerformanceMetric логирует метрики производительности
func (bl *BusinessLogger) LogPerformanceMetric(ctx context.Context, operation string, duration time.Duration, metadata map[string]interface{}) {
	logArgs := []any{
		slog.String("domain", "performance"),
		slog.String("operation", operation),
		slog.Duration("duration", duration),
	}
	
	for key, value := range metadata {
		logArgs = append(logArgs, slog.Any(key, value))
	}
	
	bl.Logger.InfoContext(ctx, "Performance metric", logArgs...)
}