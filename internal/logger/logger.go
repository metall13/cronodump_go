package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New(level string) *Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	switch strings.ToLower(level) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	return &Logger{logger}
}

func (l *Logger) LogProgress(jobID string, message string) {
	l.WithFields(logrus.Fields{
		"job_id": jobID,
		"type":   "progress",
	}).Info(message)
}

func (l *Logger) LogError(jobID string, err error) {
	l.WithFields(logrus.Fields{
		"job_id": jobID,
		"type":   "error",
	}).Error(err)
}