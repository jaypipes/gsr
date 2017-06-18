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
    "os"
    "strings"
    "syscall"

    "golang.org/x/net/context"
    "google.golang.org/grpc"
    etcd "github.com/coreos/etcd/clientv3"
    "github.com/cenkalti/backoff"
)

type Service struct {
    Name string
}

type Endpoint struct {
    Service *Service
    Address string
    lease etcd.LeaseID
}

type Heartbeat struct {
    ka <-chan *etcd.LeaseKeepAliveResponse
}

type registryLogs struct {
    elog *log.Logger
    log1 *log.Logger
    log2 *log.Logger
}

type Registry struct {
    config *Config
    logs *registryLogs
    client *etcd.Client
    watcher etcd.WatchChan
    heartbeats map[*Endpoint]*Heartbeat
}

// Returns the etcd key prefix representing the top-level "services" directory.
func (r *Registry) servicesKey() string {
    return r.config.EtcdKeyPrefix + "services/"
}

// Returns the etcd key prefix for a specific service.
func (r *Registry) serviceKey(service string) string {
    return r.servicesKey() + service
}

// Returns the etcd key for an endpoint within a service.
func (r *Registry) endpointKey(service string, endpoint string) string {
    return r.serviceKey(service) + "/" + endpoint
}

func (r *Registry) requestCtx() (context.Context, context.CancelFunc) {
    return context.WithTimeout(
        context.Background(),
        r.config.EtcdRequestTimeoutSeconds,
    )
}

// Given a full key, e.g. "gsr/services/web/127.0.0.1:80", returns the service
// and endpoint as strings, e.g. "web", "127.0.0.1:80"
func (r *Registry) partsFromKey(key string) (string, string) {
    parts := strings.Split(key[len(r.servicesKey()):], "/")
    return parts[0], parts[1]
}

func (r *Registry) LERR(message string, args ...interface{}) {
    if r.logs.elog == nil {
        return
    }
    outLevel := 1
    if (r.logs.elog.Flags() & log.Lshortfile) == log.Lshortfile {
        outLevel = 2
    }
    r.logs.elog.Output(
        outLevel,
        fmt.Sprintf("[gsr] ERROR: " + message, args...),
    )
}

func (r *Registry) L1(message string, args ...interface{}) {
    if r.logs.log1 == nil {
        return
    }
    outLevel := 1
    if (r.logs.log2.Flags() & log.Lshortfile) == log.Lshortfile {
        outLevel = 2
    }
    if r.config.LogLevel > 0 {
        r.logs.log1.Output(
            outLevel,
            fmt.Sprintf("[gsr] " + message, args...),
        )
    }
}

func (r *Registry) L2(message string, args ...interface{}) {
    if r.logs.log2 == nil {
        return
    }
    outLevel := 1
    if (r.logs.log1.Flags() & log.Lshortfile) == log.Lshortfile {
        outLevel = 2
    }
    if r.config.LogLevel > 1 {
        r.logs.log2.Output(
            outLevel,
            fmt.Sprintf("[gsr] " + message, args...),
        )
    }
}

// Returns a list of endpoints for a requested service type.
func (r *Registry) Endpoints(service string) ([]*Endpoint) {
    c := r.client
    sort := etcd.WithSort(etcd.SortByKey, etcd.SortAscend)
    skey := r.serviceKey(service)
    ctx, cancel := r.requestCtx()
    resp, err := c.KV.Get(ctx, skey, etcd.WithPrefix(), sort)
    cancel()
    if err != nil {
        r.L2("error looking up endpoints for service %s: %v",
             service, err)
        return []*Endpoint{}
    }

    numEps := resp.Count
    r.L2("read %d endpoints @ generation %d", numEps, resp.Header.Revision)

    lenServicesKey := len(skey)
    eps := make([]*Endpoint, numEps)
    for x, kv := range(resp.Kvs) {
        key := kv.Key
        // The full key will be "$KEY_PREFIX/services/$SERVICE/$ENDPOINT
        parts := strings.Split(string(key[lenServicesKey:]), "/")
        sname := parts[0]
        addr := parts[1]
        eps[x] = &Endpoint{
            Service: &Service{Name: sname},
            Address: addr,
        }
    }
    return eps
}

// Sets up a watch channel for any changes to the gsr registry so that the
// Registry object can refresh its map of service endpoints when changes occur.
func (r *Registry) setupWatch() {
    c := r.client
    key := r.servicesKey()
    r.L2("creating watch on %s", key)
    r.watcher = c.Watch(context.Background(), key, etcd.WithPrefix())
    go handleChanges(r)
}

// Sets up the channel heartbeat mechanism for the endpoint registered in this
// Registry.
func (r *Registry) setupHeartbeat(ep *Endpoint) error {
    c := r.client
    ch, err := c.KeepAlive(context.TODO(), ep.lease)
    if err != nil {
        return err
    }
    r.heartbeats[ep] = &Heartbeat{ka: ch}
    return nil
}

// Registers an endpoint for a service type and sets up all necessary heartbeat
// and watch mechanisms.
func (r *Registry) Register(ep *Endpoint) error {
    service := ep.Service.Name
    addr := ep.Address
    c := r.client
    lease, err := c.Grant(context.TODO(), r.config.LeaseSeconds)
    if err != nil {
        r.LERR("failed to grant lease in etcd: %v", err)
        return err
    }
    ep.lease = lease.ID
    eps := r.Endpoints(service)
    if ! contains(addr, eps) {
        err = r.createEndpoint(ep)
        if err != nil {
            return err
        }
    }

    err = r.setupHeartbeat(ep)
    if err != nil {
        return err
    }
    r.L2("started heartbeat channel")
    return nil
}

// Creates an entry for an endpoint in the gsr registry
func (r *Registry) createEndpoint(ep *Endpoint) error {
    service := ep.Service.Name
    endpoint := ep.Address
    c := r.client

    r.L2("creating new registry entry for %s:%s", service, endpoint)

    ekey := r.endpointKey(service, endpoint)
    onSuccess := etcd.OpPut(ekey, "", etcd.WithLease(ep.lease))
    // Ensure the $PREFIX/services/$SERVICE/$ENDPOINT key doesn't yet exist
    compare := etcd.Compare(etcd.Version(ekey), "=", 0)
    ctx, cancel := r.requestCtx()
    resp, err := c.KV.Txn(ctx).If(compare).Then(onSuccess).Commit()
    cancel()

    if err != nil {
        r.LERR("failed to create txn in etcd: %v", err)
        return err
    } else if resp.Succeeded == false {
        r.L2("concurrent write detected to key %v.", ekey)
    }
    return nil
}

// Reads the registry's watch channel and processes incoming events.
func handleChanges(r *Registry) {
    for cin := range(r.watcher) {
        for _, ev := range(cin.Events) {
            service, endpoint := r.partsFromKey(string(ev.Kv.Key))
            switch ev.Type {
                case etcd.EventTypeDelete:
                    r.L2("received notification that %s:%s was deleted. ",
                         service, endpoint)
                case etcd.EventTypePut:
                    r.L2("received notification that %s:%s was created. ",
                         service, endpoint)
            }
        }
    }
}

// Returns an etcd3 client using an exponential backoff and reconnect strategy.
// This is to be tolerant of the etcd infrastructure VMs/containers starting
// *after* a service that requires it.
func (r *Registry) connect() (*etcd.Client, error) {
    var err error
    var client *etcd.Client
    fatal := false
    connectTimeout := r.config.EtcdConnectTimeoutSeconds
    cfg := r.config.EtcdConfig()
    etcdEps := cfg.Endpoints

    bo := backoff.NewExponentialBackOff()
    bo.MaxElapsedTime = connectTimeout

    r.L2("connecting to etcd endpoints %v (w/ %s overall timeout).",
         etcdEps, connectTimeout.String())

    fn := func() error {
        client, err = etcd.New(*cfg)
        if err != nil {
            if (err == grpc.ErrClientConnTimeout ||
                err == context.Canceled ||
                err == context.DeadlineExceeded) {
                // Each of these scenarios are errors that we can retry the
                // operation. Services may come up in different order and we
                // don't want to require a specific order of startup...
                return err
            }
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
                        destAddr := oerr.Addr
                        if destAddr == nil {
                            // Unknown host... probably a DNS failure and not
                            // something we're going to be able to recover from in
                            // a retry, so bail out
                            fatal = true
                        }
                        // If not unknown host, most likely a dial: tcp
                        // connection refused. In that case, let's retry. etcd
                        // may not have been brought up before the calling
                        // application/service..
                        return err
                    } else if t.Op == "read" {
                        // connection refused. In that case, let's retry. etcd
                        // may not have been brought up before the calling
                        // application/service..
                        return err
                    }
                case syscall.Errno:
                    if t == syscall.ECONNREFUSED {
                        // connection refused. In that case, let's retry. etcd
                        // may not have been brought up before the calling
                        // application/service..
                        return err
                    }
                default:
                    r.L2("got unrecoverable %T error: %v attempting to " +
                         "connect to etcd", err, err)
                    fatal = true
                    return err
            }
        }
        return nil
    }

    ticker := backoff.NewTicker(bo)

    attempts:= 0
    for _ = range ticker.C {
        if err = fn(); err != nil {
            attempts += 1
            if fatal {
                break
            }
            r.L2("failed to connect to gsr: %v. retrying.", err)
            continue
        }

        ticker.Stop()
        break
    }

    if err != nil {
        r.LERR("failed to connect to gsr. final error reported: %v", err)
        r.L2("attempted %d times over %v. exiting.",
             attempts, bo.GetElapsedTime().String())
        return nil, err
    }
    return client, nil
}

// Creates a new gsr.Registry object, registers a service and endpoint with the
// registry, and returns the registry object.
func New() (*Registry, error) {
    r := new(Registry)
    r.config = configFromEnv()
    logMode := (log.Ldate | log.Ltime | log.LUTC)
    if r.config.LogMicroseconds {
        logMode |= log.Lmicroseconds
    }
    if r.config.LogFileTrace {
        logMode |= log.Lshortfile
    }
    r.logs = &registryLogs{
        elog: log.New(os.Stderr, "", logMode),
        log1: log.New(os.Stderr, "", logMode),
        log2: log.New(os.Stderr, "", logMode),
    }
    client, err := r.connect()
    if err != nil {
        return nil, err
    }
    r.client = client
    r.L1("connected to registry.")

    r.heartbeats = make(map[*Endpoint]*Heartbeat, 0)

    r.setupWatch()
    return r, nil
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
