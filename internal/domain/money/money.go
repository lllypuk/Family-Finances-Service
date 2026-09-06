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
	return []byte(strconv.FormatInt(int64(m), decimalBase)), nil
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
	case float64:
		*m = Minor(v)
	default:
		return fmt.Errorf("money: unsupported source type %T", src)
	}

	return nil
}
