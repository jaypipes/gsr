package gsr

import (
    "log"
)

func debug(message string, args ...interface{}) {
    if logLevel() > 1 {
        log.Printf("[gsr] debug: " + message, args...)
    }
}

func info(message string, args ...interface{}) {
    if logLevel() > 0 {
        log.Printf("[gsr] " + message, args...)
    }
}
