package gsr

import (
    "os"
    "testing"
)

func TestStartBadAddress(t *testing.T) {
    // Set the environs variables to a known bad address and ensure that Start()
    // returns an error and a nil Registry pointer.
    orig, found := os.LookupEnv("GSR_ETCD_ENDPOINTS")
    if ! found {
        defer os.Unsetenv("GSR_ETCD_ENDPOINTS")
    } else {
        defer os.Setenv("GSR_ETCD_ENDPOINTS", orig)
    }

    os.Setenv("GSR_ETCD_ENDPOINTS", "badaddress!")

    r, err := Start("service", "endpoint")
    if err == nil {
        t.Fatal("Expected error, but got nil.")
    }
    if r != nil {
        t.Fatalf("Expected nil, but got %v.", r)
    }
}

func TestFunctionalSimple(t *testing.T) {
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

    service := "web"
    ep1 := "192.168.1.12"

    r, err := Start(service, ep1)
    if err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if r == nil {
        t.Fatal("Expected *gsr.Registry, but got nil.")
    }

    eps := r.Endpoints(service)
    if eps == nil {
        t.Fatalf("Expected []string, but got nil")
    }
    found = false
    for _, ep := range(eps) {
        if ep == ep1 {
            found = true
            break
        }
    }
    if ! found {
        t.Fatalf("Expected to find %s in %v.", ep1, eps)
    }
}

func TestFunctionalConcurrency(t *testing.T) {
    testEps, found := os.LookupEnv("GSR_TEST_ETCD_ENDPOINTS")
    if ! found {
        t.Skip("GSRT_TEST_ETCD_ENDPOINTS not set. Skipping functional " +
               "concurrency test.")
    }

    orig, found := os.LookupEnv("GSR_ETCD_ENDPOINTS")
    if ! found {
        defer os.Unsetenv("GSR_ETCD_ENDPOINTS")
    } else {
        defer os.Setenv("GSR_ETCD_ENDPOINTS", orig)
    }

    os.Setenv("GSR_ETCD_ENDPOINTS", testEps)

    service := "data-access"
    ep1 := "192.168.1.12"
    ep2 := "192.168.1.13"

    r1, err := Start(service, ep1)
    if err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if r1 == nil {
        t.Fatal("Expected *gsr.Registry, but got nil.")
    }

    eps1 := r1.Endpoints(service)
    if eps1 == nil {
        t.Fatalf("Expected []string, but got nil")
    }
    if ! contains(t, ep1, eps1) {
        t.Fatalf("Expected to find %s in %v.", ep1, eps1)
    }

    r2, err := Start(service, ep2)
    if err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if r2 == nil {
        t.Fatal("Expected *gsr.Registry, but got nil.")
    }

    eps2 := r2.Endpoints(service)
    if eps2 == nil {
        t.Fatalf("Expected []string, but got nil")
    }
    if ! contains(t, ep2, eps2) {
        t.Fatalf("Expected to find %s in %v.", ep2, eps2)
    }
    // first endpoint should have been read into second service registry on
    // load()
    if ! contains(t, ep1, eps2) {
        t.Fatalf("Expected to find %s in %v.", ep1, eps2)
    }
    // second service registry should have picked up the second registered
    // service
    if ! contains(t, ep2, eps1) {
        t.Fatalf("Expected to find %s in %v.", ep2, eps1)
    }
}

func contains(t *testing.T, search string, in []string) bool {
    for _, s := range(in) {
        if s == search {
            return true
        }
    }
    return false
}

func containsAll(t *testing.T, all []string, in []string) bool {
    for _, each := range(all) {
        if ! contains(t, each, in) {
            return false
        }
    }
    return true
}
