package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/jaypipes/gsr/examples/pkg/util"
    "github.com/jaypipes/gsr"
)

const (
    myServiceName = "data"
)

var (
    myAddr = util.BindHost() + ":9000"
    reg *gsr.Registry
)

func main() {
    var err error
    info("Starting %s service on %s.", myServiceName, myAddr)

    info("Connecting to gsr.")
    reg, err = gsr.New()
    if err != nil {
        log.Fatalf("Failed to connect to gsr registry: %v", err)
    }

    info("Registering %s with gsr.", myAddr)
    ep := gsr.Endpoint{
        Service: &gsr.Service{Name: myServiceName},
        Address: myAddr,
    }
    err = reg.Register(&ep)
    if err != nil {
        log.Fatalf("Failed to register with gsr: %v", err)
    }

    info("Listening for HTTP traffic on %s.", myAddr)
    http.HandleFunc("/", handleHttp)
    log.Fatal(http.ListenAndServe(myAddr, nil))
}

func handleHttp(w http.ResponseWriter, r *http.Request) {
    eps := reg.Endpoints("")
    out := make([]string, len(eps))
    for x, ep := range(eps) {
        out[x] = ep.Service.Name + ":" + ep.Address
    }
    fmt.Fprintf(w, "%s service knows about endpoints: %v\n", myServiceName, out)
}

func info(message string, args ...interface{}) {
    header := fmt.Sprintf("[%s:%s] ", myServiceName, myAddr)
    log.Printf(header + message, args...)
}

