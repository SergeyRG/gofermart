package logging

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l, err := cfg.Build()
	if err != nil {
		return err
	}

	Logger = l

	return nil
}

func WithLogging(h http.Handler) http.Handler {
	f := func(wr http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		url := r.URL
		method := r.Method

		respData := &responseData{
			status: 0,
			length: 0,
		}

		lwr := &loggingRespWriter{
			ResponseWriter: wr,
			data:           respData,
		}

		h.ServeHTTP(lwr, r)

		duration := time.Since(startTime)

		Logger.Info("request has been processed",
			zap.String("URL", url.String()),
			zap.String("method", method),
			zap.Duration("time", duration),
			zap.Int("respones status", respData.status),
			zap.Int("respones lenght", respData.length),
		)

	}
	return http.HandlerFunc(f)
}

type loggingRespWriter struct {
	http.ResponseWriter
	data *responseData
}

func (rw *loggingRespWriter) Write(b []byte) (int, error) {
	length, err := rw.ResponseWriter.Write(b)
	rw.data.length += length
	return length, err
}

func (rw *loggingRespWriter) WriteHeader(statusCode int) {
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.data.status = statusCode
}

type responseData struct {
	status int
	length int
}
