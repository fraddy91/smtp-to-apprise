package logger

import (
	"log"
	"os"
)

var debugEnabled = false

func Init(enableDebug bool) {
	debugEnabled = enableDebug
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

func Debugf(format string, v ...any) {
	if debugEnabled {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func Infof(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func Warnf(format string, v ...any) {
	log.Printf("[WARN] "+format, v...)
}

func Errorf(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}
