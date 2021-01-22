package util

import (
	"log"
	"os"
)

func CreateLogger(prefix, logfile string) *log.Logger{
	if logfile == "" {
		return log.New(os.Stdout, prefix, log.LstdFlags)
	} else {
		file, _ := os.Open(logfile)
		return log.New(file, prefix, log.LstdFlags)
	}
}
