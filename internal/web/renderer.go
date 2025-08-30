package web

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	// Number formatting thresholds
	millionThreshold  = 1000000
	thousandThreshold = 1000
	millionDivisor    = 1000000
	thousandDivisor   = 1000
)

// TemplateRenderer реализует echo.Renderer интерфейс для Go Templates
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer создает новый рендерер шаблонов
func NewTemplateRenderer(templatesDir string) (*TemplateRenderer, error) {
	// Определяем функции для использования в шаблонах
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b != 0 {
				return a / b
			}
			return 0
		},
		"abs": func(a float64) float64 {
			if a < 0 {
				return -a
			}
			return a
		},
		"formatCurrency": formatCurrency,
		"formatDate":     formatDate,
		"safe": func(s string) template.HTML {
			return template.HTML(s) //nolint:gosec // This is intentionally used for trusted content
		},
		"dict":  createDict,
		"title": titleCase,
	}

	// Загружаем все шаблоны
	tmpl := template.New("").Funcs(funcMap)

	// Загружаем layouts
	layoutPattern := filepath.Join(templatesDir, "layouts", "*.html")
	tmpl, err := tmpl.ParseGlob(layoutPattern)
	if err != nil {
		return nil, err
	}

	// Загружаем компоненты
	componentPattern := filepath.Join(templatesDir, "components", "*.html")
	tmpl, err = tmpl.ParseGlob(componentPattern)
	if err != nil {
		return nil, err
	}

	// Загружаем страницы
	pagePattern := filepath.Join(templatesDir, "pages", "*.html")
	tmpl, err = tmpl.ParseGlob(pagePattern)
	if err != nil {
		return nil, err
	}

	return &TemplateRenderer{
		templates: tmpl,
	}, nil
}

// Render рендерит шаблон с данными
func (t *TemplateRenderer) Render(w io.Writer, name string, data any, _ echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// formatCurrency форматирует сумму с валютой
func formatCurrency(amount float64, currency string) string {
	switch currency {
	case "USD":
		return "$" + formatNumber(amount)
	case "EUR":
		return "€" + formatNumber(amount)
	case "RUB":
		return formatNumber(amount) + " ₽"
	default:
		return formatNumber(amount) + " " + currency
	}
}

// formatNumber форматирует число с разделителями тысяч
func formatNumber(amount float64) string {
	// Простое форматирование - можно улучшить
	if amount >= millionThreshold {
		return fmt.Sprintf("%.1fM", amount/millionDivisor)
	} else if amount >= thousandThreshold {
		return fmt.Sprintf("%.1fK", amount/thousandDivisor)
	}
	return fmt.Sprintf("%.2f", amount)
}

// formatDate форматирует дату
func formatDate(_ any) string {
	// TODO: Реализовать форматирование даты
	return "01.01.2024"
}

// createDict создает словарь для передачи в шаблон
func createDict(values ...any) map[string]any {
	dict := make(map[string]any)
	for i := 0; i < len(values); i += 2 {
		if i+1 < len(values) {
			if key, ok := values[i].(string); ok {
				dict[key] = values[i+1]
			}
		}
	}
	return dict
}

// titleCase конвертирует строку в заглавный регистр (замена для deprecated strings.Title)
func titleCase(s string) string {
	if s == "" {
		return s
	}
	
	// Заменяем подчеркивания на пробелы для читаемости
	s = strings.ReplaceAll(s, "_", " ")
	
	// Разбиваем на слова и капитализируем первую букву каждого
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	
	return strings.Join(words, " ")
}
