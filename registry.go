package gsr

// Implements a simple service registry using an etcd3 K/V store. Within the
// etcd3 key namespace, there are "indexes" for service:
//
// $KEY_PREFIX <-- environ['GSR_KEY_PREFIX']
// |
// -> /services
//    |
//    ->> /$SERVICE
//        |
//        -> /$ENDPOINT1
//        -> /$ENDPOINT2

import (
    "fmt"
    "log"
    "net"
    "strings"
    "syscall"
    "time"

    "golang.org/x/net/context"
    etcd "github.com/coreos/etcd/clientv3"

    "github.com/cenkalti/backoff"
)

type Registry struct {
    service string
    endpoint string
    client *etcd.Client
    heartbeat <-chan *etcd.LeaseKeepAliveResponse
    watcher etcd.WatchChan
    lease etcd.LeaseID
}

// Returns a list of endpoints for a requested service type.
func (r *Registry) Endpoints(service string) ([]string) {
    c := r.client
    sort := etcd.WithSort(etcd.SortByKey, etcd.SortAscend)
    skey := serviceKey(service)
    ctx, cancel := requestCtx()
    resp, err := c.KV.Get(ctx, skey, etcd.WithPrefix(), sort)
    cancel()
    if err != nil {
        r.debug("error looking up endpoints for service %s: %v",
                service, err)
        return []string{}
    }

    numEps := resp.Count
    r.debug("read %d endpoints @ generation %d", numEps, resp.Header.Revision)

    lenServicesKey := len(skey)
    eps := make([]string, numEps)
    for x, kv := range(resp.Kvs) {
        key := kv.Key
        // The full key will be "$KEY_PREFIX/services/$SERVICE/$ENDPOINT
        parts := strings.Split(string(key[lenServicesKey:]), "/")
        ep := parts[1]
        eps[x] = ep
    }
    r.debug("found endpoints %v", eps)
    return eps
}

// Sets up a watch channel for any changes to the gsr registry so that the
// Registry object can refresh its map of service endpoints when changes occur.
func (r *Registry) setupWatch() {
    c := r.client
    key := servicesKey()
    r.debug("creating watch on %s", key)
    r.watcher = c.Watch(context.Background(), key, etcd.WithPrefix())
    go handleChanges(r)
}

// Sets up the channel heartbeat mechanism for the endpoint registered in this
// Registry.
func (r *Registry) setupHeartbeat() error {
    c := r.client
    ch, err := c.KeepAlive(context.TODO(), r.lease)
    if err != nil {
        return err
    }
    r.heartbeat = ch
    return nil
}

// Registers an endpoint for a service type and sets up all necessary heartbeat
// and watch mechanisms.
func (r *Registry) register() error {
    service := r.service
    endpoint := r.endpoint
    c := r.client
    lease, err := c.Grant(context.TODO(), leaseTimeout())
    if err != nil {
        log.Printf("error: failed to grant lease in etcd: %v", err)
        return err
    }
    r.lease = lease.ID
    eps := r.Endpoints(service)
    if ! contains(endpoint, eps) {
        err = r.createEndpoint()
        if err != nil {
            return err
        }
    }

    err = r.setupHeartbeat()
    if err != nil {
        return err
    }
    r.debug("started heartbeat channel")

    r.setupWatch()
    return nil
}

// Creates an entry for a brand new service type in the gsr registry
func (r *Registry) createEndpoint() error {
    service := r.service
    endpoint := r.endpoint
    c := r.client

    r.debug("creating new registry entry for %s:%s", service, endpoint)

    ekey := endpointKey(service, endpoint)
    onSuccess := etcd.OpPut(ekey, "", etcd.WithLease(r.lease))
    // Ensure the $PREFIX/services/$SERVICE/$ENDPOINT key doesn't yet exist
    compare := etcd.Compare(etcd.Version(ekey), "=", 0)
    ctx, cancel := requestCtx()
    resp, err := c.KV.Txn(ctx).If(compare).Then(onSuccess).Commit()
    cancel()

    if err != nil {
        log.Printf("error: failed to create txn in etcd: %v", err)
        return err
    } else if resp.Succeeded == false {
        r.debug("concurrent write detected to key %v.", ekey)
    }
    return nil
}

// Reads the registry's watch channel and processes incoming events.
func handleChanges(r *Registry) {
    for cin := range(r.watcher) {
        for _, ev := range(cin.Events) {
            service, endpoint := partsFromKey(string(ev.Kv.Key))
            switch ev.Type {
                case etcd.EventTypeDelete:
                    r.debug("received notification that %s:%s was deleted. ",
                            service, endpoint)
                case etcd.EventTypePut:
                    r.debug("received notification that %s:%s was created. ",
                            service, endpoint)
            }
        }
    }
}

// Returns an etcd3 client using an exponential backoff and reconnect strategy.
// This is to be tolerant of the etcd infrastructure VMs/containers starting
// *after* a service that requires it.
func connect() (*etcd.Client, error) {
    var err error
    var client *etcd.Client
    fatal := false

    bo := backoff.NewExponentialBackOff()
    bo.MaxElapsedTime = etcdConnectTimeout()

    etcdEps := etcdEndpoints()

    debug("connecting to etcd endpoints: %v.", etcdEps)
    cfg := etcd.Config{
        Endpoints: etcdEps,
        DialTimeout: time.Second,
    }

    fn := func() error {
        client, err = etcd.New(cfg)
        if err != nil {
            switch t := err.(type) {
                case *net.OpError:
                    oerr := err.(*net.OpError)
                    if oerr.Temporary() || oerr.Timeout() {
                        // Each of these scenarios are errors that we can retry
                        // the operation. Services may come up in different
                        // order and we don't want to require a specific order
                        // of startup...
                        return err
                    }
                    if t.Op == "dial" {
                        // Unknown host... probably a DNS failure and not
                        // something we're going to be able to recover from in
                        // a retry, to bail out
                        debug("error: unknown host trying reach etcd " +
                              "endpoint. unrecoverable error, so exiting.")
                        fatal = true
                        return err
                    } else if t.Op == "read" {
                        // Connection refused. This is an error we can backoff
                        // and retry in case the application running
                        // gsr.Start() started before the etcd data store
                        return err
                    }
                case syscall.Errno:
                    if t == syscall.ECONNREFUSED {
                        // Connection refused. This is an error we can backoff
                        // and retry in case the application running
                        // gsr.Start() started before the etcd data store
                        return err
                    }
            }
            if err == context.Canceled || err == context.DeadlineExceeded {
                return err
            }
            fatal = true
            return err
        }
        ctx, cancel := requestCtx()
        _, err = client.KV.Get(ctx, "/services", etcd.WithPrefix())
        cancel()
        if err != nil {
            // Each of these scenarios are errors that we can retry the
            // operation. Services may come up in different order and we don't
            // want to require a specific order of startup...
            if err == context.Canceled || err == context.DeadlineExceeded {
                return err
            }
            debug("error: failed attempting to get service registry: " +
                  "%v. Exiting.")
            fatal = true
            return err
        }
        return nil
    }

    ticker := backoff.NewTicker(bo)

    attempts:= 0
    debug("grabbing service registry from etcd.")
    for _ = range ticker.C {
        if err = fn(); err != nil {
            attempts += 1
            if fatal {
                break
            }
            debug("failed to get service registry: %v. retrying.", err)
            continue
        }

        ticker.Stop()
        break
    }

    if err != nil {
        debug("failed to get service registry. final error reported: %v", err)
        debug("attempted %d times over %v. exiting.",
              attempts, bo.GetElapsedTime().String())
        return nil, err
    }
    return client, nil
}

func (r *Registry) debug(message string, args ...interface{}) {
    message = fmt.Sprintf("%s:%s %s", r.service, r.endpoint, message)
    debug(message, args...)
}

func (r *Registry) info(message string, args ...interface{}) {
    message = fmt.Sprintf("%s:%s %s", r.service, r.endpoint, message)
    info(message, args...)
}

// Creates a new gsr.Registry object, registers a service and endpoint with the
// registry, and returns the registry object.
func Start(service string, endpoint string) (*Registry, error) {
    r := new(Registry)
    r.service = service
    r.endpoint = endpoint
    client, err := connect()
    if err != nil {
        return nil, err
    }
    r.info("connected to registry.")
    r.client = client
    err = r.register()
    if err != nil {
        return nil, err
    }
    r.info("registered %s:%s", service, endpoint)
    return r, nil
}

// Given a full key, e.g. "gsr/services/web/127.0.0.1:80", returns the service
// and endpoint as strings, e.g. "web", "127.0.0.1:80"
func partsFromKey(key string) (string, string) {
    parts := strings.Split(key[len(servicesKey()):], "/")
    return parts[0], parts[1]
}

// Given a slice of endpoint strings, remove one of the endpoints from the
// slice and return the resulting slice.
func removeEndpoint(eps []string, endpoint string, found *bool) []string {
    *found = true
    idx := -1
    for x, v := range(eps) {
        if v == endpoint {
            idx = x
        }
    }
    if idx == -1 {
        *found = false
        return eps
    }
    numEps := len(eps)
    eps[numEps - 1], eps[idx] = eps[idx], eps[numEps -1]
    return eps[:numEps - 1]
}

func servicesKey() string {
    return etcdKeyPrefix() + "services/"
}

func serviceKey(service string) string {
    return servicesKey() + service
}

func endpointKey(service string, endpoint string) string {
    return serviceKey(service) + "/" + endpoint
}

func requestCtx() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), etcdRequestTimeout())
}
