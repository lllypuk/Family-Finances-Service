package recognize_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/recognize"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)

	return string(data)
}

func TestParse_ProbeFixture(t *testing.T) {
	answer, err := recognize.Parse(readFixture(t, "probe_list.json"))
	require.NoError(t, err)

	require.Len(t, answer.Items, 4)
	assert.False(t, answer.Incomplete)
	assert.Equal(t, "Пятёрочка", answer.Items[0].Description)
	require.NotNil(t, answer.Items[3].DateText)
	assert.Equal(t, "вчера", *answer.Items[3].DateText)
}

func TestParse_Fence(t *testing.T) {
	body := readFixture(t, "probe_list.json")

	for name, text := range map[string]string{
		"json fence":  "```json\n" + body + "\n```",
		"bare fence":  "```\n" + body + "```\n",
		"spaces":      "  \n" + body + "\n\n",
		"inline text": "```json\n" + body + "\n```  ",
	} {
		t.Run(name, func(t *testing.T) {
			answer, err := recognize.Parse(text)
			require.NoError(t, err)
			assert.Len(t, answer.Items, 4)
		})
	}
}

func TestParse_EmptyItems(t *testing.T) {
	answer, err := recognize.Parse(`{"items": [], "incomplete": true}`)
	require.NoError(t, err)

	assert.Empty(t, answer.Items)
	assert.True(t, answer.Incomplete)
}

func TestParse_BadAnswer(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "items is an object", text: `{"items": {}}`},
		{name: "items is null", text: `{"items": null}`},
		{name: "items missing", text: `{"incomplete": false}`},
		{name: "garbage after JSON", text: `{"items": []} и ещё текст`},
		{name: "second object", text: `{"items": []}{"items": []}`},
		{name: "not JSON", text: `Не удалось распознать`},
		{name: "top-level array", text: `[]`},
		{name: "empty", text: ""},
		{name: "wrong field type", text: `{"items": [{"description": 5}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := recognize.Parse(tt.text)
			require.ErrorIs(t, err, recognize.ErrBadAnswer)
		})
	}
}
