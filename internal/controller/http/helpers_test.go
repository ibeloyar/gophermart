package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func TestReadBody_TextPlain_String_Success(t *testing.T) {
	body := "test string"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	got, err := readBody[string](req)

	require.NoError(t, err)
	assert.Equal(t, body, got)
}

func TestReadBody_TextPlain_String_Empty(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "text/plain")

	got, err := readBody[string](req)

	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestReadBody_TextPlain_NonString_Fail(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("test"))
	req.Header.Set("Content-Type", "text/plain")

	type TestStruct struct{ Field string }

	_, err := readBody[TestStruct](req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read request body: text/plain")
}

func TestReadBody_JSON_Success(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}
	expected := TestStruct{Name: "test"}

	bodyJSON, _ := json.Marshal(expected)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	got, err := readBody[TestStruct](req)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestReadBody_JSON_Invalid_Fail(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"invalid": "json"`))
	req.Header.Set("Content-Type", "application/json")

	type TestStruct struct{ Name string }

	_, err := readBody[TestStruct](req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read request body application/json")
}

func TestReadBody_ReadError(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", errorReader{})
	req.Header.Set("Content-Type", "application/json")

	type TestStruct struct{ Name string }

	_, err := readBody[TestStruct](req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read request body")
}

func TestReadBody_NoContentType_JSON(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}
	expected := TestStruct{Name: "test"}

	bodyJSON, _ := json.Marshal(expected)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(bodyJSON))
	// НЕ устанавливаем Content-Type

	got, err := readBody[TestStruct](req)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestReadBody_TextPlainWithCharset(t *testing.T) {
	body := "test string"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	got, err := readBody[string](req)
	require.NoError(t, err)
	assert.Equal(t, body, got)
}

func TestWriteJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	writeJSON(w, data, http.StatusOK)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, data, got)
}

func TestWriteJSON_MarshalError(t *testing.T) {
	w := httptest.NewRecorder()
	data := make(chan int)

	writeJSON(w, data, http.StatusOK)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))

	assert.Contains(t, w.Body.String(), "Internal Server Error")
}
