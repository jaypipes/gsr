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
located in this directory. This script uses `docker` to launch an etcd3
Application Container Image in a temporary container namespace and reports the
host address the etcd3 service is exposed from:

```
[jaypipes@uberbox examples]$ ./etcd3-container.bash
Starting etcd container for gsr tests (data-dir: /tmp/gsr-example-cRQReU)... ok.
Determining etcd3 endpoint address ... ok.
etcd running in container at 172.17.0.2:2379.
```

### Handy `docker` commands

If you want to check out the status of your etcd3 container, you can run
`docker container inspetct gsr-example-etcd`, which outputs a JSON object of
lots of information about the container. You can pipe the result of that to the
`jq` program to slice and dice the contents any way you want.

**NOTE**: If you don't have it installed, install the `jq` program using your
typical operating system package manager.

Here's an example of getting the status and IP address of the container:

```
[jaypipes@uberbox examples]$ docker container inspect gsr-example-etcd \
 | jq '.[0] | {status: .State.Status, ip: .NetworkSettings.Networks.bridge.IPAddress}'
{
  "status": "running",
  "ip": "172.17.0.2"
}
```

Want to stop your etcd3 container? Use `docker container kill`:

```
[jaypipes@uberbox examples]$ docker container kill gsr-example-etcd 
gsr-example-etcd
```

Want to clean up old shut-down containers? Use `docker system prune`

```
[jaypipes@uberbox examples]$ docker system prune --force
Deleted Containers:
68647a58d41ded832c321e49c279f38461413155c155aff9efb5a36e1b7c9980
b24299f95cf6c61fe1df60f657bb5737cdad3c0c1615e2940b9fd0bb155c9d9c
8317f1f48462859d55f99f99a578b7ac2057946d710cbdb050e64ef50ebbec98

Deleted Images:
deleted: sha256:f0dc69907bba6166ad1029a3d4fc7ff475a02d5f7777ef6efe598284a8be29e1
deleted: sha256:f0159089b790cce5d80e14735145e6b596e44d9b3fe3354fe6caa5304a449a7a
deleted: sha256:c0756f6dc7f55f5c181dc2734c8911c7ec89def640b48e086da7aec48c8de389
deleted: sha256:86d96376daa9c4b035fa529255c5eab269566602af5119d8bf57fe6496b836c1
deleted: sha256:b8de90d2b40fa46dfdc167f7845ec71b3464d5b139352c92c8421219bf345161
deleted: sha256:57e953a27ab27bc1f2dfb6ff4b51dfa5013be0e119a0d7aff808c9c2165338a6
deleted: sha256:bec63ccd0de19c6274b033449ba9383896ca6f16b2097f452d0900784bb923a2
deleted: sha256:337d9021353f2bd9aadc57659a975481932a8d8555b3d839d0023ea023526d14
deleted: sha256:c6d2f3e51fb44dd49b38c3a05f3b474da429edd79371b449501e78cebd284788
deleted: sha256:3f08f41e6ec95ba55af06344650cdbbf8a7a1ee2a91211da37be568676726601
deleted: sha256:d38735f3bd94af5bdd54549fb2db530b850262b6d6dbf3e1200565265066eac7
deleted: sha256:5e60c0a651871bbd10d4a87e5dc67681d69649ffdcfcfd2cdc05742b25101756
deleted: sha256:330c7b21c5be36afec85a424ce6d71e059a16693bb34ce363001466e110f505a
deleted: sha256:97ae869f435ab79b108796844d6f10f1a4cc1a5e06451166f8875ec055695531
deleted: sha256:96c28024756302a07e440bbcd492252d27113842dbb0d1b1a98a963bb8749277
deleted: sha256:3ea86214d7779b70d5c7c4746a8fb9cdaa133e50ee73736e82f91f3640891d17
deleted: sha256:aa493423841380bb10cb6f0eeb48ba74a85f8df770a322948d31ca2cc0a927d0
deleted: sha256:1d3cadb473500253dd7719a0fd5b1f23611c2ac3e97074095fc07cce11698264
deleted: sha256:7545b4b78f4432fc0372917f83b55044d1cf96634dabb56125cb9513a5675b22
deleted: sha256:6bfeaab91ac2d245d622ef3c2435f9439a1e18413f0ae59d18872c66eb225f42
deleted: sha256:d794acdfb02a0424668325186006483c8b4b9877eb20b136a9bc153a056dba53
deleted: sha256:ca1cf51d0606051bc8fb2c8c8a9fa68cb4fcbbc6308c1e2ed57595353ba463c7
deleted: sha256:54d27607c184e835bb82869da8b3fb966fd15e2fbfe4fea646182dc636888888
deleted: sha256:bf2bfe4738c84348764b034a6cf30f77e6f0d2814c77c15b6c549b4f722d8736
deleted: sha256:9474a8458a18989f75cbd4f51613a35ff83ab9330e0a25456dc8c52b39094489
deleted: sha256:7de31ee7403df9c61bde96258c3030a8d5d1343c239a85caccd0390b9c17e005
deleted: sha256:493154b8ad9ff968713bfca948c58373b3984414d3a57f7e7e8b5deba0def729
deleted: sha256:20722dd6e94e476a62a0a4a6556058af3a9c1e2508f6599aa8386ec82315b75a

Total reclaimed space: 1.613GB
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

## Step 8: Stop the data application and verify the data endpoint is removed from `gsr`

OK, now we want to verify that when an application that has registered with
`gsr` stops, that the corresponding endpoint in the `gsr` registry ends up
being removed.

So, let's stop the "data" application by simply entering Ctrl-c in our terminal
window that has the "data" application running:

```
data$ GSR_ETCD_ENDPOINTS="http://172.16.28.68:2379" go run examples/cmd/data/main.go 
2017/04/25 12:51:35 [data:192.168.1.7:9000] Starting data service on 192.168.1.7:9000.
2017/04/25 12:51:35 [data:192.168.1.7:9000] Connecting to gsr.
2017/04/25 12:51:35 [data:192.168.1.7:9000] Registering 192.168.1.7:9000 with gsr.
2017/04/25 12:51:35 [data:192.168.1.7:9000] Listening for HTTP traffic on 192.168.1.7:9000.
^Csignal: interrupt
data$ 
```

We can check that the "web" application no longer sees the "data" endpoint in
the `gsr` registry by making an HTTP request to the "web" application again or
using the `etcdctl` CLI tool:

```
jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ ETCDCTL_API=3 etcdctl --endpoints="http://172.16.28.68:2379"  get --prefix gsr/services
gsr/services/web/192.168.1.7:8080

jaypipes@uberbox:~/src/github.com/jaypipes/gsr$ curl http://192.168.1.7:8080/
web service knows about endpoints: [web:192.168.1.7:8080]
```

**NOTE**: Behind the scenes, the "data" endpoint isn't actually *removed* from
the `gsr` registry. Rather, what happens is there is a keepalive heartbeat that
continually refreshes the TTL on the registered endpoint in `gsr` associated
with the `gsr.Registry` object that an application uses. When the application
shuts down and the `gsr.Registry` object is destroyed/garbage-collected, the
TTL on the endpoint expires and other applications are no longer able to see
the endpoint.

You can control this keepalive/lease timeout using the `GSR_LEASE_SECONDS`
environment variable, which defaults to 60 seconds.
