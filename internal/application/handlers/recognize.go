package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/recognize"
	"family-budget-service/internal/services"
)

const (
	// RecognizeUploadTimeout — срок приёма тела POST /transactions/recognize.
	RecognizeUploadTimeout = 60 * time.Second
	// RecognizeBodyLimit — 5 × 2 МиБ и оболочка multipart; тот же предел у Caddy.
	RecognizeBodyLimit = "11M"

	// recognizeWriteSlack — запас дедлайна записи сверх загрузки и бюджета движка.
	recognizeWriteSlack = 15 * time.Second
	// recognizeEngineSlack — запас контекста сверх Budget(): отказ движка приходит раньше отмены.
	recognizeEngineSlack = 5 * time.Second

	fieldImages = "images"
)

// RecognizeHandler — распознавание скриншотов банка в кандидатов операций.
type RecognizeHandler struct {
	service       services.RecognizeService
	uploadTimeout time.Duration
}

// NewRecognizeHandler собирает handler; uploadTimeout — срок приёма тела (RecognizeUploadTimeout).
func NewRecognizeHandler(service services.RecognizeService, uploadTimeout time.Duration) *RecognizeHandler {
	return &RecognizeHandler{service: service, uploadTimeout: uploadTimeout}
}

// Recognize принимает multipart с частями images и отдаёт кандидатов; вызов платный и не идемпотентный.
func (h *RecognizeHandler) Recognize(c echo.Context) error {
	if err := h.extendDeadlines(c); err != nil {
		return err
	}

	images, err := h.readImages(c)
	if err != nil {
		return ignoreWritten(err)
	}

	// Новый контекст от запроса, а не от фазы загрузки: срок загрузки к модели не относится.
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.service.Budget()+recognizeEngineSlack)
	defer cancel()

	result, err := h.service.Recognize(ctx, images)
	if err != nil {
		return respondRecognizeError(c, err)
	}

	if result.Items == nil {
		result.Items = []recognize.Item{}
	}

	return respondAPI(c, http.StatusOK, result)
}

// extendDeadlines раздвигает SERVER_READ/WRITE_TIMEOUT под загрузку и вызов модели.
// ResponseRecorder дедлайнов не умеет — http.ErrNotSupported не ошибка.
func (h *RecognizeHandler) extendDeadlines(c echo.Context) error {
	now := time.Now()
	rc := http.NewResponseController(c.Response())

	write := now.Add(h.uploadTimeout + h.service.Budget() + recognizeWriteSlack)
	if err := rc.SetWriteDeadline(write); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("set write deadline: %w", err)
	}

	if err := rc.SetReadDeadline(now.Add(h.uploadTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("set read deadline: %w", err)
	}

	return nil
}

// readImages читает части потоком, не дальше MaxImageBytes+1 на часть. Отказ уже записан —
// возвращается errResponseAlreadyWritten.
func (h *RecognizeHandler) readImages(c echo.Context) ([]recognize.Image, error) {
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.uploadTimeout)
	defer cancel()

	reader, err := c.Request().MultipartReader()
	if err != nil {
		return nil, writeBodyReadError(c, err)
	}

	var images []recognize.Image

	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}

		if nextErr != nil {
			return nil, writeBodyReadError(c, nextErr)
		}

		field := fmt.Sprintf("%s[%d]", fieldImages, len(images))

		if name := part.FormName(); name != fieldImages {
			if name == "" {
				name = fieldBody
			}

			return nil, writeImageError(c, name, "unexpected part; only images are accepted")
		}

		if len(images) == recognize.MaxImages {
			return nil, writeImageError(c, field, "at most "+strconv.Itoa(recognize.MaxImages)+" images are accepted")
		}

		data, readErr := io.ReadAll(io.LimitReader(ctxReader{ctx: ctx, r: part}, recognize.MaxImageBytes+1))
		if readErr != nil {
			return nil, writeBodyReadError(c, readErr)
		}

		mime, checkErr := recognize.CheckImage(data)
		if checkErr != nil {
			return nil, writeImageError(c, field, imageErrorMessage(checkErr))
		}

		images = append(images, recognize.Image{MIME: mime, Data: data})
	}

	if len(images) == 0 {
		return nil, writeImageError(c, fieldImages, "is required")
	}

	return images, nil
}

// ctxReader отдаёт ошибку контекста до чтения; заблокированное чтение прерывает дедлайн соединения.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (r ctxReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.r.Read(p)
}

func imageErrorMessage(err error) string {
	if errors.Is(err, recognize.ErrImageTooLarge) {
		return "image is larger than " + strconv.Itoa(recognize.MaxImageBytes) + " bytes or " +
			strconv.Itoa(recognize.MaxPixels) + " pixels"
	}

	return "must be a PNG or JPEG image"
}

func writeImageError(c echo.Context, field, message string) error {
	if err := respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: field, Message: message, Code: ErrCodeValidationError}); err != nil {
		return err
	}

	return errResponseAlreadyWritten
}

// writeBodyReadError: лимитер BodyLimit сквозь multipart-парсер — 413, истёкший срок загрузки — 408.
func writeBodyReadError(c echo.Context, err error) error {
	var (
		he      *echo.HTTPError
		written error
	)

	switch {
	case errors.As(err, &he) && he.Code == http.StatusRequestEntityTooLarge:
		written = respondError(c, http.StatusRequestEntityTooLarge, ErrCodePayloadTooLarge, ErrMessagePayloadTooLarge)
	case errors.Is(err, os.ErrDeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		written = respondError(c, http.StatusRequestTimeout, ErrCodeRequestTimeout, ErrMessageRequestTimeout)
	default:
		written = respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, err.Error()))
	}

	if written != nil {
		return written
	}

	return errResponseAlreadyWritten
}

// respondRecognizeError: плечо выключено или модель недоступна — 503, ответ не разобрался — 502.
// Прочее уходит в error handler — 500 с текстом в логе.
func respondRecognizeError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, recognize.ErrDisabled), errors.Is(err, recognize.ErrUnavailable):
		var unavailable *recognize.UnavailableError
		if errors.As(err, &unavailable) && unavailable.RetryAfter > 0 {
			seconds := (unavailable.RetryAfter + time.Second - 1) / time.Second
			c.Response().Header().Set(echo.HeaderRetryAfter, strconv.FormatInt(int64(seconds), 10))
		}

		return respondError(c, http.StatusServiceUnavailable,
			ErrCodeRecognitionUnavailable, ErrMessageRecognitionUnavailable)
	case errors.Is(err, recognize.ErrBadAnswer):
		return respondError(c, http.StatusBadGateway, ErrCodeRecognitionFailed, ErrMessageRecognitionFailed)
	default:
		return fmt.Errorf("recognize transactions: %w", err)
	}
}
