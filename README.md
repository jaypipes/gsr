`gsr` - The Golang Service Registry library
=============================================

`gsr` is a Golang library that can be used to provide service registration
and discovery capabilities to Golang applications. `gsr`'s library interfaces
are simple to use, simple to reason about, and importantly, do not require a
particular container runtime.

Overview
--------

`gsr` uses `etcd` for storing its service registry. Within the `etcd` store,
`gsr` sets up a series of keys representing services that have registered with
`gsr`. The structure of the `gsr` registry in `etcd` looks like so:

```
$KEY_PREFIX <-- environ['GSR_KEY_PREFIX']
|
-> /services
   |
   -> /by-type
   |
   -> /$SERVICE_TYPE: JSON-serialized list of endpoints
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
gsr
|
-> /services
   |
   -> /by-type
      |
      -> web => ["127.16.28.23:80"]
      -> data-access => ["127.16.28.24:10000"]
      -> background -> ["127.16.28.25:10000"]
```


Configuring `gsr`
-----------------

In the spirit of 12-factor apps, `gsr` can be configured entirely by setting
environment variables. Here is a list of environment variables that influence
`gsr`'s behaviour:

* `GSR_ETCD_ENDPOINTS`: a comma-separated list of `etcd` endpoints `gsr` will
   look for an `etcd` K/V store. (default: `http://127.0.0.1:2379`)

* `GSR_KEY_PREFIX`: a string indicating the prefix of keys in `etcd` where `gsr`
   will store service and endpoint information. (default: `''`)

* `GSR_ETCD_CONNECT_TIMEOUT_SECONDS`: the number of seconds to attempt to
  connect to the `etcd` when calling `gsr.New()`. (default: `300`)

* `GSR_ETCD_REQUEST_TIMEOUT_SECONDS`: the number of seconds to set each `etcd`
  request timeout to, once connected. (default: `1`)

* `GSR_LOG_LEVEL`: an integer representing the verbosity of logging. The higher
  the number, the more verbose. (default: `0` almost no output during normal
  operation)
