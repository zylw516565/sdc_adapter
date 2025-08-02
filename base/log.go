package base

import (
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

const (
	TimestampFormat = "2006-01-02T15:04:05.000000Z08:00"
)
