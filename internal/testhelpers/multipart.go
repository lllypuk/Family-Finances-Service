package testhelpers

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

// FilePart — часть multipart-тела: поле формы, имя файла и байты.
type FilePart struct {
	Field string
	Name  string
	Data  []byte
}

// PNGImage — валидный PNG заданного размера.
func PNGImage(t *testing.T, width, height int) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewGray(image.Rect(0, 0, width, height))))

	return buf.Bytes()
}

// JPEGImage — валидный JPEG заданного размера.
func JPEGImage(t *testing.T, width, height int) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, width, height)), nil))

	return buf.Bytes()
}

// MultipartBody собирает multipart/form-data и возвращает тело с его Content-Type.
func MultipartBody(t *testing.T, parts ...FilePart) ([]byte, string) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for _, p := range parts {
		fw, err := w.CreateFormFile(p.Field, p.Name)
		require.NoError(t, err)
		_, err = fw.Write(p.Data)
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())

	return buf.Bytes(), w.FormDataContentType()
}
