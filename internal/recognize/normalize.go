package recognize

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
)

const (
	minorDigits    = 2
	minorPerUnit   = 100
	maxUnitDigits  = 12
	decimalBase    = 10
	pathSeparator  = "/"
	pathJoinString = " / "
)

// Normalize переводит ответ модели в кандидатов; строка с негодной суммой или типом отбрасывается и помечает Incomplete.
func Normalize(answer Answer, input Input) Result {
	result := Result{Items: make([]Item, 0, min(len(answer.Items), MaxItems)), Incomplete: answer.Incomplete}

	for _, raw := range answer.Items {
		item, ok := normalizeItem(raw, input)
		if !ok {
			result.Incomplete = true

			continue
		}

		if len(result.Items) == MaxItems {
			result.Incomplete = true

			break
		}

		result.Items = append(result.Items, item)
	}

	return result
}

func normalizeItem(raw AnswerItem, input Input) (Item, bool) {
	typ := transaction.Type(strings.ToLower(strings.TrimSpace(raw.Type)))
	if typ != transaction.TypeIncome && typ != transaction.TypeExpense {
		return Item{}, false
	}

	amount, ok := parseAmount(raw.Amount)
	if !ok {
		return Item{}, false
	}

	item := Item{
		Source:      normalizeSource(raw.Source, len(input.Images)),
		AmountMinor: amount,
		Currency:    normalizeCurrency(raw.Currency),
		Type:        typ,
		Description: truncateRunes(strings.TrimSpace(raw.Description), maxDescription),
		CategoryID:  matchCategory(raw.Category, typ, input.Categories),
		Similar:     []Similar{},
	}
	item.Date, item.DateAssumed = normalizeDate(raw.Date, raw.YearPresent, input.Today)

	return item, true
}

// parseAmount разбирает десятичную строку в минорные единицы без float64; знак игнорируется — его несёт тип.
func parseAmount(value any) (money.Minor, bool) {
	var s string

	switch v := value.(type) {
	case string:
		s = v
	case json.Number:
		s = v.String()
	default:
		return 0, false
	}

	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', ' ', ' ', ' ':
			return -1
		case ',':
			return '.'
		default:
			return r
		}
	}, strings.TrimSpace(s))
	s = strings.TrimLeft(s, "+-−")

	units, fraction, _ := strings.Cut(s, ".")
	if units == "" || len(units) > maxUnitDigits || len(fraction) > minorDigits ||
		!allDigits(units) || !allDigits(fraction) {
		return 0, false
	}

	fraction += strings.Repeat("0", minorDigits-len(fraction))

	whole, err := strconv.ParseInt(units, decimalBase, 64)
	if err != nil {
		return 0, false
	}

	cents, err := strconv.ParseInt(fraction, decimalBase, 64)
	if err != nil {
		return 0, false
	}

	amount := money.Minor(whole*minorPerUnit + cents)
	if amount <= 0 || amount > money.MaxAmount {
		return 0, false
	}

	return amount, true
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// normalizeSource переводит номер картинки из промпта (с единицы) в индекс Input.Images.
func normalizeSource(value any, images int) *int {
	var (
		n   int64
		err error
	)

	switch v := value.(type) {
	case json.Number:
		n, err = v.Int64()
	case string:
		n, err = strconv.ParseInt(strings.TrimSpace(v), decimalBase, 64)
	default:
		return nil
	}

	if err != nil || n < 1 || n > int64(images) {
		return nil
	}

	index := int(n - 1)

	return &index
}

func normalizeCurrency(value *string) *string {
	if value == nil {
		return nil
	}

	s := strings.ToLower(strings.TrimSpace(*value))
	s = strings.TrimSpace(strings.TrimSuffix(s, "."))

	code := currencyCode(s)
	if code == "" {
		return nil
	}

	return &code
}

func currencyCode(s string) string {
	switch s {
	case "₽", "р", "руб", "рубль", "рубля", "рублей", "rub", "rur":
		return "RUB"
	case "$", "usd", "долл", "доллар", "доллара", "долларов":
		return "USD"
	case "€", "eur", "евро":
		return "EUR"
	case "£", "gbp":
		return "GBP"
	case "cny", "юань", "юаня", "юаней":
		return "CNY"
	case "₸", "kzt", "тенге":
		return "KZT"
	case "byn", "бел. руб", "бел.руб":
		return "BYN"
	case "₺", "try", "лира", "лир":
		return "TRY"
	case "₾", "gel", "лари":
		return "GEL"
	case "֏", "amd", "драм", "драмов":
		return "AMD"
	default:
		return ""
	}
}

// normalizeDate применяет правило года: без года — год опорной даты, а если так выходит будущее — предыдущий.
func normalizeDate(value *string, yearPresent *bool, today date.Date) (*date.Date, bool) {
	if value == nil {
		return nil, false
	}

	parsed, err := date.Parse(strings.TrimSpace(*value))
	if err != nil {
		return nil, false
	}

	if yearPresent != nil && *yearPresent {
		if parsed.After(today) {
			return nil, false
		}

		return &parsed, false
	}

	for _, year := range []int{today.Year, today.Year - 1} {
		candidate := date.New(year, parsed.Month, parsed.Day)
		if candidate.Month != parsed.Month || candidate.After(today) {
			continue
		}

		return &candidate, true
	}

	return nil, false
}

// matchCategory ищет единственную категорию того же типа с тем же путём; ноль или несколько совпадений — nil.
func matchCategory(value *string, typ transaction.Type, categories []Category) *uuid.UUID {
	if value == nil {
		return nil
	}

	s := strings.TrimSpace(*value)
	if prefix, rest, found := strings.Cut(s, ":"); found && transaction.Type(strings.TrimSpace(prefix)) == typ {
		s = rest
	}

	path := canonicalPath(s)
	if path == "" {
		return nil
	}

	var match *uuid.UUID

	for i := range categories {
		c := &categories[i]
		if c.Type != typ || !strings.EqualFold(canonicalPath(c.Path), path) {
			continue
		}

		if match != nil {
			return nil
		}

		match = &c.ID
	}

	if match == nil {
		return nil
	}

	id := *match

	return &id
}

func canonicalPath(s string) string {
	parts := strings.Split(s, pathSeparator)
	for i, part := range parts {
		parts[i] = strings.Join(strings.Fields(part), " ")
		if parts[i] == "" {
			return ""
		}
	}

	return strings.Join(parts, pathJoinString)
}

func truncateRunes(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	return string([]rune(s)[:limit])
}
