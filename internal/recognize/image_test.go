package recognize_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/recognize"
)

func smallImage() image.Image {
	return image.NewRGBA(image.Rect(0, 0, 8, 8))
}

// pngHeader — сигнатура и IHDR с заданными размерами: DecodeConfig дальше не читает.
func pngHeader(width, height uint32) []byte {
	ihdr := make([]byte, 0, 17)
	ihdr = append(ihdr, "IHDR"...)
	ihdr = binary.BigEndian.AppendUint32(ihdr, width)
	ihdr = binary.BigEndian.AppendUint32(ihdr, height)
	ihdr = append(ihdr, 8, 6, 0, 0, 0)

	var buf bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")
	buf.Write(binary.BigEndian.AppendUint32(nil, 13))
	buf.Write(ihdr)
	buf.Write(binary.BigEndian.AppendUint32(nil, crc32.ChecksumIEEE(ihdr)))

	return buf.Bytes()
}

func TestCheckImage_PNG(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, smallImage()))

	mime, err := recognize.CheckImage(buf.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "image/png", mime)
}

func TestCheckImage_JPEG(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, smallImage(), nil))

	mime, err := recognize.CheckImage(buf.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", mime)
}

func TestCheckImage_GIFRejected(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, gif.Encode(&buf, smallImage(), nil))

	_, err := recognize.CheckImage(buf.Bytes())
	require.ErrorIs(t, err, recognize.ErrNotImage)
}

func TestCheckImage_HeaderOnlyIsEnough(t *testing.T) {
	mime, err := recognize.CheckImage(pngHeader(1080, 2400))
	require.NoError(t, err, "целостность не обещается — хватает заголовка")
	assert.Equal(t, "image/png", mime)
}

func TestCheckImage_TooManyPixels(t *testing.T) {
	_, err := recognize.CheckImage(pngHeader(5000, 4000))
	require.ErrorIs(t, err, recognize.ErrImageTooLarge)
}

func TestCheckImage_TruncatedHeader(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, smallImage()))

	_, err := recognize.CheckImage(buf.Bytes()[:20])
	require.ErrorIs(t, err, recognize.ErrNotImage)

	_, err = recognize.CheckImage([]byte{0xFF, 0xD8, 0xFF})
	require.ErrorIs(t, err, recognize.ErrNotImage)
}

func TestCheckImage_TooManyBytes(t *testing.T) {
	data := append(pngHeader(10, 10), make([]byte, recognize.MaxImageBytes)...)

	_, err := recognize.CheckImage(data)
	require.ErrorIs(t, err, recognize.ErrImageTooLarge)
}

func TestCheckImage_Empty(t *testing.T) {
	_, err := recognize.CheckImage(nil)
	require.ErrorIs(t, err, recognize.ErrNotImage)
}

func TestUnavailableError_Is(t *testing.T) {
	cause := errors.New("connection refused")
	err := error(&recognize.UnavailableError{RetryAfter: 10 * time.Second, Err: cause})

	require.ErrorIs(t, err, recognize.ErrUnavailable)
	require.ErrorIs(t, err, cause)

	var unavailable *recognize.UnavailableError
	require.ErrorAs(t, err, &unavailable)
	assert.Equal(t, 10*time.Second, unavailable.RetryAfter)

	require.ErrorIs(t, &recognize.UnavailableError{}, recognize.ErrUnavailable)
}
