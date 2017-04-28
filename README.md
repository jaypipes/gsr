# `gsr` - The Golang Service Registry library

`gsr` is a Golang library that can be used to provide service registration
and discovery capabilities to Golang applications. `gsr`'s library interfaces
are simple to use, simple to reason about, and importantly, do not require a
particular container runtime.

## Overview

`gsr` uses `etcd` for storing its service registry. Within the `etcd` store,
`gsr` sets up a series of keys representing services that have registered with
`gsr`. The structure of the `gsr` registry in `etcd` looks like so:

```
$KEY_PREFIX <-- environ['GSR_KEY_PREFIX']
|
-> /services
   |
   -> /$SERVICE
       |
       -> /$ENDPOINT1
       -> /$ENDPOINT2
```

As an example, let's imagine you have a distributed system deployment composed
of three different service applications:

* `web`
* `data-access`
* `background`

Each of the above applications is a Golang application that is built to run
within a container, a VM, on baremetal, whatever. Assume for now that you only
deploy a single instance of each of the above service applications, and they
end up running at the following addresses:

* `web`: 172.16.28.23:80
* `data-access`: 172.16.28.24:10000
* `background`: 172.16.28.25:10000

Assuming that your `GSR_KEY_PREFIX` environment variable is "gsr", the `gsr`
registry in `etcd` would look like this:

```
/gsr
|
-> /services
   |
   -> /web
       |
       -> 127.16.28.23:80

   -> /data-access
       |
       -> 127.16.28.24:10000

   -> /background
       |
       -> 127.16.28.25:10000
```

## Usage

`gsr` can be used for both service discovery and service registration.

### Service discovery

If you have a Golang application that needs to look up information about a
required service that the application makes use of, you need *service
discovery* capabilities.

The `gsr.Registry` struct can be used to discover information about services
and their endpoints, as this example shows:

```golang
package main

import (
    "log"

    "github.com/jaypipes/gsr"

    "github.com/exampleorg/data"
)

func main() {
    // Creates a new gsr.Registry instance that is connected to the gsr etcd
    // registry
    sr, err := gsr.NewRegistry()
    if err != nil {
         log.Fatalf("Failed to connect to gsr: %v.", err)
    }

    var dbConn *data.Connection

    // Our application depends on a service called "data-access", so let's find
    // the endpoints the service has so that we can query for data
    dataEps := sr.Endpoints("data-access")
    for _, ep := range(dataEps) {
	// Try connecting to the data access service. This code is an example.
	// Your service access code might look very different...
        if dbConn, err := data.connect(ep.Address); err != nil {
            log.Printf("Failed to connect to data access service: %v", err)
        }
    }
}
```

This strategy allows you to forego injecting service and endpoint configuration
into environment variables of configuration files.

### Service registration

If you have a service application written in Golang, upon startup, you want the
service to register itself with some system in order to allow other services to
discover it. What you need is *service registration* functionality.

The `gsr.Registry` struct can be used to register a service endpoint, as this
example shows:

```golang
package main

import (
    "log"
    "net"
    "strings"

    "github.com/jaypipes/gsr"
)

func main() {
    // Creates a new gsr.Registry instance that is connected to the gsr etcd
    // registry
    sr, err := gsr.NewRegistry()
    if err != nil {
         log.Fatalf("Failed to connect to gsr: %v.", err)
    }

    ep := gsr.Endpoint{
        Service: &gsr.Service{"my-whizzbang-service"},
        Address: bindHost(),
    }

    err := gsr.Register(&ep)
    if err != nil {
        log.Fatalf("unable to register %v with gsr: %v", svc, err)
    }
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
```

**Need more example code?**

If you need more example code, please check out the
[README.md](examples/README.md) in the [examples/](examples/) directory.

## Configuring

In the spirit of 12-factor apps, `gsr` can be configured entirely by setting
environment variables. Here is a list of environment variables that influence
`gsr`'s behaviour:

* `GSR_ETCD_ENDPOINTS`: a comma-separated list of `etcd` endpoints `gsr` will
   look for an `etcd` K/V store. (default: `http://127.0.0.1:2379`)

* `GSR_KEY_PREFIX`: a string indicating the prefix of keys in `etcd` where `gsr`
   will store service and endpoint information. (default: `''`)

* `GSR_ETCD_CONNECT_TIMEOUT_SECONDS`: the number of seconds to attempt to
  connect to the `etcd` when calling `gsr.NewRegistry()`. (default: `300`)

* `GSR_ETCD_DIAL_TIMEOUT_SECONDS`: the number of seconds to use as the dial
  timeout when attempting to connect to an `etcd` endpoint. (default: `1`)

* `GSR_ETCD_REQUEST_TIMEOUT_SECONDS`: the number of seconds to set each `etcd`
  request timeout to, once connected. (default: `1`)

* `GSR_USE_TLS`: 0 (default) if communication with etcd should be secured with
  TLS.

* `GSR_TLS_CERT_PATH`: Path to the certificate file to use if TLS is used.
  (default: "/etc/gsr/server.pem")

* `GSR_TLS_KEY_PATH`: Path to the private key file to use if TLS is used.
  (default: "/etc/gsr/server.key")

* `GSR_LOG_LEVEL`: an integer representing the verbosity of logging. The higher
  the number, the more verbose. (default: `0` almost no output during normal
  operation)

* `GSR_LEASE_SECONDS`: an integer representing the number of seconds gsr should
  use when writing endpoint information into the registry. (default: `60`)
