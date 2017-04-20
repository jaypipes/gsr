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
    endpoints map[string][]string
    client *etcd.Client
    generation int64
    heartbeat <-chan *etcd.LeaseKeepAliveResponse
    watcher etcd.WatchChan
    lease etcd.LeaseID
}

// Returns a list of endpoints for a requested service type.
func (r *Registry) Endpoints(service string) ([]string) {
    eps, found := r.endpoints[service]
    if ! found {
        return nil
    }
    return eps
}

// Loads all endpoints in the gsr registry
func (r *Registry) load() error {
    c := r.client
    sort := etcd.WithSort(etcd.SortByKey, etcd.SortAscend)
    skey := servicesKey()
    ctx, cancel := requestCtx()
    resp, err := c.KV.Get(ctx, skey, etcd.WithPrefix(), sort)
    cancel()
    if err != nil {
        return err
    }

    curGen := resp.Header.Revision
    debug("found initial generation of gsr registry: %d", curGen)

    // Set the stored registry generation to the entire key range's revision
    r.generation = curGen

    // The full key will be "$KEY_PREFIX/services/$SERVICE/$ENDPOINT
    lenServicesKey := len(skey)
    eps := make(map[string][]string, 0)
    for _, kv := range(resp.Kvs) {
        key := kv.Key
        parts := strings.Split(string(key[lenServicesKey:]), "/")
        service := parts[0]
        endpoint := parts[1]
        if _, created := eps[service]; ! created {
            eps[service] = []string{endpoint}
        } else {
            eps[service] = append(eps[service], endpoint)
        }
    }
    debug("found endpoints %v", eps)
    r.endpoints = eps

    return nil
}

// Sets up a watch channel for any changes to the gsr registry so that the
// Registry object can refresh its map of service endpoints when changes occur.
func (r *Registry) setupWatch() error {
    c := r.client
    key := servicesKey()
    debug("creating watch on %s", key)
    r.watcher = c.Watch(context.Background(), key, etcd.WithPrefix())
    go handleChanges(r)
    return nil
}

// Registers an endpoint for a service type. This method is idempotent and
// merely updates the registry's TTL for the endpoint if the endpoint already
// is registered.
func (r *Registry) register(service string, endpoint string) error {
    c := r.client
    lease, err := c.Grant(context.TODO(), leaseTimeout())
    if err != nil {
        log.Printf("error: failed to grant lease in etcd: %v", err)
        return err
    }
    r.lease = lease.ID
    efound := false
    eps, sfound := r.endpoints[service]

    if sfound {
        for _, ep := range(eps) {
            if ep == endpoint {
                efound = true
            }
        }
    }
    if ! efound {
        err = r.newEndpoint(service, endpoint)
        if err != nil {
            return err
        }
    }

    ch, err := c.KeepAlive(context.TODO(), r.lease)
    if err != nil {
        return err
    }
    r.heartbeat = ch
    debug("started heartbeat channel for %s:%s", service, endpoint)
    return nil
}

// Creates an entry for a brand new service type in the gsr registry
func (r *Registry) newEndpoint(service string, endpoint string) error {
    c := r.client

    debug("creating new registry entry for %s:%s", service, endpoint)

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
        debug("concurrent write detected to key %v.", ekey)
    }
    eps, sfound := r.endpoints[service]
    if ! sfound {
        eps = []string{endpoint}
    } else {
        eps = append(eps, endpoint)
    }
    r.endpoints[service] = eps
    r.generation = resp.Header.Revision
    return nil
}

// Reads the registry's watch channel and processes incoming events.
func handleChanges(r *Registry) {
    for cin := range(r.watcher) {
        for _, ev := range(cin.Events) {
            service, endpoint := partsFromKey(string(ev.Kv.Key))
            switch ev.Type {
                case etcd.EventTypeDelete:
                    epfound := false
                    if eps, sfound := r.endpoints[service]; sfound {
                        eps = removeEndpoint(eps, endpoint, &epfound)
                        if len(eps) == 0 {
                            debug("no more endpoints in service %s. " +
                                  "deleting +service from registry.",
                                  service)
                            delete(r.endpoints, service)
                        }
                        if epfound {
                            debug("received endpoint delete notification. " +
                                  "removed %s:%s from registry.",
                                  service, endpoint)
                            continue
                        }
                    }
                    // If the endpoint or the service was not found, we
                    // drop into the debug output below...
                    debug("received endpoint delete notification for an " +
                          "unknown endpoint: %s:%s. ignoring.",
                          service, endpoint)
                case etcd.EventTypePut:
                    if eps, sfound := r.endpoints[service]; sfound {
                        for _, v := range(eps) {
                            if v == endpoint {
                                debug("received endpoint create notification " +
                                      "for %s:%s but already had endpoint " +
                                      "in registry. ignoring.",
                                      service, endpoint)
                                continue
                            }
                        }
                        eps = append(eps, endpoint)
                        continue
                    } else {
                        r.endpoints[service] = []string{endpoint}
                    }
                    debug("received endpoint create notification. added " +
                          "%s:%s to registry.", service, endpoint)
            }
            r.generation = ev.Kv.ModRevision
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
                        msg := "error: unknown host trying reach etcd " +
                               "endpoint. unrecoverable error, so exiting."
                        debug(msg)
                        fatal = true
                        return err
                    } else if t.Op == "read" {
                        // Connection refused. This is an error we can backoff
                        // and retry in case the application running gsr.New()
                        // started before the etcd data store
                        return err
                    }
                case syscall.Errno:
                    if t == syscall.ECONNREFUSED {
                        // Connection refused. This is an error we can backoff
                        // and retry in case the application running gsr.New()
                        // started before the etcd data store
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
            msg := "error: failed attempting to get service registry: " +
                   "%v. Exiting."
            log.Fatalf(msg, err)
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
        debug("attempted %d times over %v. exiting.", attempts, bo.GetElapsedTime().String())
        return nil, err
    }
    return client, nil
}

// Creates a new gsr.Registry object, registers a service and endpoint with the
// registry, and returns the registry object.
func Start(service string, endpoint string) (*Registry, error) {
    r := new(Registry)
    client, err := connect()
    if err != nil {
        return nil, err
    }
    info("connected to registry.")
    r.generation = 0
    r.client = client
    err = r.load()
    if err != nil {
        return nil, err
    }
    info("loaded registry.")
    err = r.register(service, endpoint)
    if err != nil {
        return nil, err
    }
    info("registered %s:%s", service, endpoint)
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
