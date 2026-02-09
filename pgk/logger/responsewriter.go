package logger

import "net/http"

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b) // записываем ответ, используя оригинальный http.ResponseWriter
	r.responseData.size += size            // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode) // записываем код статуса, используя оригинальный http.ResponseWriter
	r.responseData.status = statusCode       // захватываем код статуса
}
