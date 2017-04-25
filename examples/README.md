# Examples of using `gsr`

We include a set of simple example applications that show how to make use of
the `gsr` library. There are two Golang applications, "web" and "data", located
in the [examples/cmd/](cmd/) directory.

Each application connects to `gsr`, registers itself with `gsr` and provides a
simple HTTP server that responds to all requests by printing out the list of
all service endpoints that the application knows about (through `gsr`).

## Step 1: Specify the `etcd3` cluster for `gsr` to use

`gsr` uses an `etcd3` cluster for storing service and endpoint information, so
the first step in running the examples is setting the `GSR_ETCD_ENDPOINTS`
environment variable.

If you don't have an `etcd3` cluster to use for the examples, you can easily
create one by calling the [etcd3-container.bash](etcd3-container.bash) script
located in this directory. This script uses `rkt` and `systemd-nspawn` to
launch an etcd3 Application Container Image in a temporary container namespace
and reports the host address the etcd3 service is exposed from:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ ./examples/etcd3-container.bash 
Checking and installing ACI keys for etcd ... ok.
Starting etcd3 rkt pod ... ok.
Determining etcd3 endpoint address ... ok.
etcd running in container at 172.16.28.68:2379.
```

### Handy `rkt` commands

**NOTE**: If you want to check out the status of your etcd3 container, you can
run `sudo rkt list`:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ sudo rkt list
UUID        APP IMAGE NAME      STATE   CREATED     STARTED     NETWORKS
eaf1dde8    etcd    coreos.com/etcd:v3.0.6  running 16 seconds ago  16 seconds ago  default:ip4=172.16.28.68
```

Want to stop your etcd3 container?

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ sudo rkt stop eaf1dde8
"eaf1dde8-7715-48d0-a43f-883f1f12c351"
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ sudo rkt list
UUID        APP IMAGE NAME      STATE   CREATED     STARTED     NETWORKS
eaf1dde8    etcd    coreos.com/etcd:v3.0.6  exited  1 minute ago    1 minute ago    
```

Want to clean up old shut-down containers?

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ sudo rkt gc --grace-period=0
gc: moving pod "eaf1dde8-7715-48d0-a43f-883f1f12c351" to garbage
Garbage collecting pod "eaf1dde8-7715-48d0-a43f-883f1f12c351"
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ sudo rkt list
UUID    APP IMAGE NAME  STATE   CREATED STARTED NETWORKS
```

## Step 2: Verify there's an empty `gsr` service registry

Because `gsr` uses `etcd3` under the covers to read and write service endpoint
information, we can use the `etcdctl` CLI tool to look at the `gsr` registry
information. Because `gsr` uses version 3 of `etcd`, don't forget to include
the `ETCDCTL_API=3` environment setting. You can see the list of service
endpoints that `gsr` knows about using a key prefix query with `etcdctl key
--prefix`.

Here, we're checking that there are no service endpoints registered in `gsr` to
start with:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ ETCDCTL_API=3 etcdctl --endpoints="http://172.16.28.68:2379"  get --prefix gsr/services
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ 
```

We will issue the same command later when verifying that `gsr` has registered
endpoints upon our applications starting up.

## Step 3: Launch web application

In a new terminal window, launch the example web application, specifying the
`GSR_ETCD_ENDPOINTS` value as below to match your `etcd3` cluster address:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ PS1="web$ "
web$ pwd
/home/jaypipes/src/github.com/jaypipes/gsr
web$ GSR_ETCD_ENDPOINTS="http://172.16.28.68:2379" go run examples/cmd/web/main.go 
2017/04/25 12:35:50 [web:192.168.1.7:8080] Starting web service on 192.168.1.7:8080.
2017/04/25 12:35:50 [web:192.168.1.7:8080] Connecting to gsr.
2017/04/25 12:35:50 [web:192.168.1.7:8080] Registering 192.168.1.7:8080 with gsr.
2017/04/25 12:35:50 [web:192.168.1.7:8080] Listening for HTTP traffic on 192.168.1.7:8080.
```

## Step 4: Verify `gsr` registered the web application endpoint

Use `etcdctl` to verify there's been a new endpoint added to the `gsr` registry:

```
jpipes@uberbox:~/src/github.com/jaypipes/gsr$ ETCDCTL_API=3 etcdctl --endpoints="http://172.16.28.68:2379"  get --prefix gsr/services
gsr/services/web/192.168.1.7:8080

```

You can even ask the new web application what endpoints it knows about by
hitting the bind address of the web application (in the example from above, the
bind address is `192.168.1.7:8080`:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ curl http://192.168.1.7:8080/
web service knows about endpoints: [web:192.168.1.7:8080]
```

## Step 5: Launch data application

Next, in another terminal, launch the data application:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ PS1="data$ "
data$ pwd
/home/jaypipes/src/github.com/jaypipes/gsr
data$ GSR_ETCD_ENDPOINTS="http://172.16.28.68:2379" go run examples/cmd/data/main.go 
2017/04/25 12:37:19 [data:192.168.1.7:9000] Starting data service on 192.168.1.7:9000.
2017/04/25 12:37:19 [data:192.168.1.7:9000] Connecting to gsr.
2017/04/25 12:37:19 [data:192.168.1.7:9000] Registering 192.168.1.7:9000 with gsr.
2017/04/25 12:37:19 [data:192.168.1.7:9000] Listening for HTTP traffic on 192.168.1.7:9000.
```

## Step 6: Verify `gsr` registered the data application endpoint

Again, using `etcdctl`, verify the new endpoint was registered in `gsr`:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ ETCDCTL_API=3 etcdctl --endpoints="http://172.16.28.68:2379"  get --prefix gsr/services
gsr/services/data/192.168.1.7:9000

gsr/services/web/192.168.1.7:8080

```

Excellent. Both endpoints appear in the `gsr` registry.

## Step 7: Verify each application sees the other application registered with `gsr`

You can hit the bind address of both the "web" and "data" applications to see
what endpoints each application is aware of. **Note that we didn't restart either
application**. `gsr` took care of notifying each application of the other
application registering with `gsr`.

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ curl http://192.168.1.7:8080/
web service knows about endpoints: [data:192.168.1.7:9000 web:192.168.1.7:8080]
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ curl http://192.168.1.7:9000/
data service knows about endpoints: [data:192.168.1.7:9000 web:192.168.1.7:8080]
```
