package web

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

func TestPercent(t *testing.T) {
	assert.Equal(t, "1,25 %", Percent(1.25))
	assert.Equal(t, "1,5 %", Percent(1.5))
	assert.Equal(t, "2 %", Percent(2))
}

func TestPercentagePoints(t *testing.T) {
	assert.Equal(t, "−0,25", PercentagePoints(1.25-1.5))
	assert.Equal(t, "+0,07", PercentagePoints(1.44-1.37))
	assert.Equal(t, "±0", PercentagePoints(0.001))
}

func TestDates(t *testing.T) {
	date := time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, "4. März 2025", DateLong(date))
	assert.Equal(t, "04.03.2025", DateShort(date))
}

func TestTemplatesParse(t *testing.T) {
	_, err := PageTemplates()
	assert.Equal(t, nil, err)

	_, err = MailTemplates()
	assert.Equal(t, nil, err)
}
