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
    if ! contains(ep1, eps) {
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
    ep1and2 := []string{ep1, ep2}

    r1, err := Start(service, ep1)
    if err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if r1 == nil {
        t.Fatal("Expected *gsr.Registry, but got nil.")
    }

    if eps := r1.Endpoints(service); ! contains(ep1, eps)  {
        t.Fatalf("Expected to find %s in %v.", ep1, eps)
    }

    r2, err := Start(service, ep2)
    if eps := r2.Endpoints(service); ! containsAll(ep1and2, eps)  {
        t.Fatalf("Expected to find %s in %v.", ep1and2, eps)
    }
    // first registry should find both endpoints now too
    if eps := r1.Endpoints(service); ! containsAll(ep1and2, eps)  {
        t.Fatalf("Expected to find %s in %v.", ep1and2, eps)
    }
}
