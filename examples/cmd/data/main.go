package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/jaypipes/gsr"
)

const (
	myServiceName = "data"
)

var (
	myAddr = bindHost() + ":9000"
	reg    *gsr.Registry
)

func main() {
	var err error
	info("starting %s service on %s.", myServiceName, myAddr)

	printGsrEnvVars()

	info("connecting to gsr.")
	reg, err = gsr.New()
	if err != nil {
		log.Fatalf("failed to connect to gsr registry: %v", err)
	}

	eps := getEndpoints()
	info(
		"before registering itself, %s service knows about endpoints: %v\n",
		myServiceName,
		eps,
	)

	info("registering %s with gsr.", myAddr)
	ep := gsr.Endpoint{
		Service: &gsr.Service{Name: myServiceName},
		Address: myAddr,
	}
	err = reg.Register(&ep)
	if err != nil {
		log.Fatalf("failed to register with gsr: %v", err)
	}

	info("listening for HTTP traffic on %s.", myAddr)
	http.HandleFunc("/", handleHttp)
	log.Fatal(http.ListenAndServe(myAddr, nil))
}

func getEndpoints() []string {
	eps := reg.Endpoints("")
	out := make([]string, len(eps))
	for x, ep := range eps {
		out[x] = ep.Service.Name + ":" + ep.Address
	}
	return out
}

func handleHttp(w http.ResponseWriter, r *http.Request) {
	eps := getEndpoints()
	fmt.Fprintf(w, "%s service knows about endpoints: %v\n", myServiceName, eps)
}

func printGsrEnvVars() {
	for _, keyval := range os.Environ() {
		if strings.HasPrefix(keyval, "GSR_") {
			parts := strings.Split(keyval, "=")
			key := parts[0]
			val := parts[1]
			info("%s is set to %s", key, val)
		}
	}
}

func info(message string, args ...interface{}) {
	header := fmt.Sprintf("[%s:%s] ", myServiceName, myAddr)
	log.Printf(header+message, args...)
}

func bindHost() string {
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	addr := c.LocalAddr().String()
	return addr[:strings.LastIndex(addr, ":")]
}
