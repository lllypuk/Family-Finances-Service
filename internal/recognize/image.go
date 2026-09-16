package recognize

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

const (
	MIMEPNG  = "image/png"
	MIMEJPEG = "image/jpeg"
)

// CheckImage определяет MIME по сигнатуре и проверяет размеры по заголовку; целостность данных не проверяется.
func CheckImage(data []byte) (string, error) {
	if len(data) > MaxImageBytes {
		return "", ErrImageTooLarge
	}

	var (
		mime   string
		decode func(io.Reader) (image.Config, error)
	)

	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		mime, decode = MIMEPNG, png.DecodeConfig
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		mime, decode = MIMEJPEG, jpeg.DecodeConfig
	default:
		return "", ErrNotImage
	}

	cfg, err := decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotImage, err)
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return "", ErrNotImage
	}

	if int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return "", ErrImageTooLarge
	}

	return mime, nil
}
