package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/recognize"
)

// maxSimilar — сколько похожих операций получает кандидат.
const maxSimilar = 3

// Исходы распознавания — значения метки outcome у ffs_recognitions_total.
const (
	recognizeOutcomeOK          = "ok"
	recognizeOutcomeEmpty       = "empty"
	recognizeOutcomeFailed      = "failed"
	recognizeOutcomeUnavailable = "unavailable"
)

// Recognizer — плечо модели; реализация в internal/infrastructure/llmengine.
type Recognizer interface {
	Recognize(ctx context.Context, input recognize.Input) (recognize.Result, error)
	Budget() time.Duration
}

// RecognizeObserver считает вызовы распознавания; интерфейс здесь, чтобы services не зависел от пакета метрик.
type RecognizeObserver interface {
	ObserveRecognition(outcome string, d time.Duration, items int)
}

// NopRecognizeObserver — заглушка, где реестра нет.
type NopRecognizeObserver struct{}

func (NopRecognizeObserver) ObserveRecognition(string, time.Duration, int) {}

type recognizeService struct {
	engine       Recognizer
	families     FamilyRepository
	categories   CategoryRepository
	transactions TransactionRepository
	observer     RecognizeObserver
}

// NewRecognizeService собирает сервис; nil-engine выключает распознавание (recognize.ErrDisabled).
func NewRecognizeService(
	engine Recognizer,
	families FamilyRepository,
	categories CategoryRepository,
	transactions TransactionRepository,
	observer RecognizeObserver,
) RecognizeService {
	return &recognizeService{
		engine:       engine,
		families:     families,
		categories:   categories,
		transactions: transactions,
		observer:     observer,
	}
}

func (s *recognizeService) Budget() time.Duration {
	if s.engine == nil {
		return 0
	}

	return s.engine.Budget()
}

func (s *recognizeService) Recognize(ctx context.Context, images []recognize.Image) (recognize.Result, error) {
	started := time.Now()

	result, err := s.recognize(ctx, images)
	s.observer.ObserveRecognition(recognizeOutcome(result, err), time.Since(started), len(result.Items))

	return result, err
}

func (s *recognizeService) recognize(ctx context.Context, images []recognize.Image) (recognize.Result, error) {
	if s.engine == nil {
		return recognize.Result{}, recognize.ErrDisabled
	}

	input, err := s.input(ctx, images)
	if err != nil {
		return recognize.Result{}, err
	}

	result, err := s.engine.Recognize(ctx, input)
	if err != nil {
		return recognize.Result{}, err
	}

	for i := range result.Items {
		if err = s.fillSimilar(ctx, &result.Items[i]); err != nil {
			return recognize.Result{}, err
		}
	}

	return result, nil
}

func (s *recognizeService) input(ctx context.Context, images []recognize.Image) (recognize.Input, error) {
	family, err := s.families.Get(ctx)
	if err != nil {
		return recognize.Input{}, fmt.Errorf("get family: %w", err)
	}

	categories, err := s.categories.GetAll(ctx)
	if err != nil {
		return recognize.Input{}, fmt.Errorf("get categories: %w", err)
	}

	return recognize.Input{
		Images:     images,
		Categories: categoryPaths(categories),
		Today:      date.Today(family.Location()),
		Currency:   family.Currency,
	}, nil
}

// categoryPaths — активные категории с путём «Родитель / Имя»; подкатегория неактивного родителя в список не входит.
func categoryPaths(categories []*category.Category) []recognize.Category {
	byID := make(map[uuid.UUID]*category.Category, len(categories))
	for _, c := range categories {
		if c.IsActive {
			byID[c.ID] = c
		}
	}

	paths := make([]recognize.Category, 0, len(byID))

	for _, c := range categories {
		if !c.IsActive {
			continue
		}

		path := c.Name
		if c.ParentID != nil {
			parent, ok := byID[*c.ParentID]
			if !ok {
				continue
			}

			path = parent.Name + " / " + c.Name
		}

		paths = append(paths, recognize.Category{ID: c.ID, Type: transaction.Type(c.Type), Path: path})
	}

	return paths
}

// fillSimilar — до трёх операций с той же суммой и типом в пределах дня от даты кандидата.
func (s *recognizeService) fillSimilar(ctx context.Context, item *recognize.Item) error {
	item.Similar = []recognize.Similar{}
	if item.Date == nil {
		return nil
	}

	from, to := item.Date.AddDays(-1), item.Date.AddDays(1)
	amount, typ := item.AmountMinor, item.Type

	found, err := s.transactions.GetByFilter(ctx, transaction.Filter{
		Type:            &typ,
		DateFrom:        &from,
		DateTo:          &to,
		AmountFromMinor: &amount,
		AmountToMinor:   &amount,
		Limit:           maxSimilar,
	})
	if err != nil {
		return fmt.Errorf("find similar transactions: %w", err)
	}

	for _, tx := range found {
		item.Similar = append(item.Similar, recognize.Similar{ID: tx.ID, Date: tx.Date, Description: tx.Description})
	}

	return nil
}

func recognizeOutcome(result recognize.Result, err error) string {
	switch {
	case errors.Is(err, recognize.ErrDisabled), errors.Is(err, recognize.ErrUnavailable):
		return recognizeOutcomeUnavailable
	case err != nil:
		return recognizeOutcomeFailed
	case len(result.Items) == 0:
		return recognizeOutcomeEmpty
	default:
		return recognizeOutcomeOK
	}
}
