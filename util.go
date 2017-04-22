package gsr

import (
    "log"
)

func contains(search string, in []*Endpoint) bool {
    for _, s := range(in) {
        if s.Address == search {
            return true
        }
    }
    return false
}

func containsAll(all []string, in []*Endpoint) bool {
    for _, each := range(all) {
        if ! contains(each, in) {
            return false
        }
    }
    return true
}

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
