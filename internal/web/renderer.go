package web

import (
	"fmt"
	"html/template"
	"io"
	"os"
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
	funcMap := createTemplateFuncMap()
	tmpl, err := loadAllTemplates(templatesDir, funcMap)
	if err != nil {
		return nil, err
	}

	return &TemplateRenderer{
		templates: tmpl,
	}, nil
}

// createTemplateFuncMap создает карту функций для шаблонов
func createTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"add":            templateAdd,
		"sub":            templateSub,
		"mul":            templateMul,
		"div":            templateDiv,
		"abs":            templateAbs,
		"formatCurrency": formatCurrency,
		"formatDate":     formatDate,
		"safe":           templateSafe,
		"dict":           createDict,
		"title":          titleCase,
		"deref":          derefBool,
	}
}

// templateAdd складывает два числа любого типа
func templateAdd(a, b any) float64 {
	aFloat := convertToFloat64(a)
	bFloat := convertToFloat64(b)
	return aFloat + bFloat
}

// templateSub вычитает два числа любого типа
func templateSub(a, b any) float64 {
	aFloat := convertToFloat64(a)
	bFloat := convertToFloat64(b)
	return aFloat - bFloat
}

// templateMul умножает два целых числа
func templateMul(a, b int) int {
	return a * b
}

// templateDiv делит два целых числа
func templateDiv(a, b int) int {
	if b != 0 {
		return a / b
	}
	return 0
}

// templateAbs возвращает абсолютное значение числа
func templateAbs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

// templateSafe создает безопасный HTML
func templateSafe(s string) template.HTML {
	return template.HTML(s) //nolint:gosec // This is intentionally used for trusted content
}

// convertToFloat64 конвертирует значение в float64
func convertToFloat64(v any) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case float64:
		return val
	default:
		return 0
	}
}

// loadAllTemplates загружает все шаблоны
func loadAllTemplates(templatesDir string, funcMap template.FuncMap) (*template.Template, error) {
	tmpl := template.New("").Funcs(funcMap)

	// Загружаем layouts
	tmpl, err := loadLayoutTemplates(tmpl, templatesDir)
	if err != nil {
		return nil, err
	}

	// Загружаем компоненты
	tmpl, err = loadComponentTemplates(tmpl, templatesDir)
	if err != nil {
		return nil, err
	}

	// Загружаем страницы
	tmpl, err = loadPageTemplates(tmpl, templatesDir)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

// loadLayoutTemplates загружает шаблоны макетов
func loadLayoutTemplates(tmpl *template.Template, templatesDir string) (*template.Template, error) {
	layoutPattern := filepath.Join(templatesDir, "layouts", "*.html")
	return tmpl.ParseGlob(layoutPattern)
}

// loadComponentTemplates загружает шаблоны компонентов
func loadComponentTemplates(tmpl *template.Template, templatesDir string) (*template.Template, error) {
	componentPattern := filepath.Join(templatesDir, "components", "*.html")
	return tmpl.ParseGlob(componentPattern)
}

// loadPageTemplates загружает шаблоны страниц рекурсивно
func loadPageTemplates(tmpl *template.Template, templatesDir string) (*template.Template, error) {
	pagesDir := filepath.Join(templatesDir, "pages")
	err := filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			tmpl, err = tmpl.ParseFiles(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return tmpl, err
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

// derefBool dereferences a boolean pointer, returning false if nil
func derefBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}
