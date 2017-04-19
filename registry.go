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
    "log"
    "net"
    "strings"
    "syscall"
    "time"

    "golang.org/x/net/context"
    etcd "github.com/coreos/etcd/clientv3"

    "github.com/cenkalti/backoff"
)

const (
    defaultEtcdEndpoints = "http://127.0.0.1:2379"
    defaultEtcdKeyPrefix = "gsr/"
    defaultEtcdConnectTimeout = 300 // 5 minutes
    defaultEtcdRequestTimeout = 1
    defaultLogLevel = 0
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
    etcdConnectTimeout = time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_CONNECT_TIMEOUT_SECONDS",
            defaultEtcdConnectTimeout,
        ),
    ) * time.Second
    etcdRequestTimeout = time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_REQUEST_TIMEOUT_SECONDS",
            defaultEtcdRequestTimeout,
        ),
    ) * time.Second
    logLevel = EnvOrDefaultInt(
        "GSR_LOG_LEVEL",
        defaultLogLevel,
    )
    servicesKey = etcdKeyPrefix + "services/"
    byTypeKey = servicesKey + "by-type/"
)

type Registry struct {
    endpoints map[string][]string
    client *etcd.Client
    generation int64
}

// Registers an endpoint for a service type. This method is idempotent and
// merely updates the registry's TTL for the endpoint if the endpoint already
// is registered.
func (r *Registry) Register(stype string, endpoint string) error {
    eps, found := r.endpoints[endpoint]
    if ! found {
        eps = make([]string, 10)
        r.endpoints[stype] = eps
    }
    for _, ep := range(eps) {
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
    val, err := json.Marshal(eps)
    if err != nil {
        log.Printf("error: failure to serialize %v to JSON: %v", eps, err)
        return err
    }

    onSuccess := etcd.OpPut(key, string(val))
    compare := etcd.Compare(etcd.Version(servicesKey), "=", r.generation)
    ctx, cancel := requestCtx()
    _, err = c.KV.Txn(ctx).If(compare).Then(onSuccess).Commit()
    cancel()

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
func connect() (*etcd.Client, error) {
    var err error
    var client *etcd.Client
    fatal := false

    bo := backoff.NewExponentialBackOff()
    bo.MaxElapsedTime = etcdConnectTimeout

    debug("connecting to etcd endpoints: %v", etcdEndpoints)
    cfg := etcd.Config{
        Endpoints: etcdEndpoints,
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

// Loads all endpoints in the gsr registry
func (r *Registry) load() error {
    c := r.client
    ctx, cancel := requestCtx()
    sort := etcd.WithSort(etcd.SortByKey, etcd.SortAscend)
    resp, err := c.KV.Get(ctx, servicesKey, etcd.WithPrefix(), sort)
    cancel()
    if err != nil {
        return err
    }
    // The full key will be "$KEY_PREFIX/services/by-type/$SERVICE_TYPE
    // We will strip everything other than $SERVICE_TYPE below.
    lenPrefix := len(byTypeKey)
    // Set the stored registry generation to the entire key range's revision
    r.generation = resp.Header.Revision
    var eps []string
    for _, kv := range(resp.Kvs) {
        key := kv.Key
        if strings.HasPrefix(string(key), byTypeKey) {
            val := kv.Value
            stype := key[lenPrefix:]
            err := json.Unmarshal(val, &eps)
            if err != nil {
                debug("skipping key %v. failed to unmarshal JSON: %v.", key, err)
                continue
            }
            r.endpoints[string(stype)] = eps
        }
    }

    return nil
}

// Creates a new gsr.Registry object
func New() (*Registry, error) {
    r := new(Registry)
    client, err := connect()
    if err != nil {
        return nil, err
    }
    r.generation = 0
    r.client = client
    r.endpoints = make(map[string][]string, 10)
    err = r.load()
    if err != nil {
        return nil, err
    }
    info("connected to gsr registry.")
    return r, nil
}

func requestCtx() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), etcdRequestTimeout)
}
