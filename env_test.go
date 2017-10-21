package gsr

import (
	"os"
	"testing"
)

var (
	key = "TESTING"
)

func TestEnvOrDefaultStr(t *testing.T) {
	val := "value"
	defval := "default"
	os.Setenv(key, val)

	defer os.Unsetenv(key)

	res := EnvOrDefaultStr(key, defval)

	if res != val {
		t.Errorf(
			"Expected %v. Got %v.",
			val,
			res,
		)
	}

	os.Unsetenv(key)

	res = EnvOrDefaultStr(key, defval)

	if res != defval {
		t.Errorf(
			"Expected %v. Got %v.",
			defval,
			res,
		)
	}
}

func TestEnvOrDefaultInt(t *testing.T) {
	val := "42"
	badval := "meaning of life"
	intval := 42
	defval := 84
	os.Setenv(key, val)

	defer os.Unsetenv(key)

	res := EnvOrDefaultInt(key, defval)

	if res != intval {
		t.Errorf(
			"Expected %v. Got %v.",
			intval,
			res,
		)
	}

	os.Unsetenv(key)

	res = EnvOrDefaultInt(key, defval)

	if res != defval {
		t.Errorf(
			"Expected %v. Got %v.",
			defval,
			res,
		)
	}

	// Verify type conversion failure produces default value
	os.Setenv(key, badval)

	res = EnvOrDefaultInt(key, defval)

	if res != defval {
		t.Errorf(
			"Expected %v. Got %v.",
			defval,
			res,
		)
	}
}

func TestEnvOrDefaultBool(t *testing.T) {
	val := "true"
	badval := "meaning of life"
	boolval := true
	defval := false
	os.Setenv(key, val)

	defer os.Unsetenv(key)

	res := EnvOrDefaultBool(key, defval)

	if res != boolval {
		t.Errorf(
			"Expected %v. Got %v.",
			boolval,
			res,
		)
	}

	os.Unsetenv(key)

	res = EnvOrDefaultBool(key, defval)

	if res != defval {
		t.Errorf(
			"Expected %v. Got %v.",
			defval,
			res,
		)
	}

	// Verify type conversion failure produces default value
	os.Setenv(key, badval)

	res = EnvOrDefaultBool(key, defval)

	if res != defval {
		t.Errorf(
			"Expected %v. Got %v.",
			defval,
			res,
		)
	}
}
