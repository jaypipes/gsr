package gsr

import (
    "os"
    "testing"
)

func TestNewRegistryBadAddress(t *testing.T) {
    // Set the environs variables to a known bad address and ensure that NewRegistry()
    // returns an error and a nil Registry pointer.
    orig, found := os.LookupEnv("GSR_ETCD_ENDPOINTS")
    if ! found {
        defer os.Unsetenv("GSR_ETCD_ENDPOINTS")
    } else {
        defer os.Setenv("GSR_ETCD_ENDPOINTS", orig)
    }

    orig, found = os.LookupEnv("GSR_ETCD_CONNECT_TIMEOUT_SECONDS")
    if ! found {
        defer os.Unsetenv("GSR_ETCD_CONNECT_TIMEOUT_SECONDS")
    } else {
        defer os.Setenv("GSR_ETCD_CONNECT_TIMEOUT_SECONDS", orig)
    }

    os.Setenv("GSR_ETCD_ENDPOINTS", "badaddress!")
    os.Setenv("GSR_ETCD_CONNECT_TIMEOUT_SECONDS", "1")

    r, err := NewRegistry()
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
    addr := "192.168.1.12"
    ep := Endpoint{
        Service: &Service{Name: service},
        Address: addr,
    }

    r, err := NewRegistry()
    if err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if r == nil {
        t.Fatal("Expected *gsr.Registry, but got nil.")
    }

    if err = r.Register(&ep); err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }

    eps := r.Endpoints(service)
    if eps == nil {
        t.Fatalf("Expected []string, but got nil")
    }
    if ! contains(addr, eps) {
        t.Fatalf("Expected to find %s in %v.", addr, eps)
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
    addr1 := "192.168.1.12"
    addr2 := "192.168.1.13"
    addrs := []string{addr1, addr2}
    ep1 := Endpoint{
        Service: &Service{Name: service},
        Address: addr1,
    }
    ep2 := Endpoint{
        Service: &Service{Name: service},
        Address: addr2,
    }

    r1, err := NewRegistry()
    if err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if r1 == nil {
        t.Fatal("Expected *gsr.Registry, but got nil.")
    }

    if err = r1.Register(&ep1); err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }

    if eps := r1.Endpoints(service); ! contains(addr1, eps)  {
        t.Fatalf("Expected to find %s in %v.", addr1, eps)
    }

    r2, err := NewRegistry()
    if err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if r1 == nil {
        t.Fatal("Expected *gsr.Registry, but got nil.")
    }

    if err = r2.Register(&ep2); err != nil {
        t.Fatalf("Expected nil, but got %v.", err)
    }
    if eps := r2.Endpoints(service); ! containsAll(addrs, eps)  {
        t.Fatalf("Expected to find %s in %v.", addrs, eps)
    }

    // first registry should find both endpoints now too
    if eps := r1.Endpoints(service); ! containsAll(addrs, eps)  {
        t.Fatalf("Expected to find %s in %v.", addrs, eps)
    }
}
