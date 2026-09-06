// Package money хранит суммы в минимальных единицах валюты (копейки) целым числом.
package money

import (
	"database/sql/driver"
	"fmt"
	"strconv"
)

const (
	percentBase = 100
	decimalBase = 10
	half        = 2
)

// MaxAmount — потолок суммы, единый для сервисов, репозиториев и openapi (Money.maximum):
// выше него значение не влезает в контракт клиента.
const MaxAmount = Minor(99_999_999_999)

// Minor — сумма в минимальных единицах валюты. Сложение и вычитание — обычными операторами.
//
//nolint:recvcheck // Scan обязан быть на указателе, остальное — значение
type Minor int64

// Abs возвращает модуль суммы.
func (m Minor) Abs() Minor {
	if m < 0 {
		return -m
	}

	return m
}

// Percent возвращает долю m от total в процентах; при total == 0 — 0, а не NaN.
func (m Minor) Percent(total Minor) float64 {
	if total == 0 {
		return 0
	}

	return float64(m) / float64(total) * percentBase
}

// Share возвращает долю m от total в 0..1 — единица, в которой доли уходят в API;
// при total == 0 — 0.
func (m Minor) Share(total Minor) float64 {
	return m.Percent(total) / percentBase
}

// String печатает сумму десятичными цифрами — ровно то же, что уходит в JSON и в CSV.
func (m Minor) String() string {
	return strconv.FormatInt(int64(m), decimalBase)
}

// DivRound делит на n с округлением половины от нуля (half-up); при n == 0 возвращает 0.
func (m Minor) DivRound(n int64) Minor {
	if n == 0 {
		return 0
	}

	negative := (m < 0) != (n < 0)
	value, divisor := int64(m.Abs()), int64(Minor(n).Abs())

	quotient := value / divisor
	if value%divisor >= (divisor+1)/half {
		quotient++
	}

	if negative {
		return Minor(-quotient)
	}

	return Minor(quotient)
}

// MarshalJSON пишет сумму числом, а не строкой.
func (m Minor) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

// Value пишет сумму в БД целым: без него запись зависела бы от того, что каждый
// вызов не забыл привести к int64.
func (m Minor) Value() (driver.Value, error) {
	return int64(m), nil
}

// Scan читает INTEGER-колонку.
func (m *Minor) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*m = 0
	case int64:
		*m = Minor(v)
	default:
		return fmt.Errorf("money: unsupported source type %T", src)
	}

	return nil
}
