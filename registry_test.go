package gsr

import (
    "os"
    "testing"
)

func TestNewBadAddress(t *testing.T) {
    // Set the environs variables to a known bad address and ensure that New()
    // returns an error and a nil Registry pointer.
    orig, found := os.LookupEnv("GSR_ETCD_ENDPOINTS")
    if ! found {
        defer os.Unsetenv("GSR_ETCD_ENDPOINTS")
    } else {
        defer os.Setenv("GSR_ETCD_ENDPOINTS", orig)
    }

    os.Setenv("GSR_ETCD_ENDPOINTS", "badaddress!")

    r, err := New()
    if err == nil {
        t.Error("Expected error, but got nil.")
    }
    if r != nil {
        t.Error("Expected nil, but got %v.", r)
    }
}

func TestFunctionalNewSuccess(t *testing.T) {
    testEps, found := os.LookupEnv("GSR_TEST_ETCD_ENDPOINTS")
    if ! found {
        t.Skip("GSRT_TEST_ETCD_ENDPOINTS not set. Skipping functional test.")
    }

    orig, found := os.LookupEnv("GSR_ETCD_ENDPOINTS")
    if ! found {
        defer os.Unsetenv("GSR_ETCD_ENDPOINTS")
    } else {
        defer os.Setenv("GSR_ETCD_ENDPOINTS", orig)
    }

    os.Setenv("GSR_ETCD_ENDPOINTS", testEps)

    r, err := New()
    if err != nil {
        t.Error("Expected nil, but got %v.", err)
    }
    if r == nil {
        t.Error("Expected *gsr.Registry, but got nil.")
    }
}
