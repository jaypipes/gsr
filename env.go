package util

import (
    "log"
	"os"
    "strconv"
)

// Returns the string value of the supplied environ variable or, if not
// present, the supplied default value
func EnvOrDefaultStr(key string, def string) string {
    val := os.Getenv(key)
    if val == "" {
        return def
    }
    return val
}

// Returns the int value of the supplied environ variable or, if not present,
// the supplied default value
func EnvOrDefaultInt(key string, def int) int {
    val := os.Getenv(key)
    if val == "" {
        return def
    }
    i, err := strconv.Atoi(val)
    if err != nil {
        log.Printf("Expected integer value for %v env variable, but got %v. Defaulting to %v", key, val, def)
        return def
    }
    return i
}

// Returns the boolvalue of the supplied environ variable or, if not present,
// the supplied default value
func EnvOrDefaultBool(key string, def bool) bool {
    val := os.Getenv(key)
    if val == "" {
        return def
    }
    b, err := strconv.ParseBool(val)
    if err != nil {
        log.Printf("Expected boolean value for %v env variable, but got %v. Defaulting to %v", key, val, def)
        return def
    }
    return b
}
