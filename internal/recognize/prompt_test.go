package recognize_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/recognize"
)

func TestSystem_ListsShapeAndCategories(t *testing.T) {
	input := defaultInput()
	input.Categories = []recognize.Category{
		{ID: uuid.New(), Type: transaction.TypeExpense, Path: "Еда / Кафе"},
		{ID: uuid.New(), Type: transaction.TypeIncome, Path: "Зарплата"},
	}

	prompt := recognize.System(input)

	for _, fragment := range []string{
		`"items"`, `"date_text"`, `"year_present"`, `"incomplete"`,
		"Картинок: 2", "«сегодня»", "балансы", "переводы между своими счетами",
		"expense: Еда / Кафе\n", "income: Зарплата\n",
	} {
		assert.Contains(t, prompt, fragment)
	}

	assert.NotContains(t, prompt, "%!", "форматирование промпта не сломано")
}

func TestSystem_NoCategories(t *testing.T) {
	assert.Contains(t, recognize.System(defaultInput()), "category всегда null")
}

func TestUserText_NumbersFromOne(t *testing.T) {
	assert.Equal(t, "Картинка 1", recognize.UserText(0))
	assert.Equal(t, "Картинка 5", recognize.UserText(4))
}
