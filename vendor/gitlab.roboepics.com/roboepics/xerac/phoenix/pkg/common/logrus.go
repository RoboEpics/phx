package common

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func levelMapping(lvl string) logrus.Level {
	switch strings.ToUpper(lvl) {
	case "TRACE", "vvv", "3":
		return logrus.TraceLevel
	case "DEBUG", "vv", "2":
		return logrus.DebugLevel
	case "INFO", "v", "1":
		return logrus.InfoLevel
	case "WARN", "WARNING":
		return logrus.WarnLevel
	case "ERROR", "ERR":
		return logrus.ErrorLevel
	case "FATAL":
		return logrus.FatalLevel
	case "PANIC":
		return logrus.PanicLevel
	default:
		return logrus.WarnLevel
	}
}

func formatterMapping(formater string) logrus.Formatter {
	switch strings.ToUpper(formater) {
	case "JSON":
		return &logrus.JSONFormatter{}
	case "TEXT", "txt", "":
		return &logrus.TextFormatter{}
	default:
		return &logrus.TextFormatter{}
	}
}

// This function uses viper values. so call this from your
// PresistentPreRun cobra hook.
func SetupLogrusWithViper() {
	var (
		level = levelMapping(
			viper.GetString("log.level"))
		formatter = formatterMapping(
			viper.GetString("log.formatter"))
	)
	logrus.SetLevel(level)
	logrus.SetFormatter(formatter)
}
