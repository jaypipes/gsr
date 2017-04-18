package gsr

// Implements a simple service registry using an etcd3 K/V store. Within the
// etcd3 key namespace, there are "indexes" for service type:
//
// $KEY_PREFIX <-- environ['GSR_KEY_PREFIX']
// |
// -> /services
//    |
//    -> /by-type
//       |
//       -> /$SERVICE_TYPE: JSON-serialized list of endpoints

import (
    "encoding/json"
    "fmt"
    "log"
    "strings"
    "time"

    "golang.org/x/net/context"
    etcd "github.com/coreos/etcd/clientv3"
    "github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"

    "github.com/cenkalti/backoff"
)

const (
    defaultEtcdEndpoints = "http://127.0.0.1:2379"
    defaultEtcdKeyPrefix = "gsr/"
)

var (
    etcdEndpoints = strings.Split(
        EnvOrDefaultStr(
            "GSR_ETCD_ENDPOINTS", defaultEtcdEndpoints,
        ),
        ",",
    )
    etcdKeyPrefix = strings.TrimRight(
        EnvOrDefaultStr(
            "GSR_KEY_PREFIX",
            defaultEtcdKeyPrefix,
        ),
        "/",
    ) + "/"
    etcdReqTimeout = time.Second
    servicesKey = etcdKeyPrefix + "services/"
    byTypeKey = servicesKey + "by-type/"
)

type Registry struct {
    endpoints map[string][]string
    client *etcd.Client
    generation uint32
}

// Registers an endpoint for a service type. This method is idempotent and
// merely updates the registry's TTL for the endpoint if the endpoint already
// is registered.
func (r *Registry) Register(stype string, endpoint string) error {
    eps, found := r.endpoints[endpoint]
    if ! found {
        r.endpoints[stype] = eps = make([]string)
    }
    for ep, _ := range(eps) {
        if ep == endpoint {
            // Refresh the TTL for the endpoint
            return nil
        }
    }
    eps = append(eps, endpoint)
    // No such endpoint known. Add it to the service type entry's list of
    // endpoints.
    c := r.client
    key := byTypeKey + stype
    val, err := json.Marshall(eps)
    if err != nil {
        log.Printf("error: failure to serialize %v to JSON: %v", eps, err)
        return err
    }

    onSuccess := etcd.OpPut(key, val)
    compare := etcd.Compare(etcd.Version(servicesKey), "=", r.generation)
    ctx, cancel := context.WithTimeout(context.Background(), etcdReqTimeout)
    res, err := c.KV.Trx(ctx).If(compare).Then(onSuccess).Commit()

    if err != nil {
        log.Printf("error: transaction failed writing to etcd: %v", err)
        return err
    }

    r.generation += 1
    return nil
}

// Returns a list of endpoints for a requested service type.
func (r *Registry) Endpoints(stype string) ([]string) {
    eps, found := r.endpoints[stype]
    if ! found {
        return nil
    }
    return eps
}

// Returns an etcd3 client using an exponential backoff and reconnect strategy.
// This is to be tolerant of the etcd infrastructure VMs/containers starting
// *after* a service that requires it.
func etcdClient() (*etcd.Client, error) {
    var err error
    bo := backoff.NewExponentialBackOff()

    log.Printf("Connecting to etcd endpoints: %v", etcdEndpoints)
    cfg := etcd.Config{
        Endpoints: etcdEndpoints,
        DialTimeout: time.Second,
    }
    client, err := etcd.New(cfg)
    if err != nil {
        log.Fatalf("Failed to allocate memory for etcd client.")
        return nil, err
    }

    fn := func() error {
        var err error
        ctx, cancel := context.WithTimeout(context.Background(), etcdReqTimeout)
        _, err = client.KV.Get(ctx, "/services", etcd.WithPrefix())
        cancel()
        if err != nil {
            if err == rpctypes.ErrEmptyKey {
                // In this case, we have a functioning etcd3 cluster but
                // nothing has yet to create the first service key by calling
                // Registry.Register(). Just return nil and then return the
                // client after the retry loop. We're all good now.
                return nil
            }
            // Each of these scenarios are errors that we can retry the
            // operation. Services may come up in different order and we don't
            // want to require a specific order of startup...
            if err == context.Canceled || err == context.DeadlineExceeded {
                return fmt.Errorf("timeout or TCP cancellation")
            }
            msg := "Fatal error attempting to get service registry: " +
                   "%v. Exiting."
            log.Fatalf(msg, err)
        }
        return nil
    }

    ticker := backoff.NewTicker(bo)

    attempts:= 0
    log.Printf("Grabbing service registry from etcd.")
    for _ = range ticker.C {
        if err = fn(); err != nil {
            attempts += 1
            log.Printf("Failed to get service registry: %v. Retrying.", err)
            continue
        }

        ticker.Stop()
        break
    }

    if err != nil {
        msg := "Failed to get service registry. Final error reported: %v"
        log.Printf(msg, err)
        log.Printf("Attempted %d times over %v. Exiting.",
                   attempts, bo.GetElapsedTime().String())
        return nil, err
    }
    return client, nil
}

// Loads all endpoints in the gsr registry
func (r *Registry) load() error {
    c := r.client
    ctx, cancel := context.WithTimeout(context.Background(), etcdReqTimeout)
    sort := clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend)
    resp, err := client.KV.Get(ctx, servicesKey, etcd.WithPrefix(), sort)
    cancel()
    if err != nil {
        log.Printf("error: failed to load endpoints from registry: %v", err)
        return err
    }
    // The full key will be "$KEY_PREFIX/services/by-type/$SERVICE_TYPE
    // We will strip everything other than $SERVICE_TYPE below.
    lenPrefix = len(byTypeKey)
    // Set the stored registry generation to the entire key range's revision
    r.generation = resp.header.revision
    for _, kv := range(resp.Kvs) {
        key := kv.Key
        if strings.HasPrefix(key, byTypeKey) {
            val := kv.Value
            stype = key[lenPrefix:]
            eps, err := json.Unmarshal(val)
            if err != nil {
                log.Printf("error: failed to unmarshal JSON: %v", err)
                continue
            }
            r.endpoints[stype] = eps
        }
    }

    return nil
}

// Creates a new gsr.Registry object
func New() (*Registry, error) {
    r := new(Registry)
    client, err := etcdClient()
    if err != nil {
        log.Printf("error: unable to connect to etcd: %v", err)
        return nil, err
    }
    r.generation = 0
    r.client = client
    r.endpoints = make(map[string][]string, 10)
    err = r.load()
    if err != nil {
        log.Printf("error: unable to load endpoints from gsr registry: %v", err)
        return nil, err
    }
    return r, nil
}
