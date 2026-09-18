//nolint:testpackage // нужен доступ к чистым функциям
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
)

func TestStatsPeriod(t *testing.T) {
	today := date.New(2026, time.September, 18)
	d := func(y int, m time.Month, day int) *date.Date {
		v := date.New(y, m, day)
		return &v
	}

	tests := []struct {
		name      string
		from, to  *date.Date
		wantStart date.Date
		wantEnd   date.Date
		wantErr   error
	}{
		{name: "defaults", wantStart: date.New(2025, time.October, 1), wantEnd: today},
		{
			name: "only to keeps default from", to: d(2026, time.March, 5),
			wantStart: date.New(2025, time.October, 1), wantEnd: date.New(2026, time.March, 5),
		},
		{
			name: "only from keeps today", from: d(2026, time.January, 31),
			wantStart: date.New(2026, time.January, 31), wantEnd: today,
		},
		{name: "reversed", from: d(2026, time.September, 2), to: d(2026, time.September, 1),
			wantErr: ErrInvalidStatsPeriod},
		{
			name: "only to before default from", to: d(2025, time.January, 1),
			wantErr: ErrInvalidStatsPeriod,
		},
		{
			name: "exactly 120 months", from: d(2017, time.January, 31), to: d(2026, time.December, 1),
			wantStart: date.New(2017, time.January, 31), wantEnd: date.New(2026, time.December, 1),
		},
		{name: "121 months", from: d(2016, time.December, 31), to: d(2026, time.December, 1),
			wantErr: ErrStatsPeriodTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := statsPeriod(today, tt.from, tt.to)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantEnd, end)
		})
	}
}
