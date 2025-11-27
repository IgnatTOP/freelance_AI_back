package logger

import (
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// Init инициализирует структурированный логгер.
func Init(level string) {
	Log = logrus.New()

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	Log.SetLevel(lvl)

	// Используем JSON формат для production, text для development
	Log.SetFormatter(&logrus.JSONFormatter{})
}

// SetTextFormatter устанавливает текстовый формат логов (для development).
func SetTextFormatter() {
	if Log != nil {
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}
