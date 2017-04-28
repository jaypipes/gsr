package gsr

import (
    "crypto/tls"
    "io/ioutil"
    "path/filepath"
    "strings"
    "time"

    etcd "github.com/coreos/etcd/clientv3"
)

const (
    cfgPath = "/etc/gsr"
    defaultEtcdEndpoints = "http://127.0.0.1:2379"
    defaultEtcdKeyPrefix = "gsr/"
    defaultEtcdConnectTimeoutSeconds = 300
    defaultEtcdRequestTimeoutSeconds = 1
    defaultEtcdDialTimeoutSeconds = 1
    defaultUseTls = false
    defaultLogLevel = 0
    defaultLeaseSeconds = 60
)

var (
    defaultTlsCertPath = filepath.Join(cfgPath, "server.pem")
    defaultTlsKeyPath = filepath.Join(cfgPath, "server.key")
)

// Returns the path to the TLS cert to use.
func tlsCertPath() string {
    return EnvOrDefaultStr(
        "GSR_TLS_CERT_PATH",
        defaultTlsCertPath,
    )
}

// Returns the path to the TLS cert to use.
func tlsKeyPath() string {
    return EnvOrDefaultStr(
        "GSR_TLS_KEY_PATH",
        defaultTlsKeyPath,
    )
}

// Returns an etcd configuration struct populated with all configured options.
func etcdConfig() *etcd.Config {
    return &etcd.Config{
        Endpoints: etcdEndpoints(),
        DialTimeout: etcdDialTimeout(),
        TLS: tlsConfig(),
    }
}

// Returns the TLS configuration struct to use with etcd client.
func tlsConfig() *tls.Config {
    cfg := &tls.Config{}

    if ! useTls() {
        return nil
    }

    certPath := tlsCertPath()
    keyPath := tlsKeyPath()

    if certPath == "" || keyPath == "" {
        debug("error setting up TLS configuration. Either cert or " +
              "key path not specified.")
        return nil
    }

    certContent, err := ioutil.ReadFile(certPath)
    if err != nil {
        debug("error getting cert content: %v", err)
        return nil
    }

    keyContent, err := ioutil.ReadFile(keyPath)
    if err != nil {
        debug("error getting key content: %v", err)
        return nil
    }

    kp, err := tls.X509KeyPair(certContent, keyContent)
    if err != nil {
        debug("error setting up TLS cert: %v.", err)
        return nil
    }

    cfg.MinVersion = tls.VersionTLS10
    cfg.InsecureSkipVerify = false
    cfg.Certificates = []tls.Certificate{kp}
    return cfg
}

// Returns whether or not to use TLS in communicating with etcd.
func useTls() bool {
    return EnvOrDefaultBool(
        "GSR_USE_TLS",
        defaultUseTls,
    )
}

// Returns the set of etcd3 endpoints used by gsr.
func etcdEndpoints() []string {
    eps := strings.Split(
        EnvOrDefaultStr(
            "GSR_ETCD_ENDPOINTS", defaultEtcdEndpoints,
        ),
        ",",
    )
    res := make([]string, len(eps))
    // Ensure endpoints begin with http[s]:// and contain a port. If missing,
    // add default etcd port.
    for x, ep := range(eps) {
        if ! strings.HasPrefix(ep, "http") {
            ep = "http://" + ep
        }
        if strings.Count(ep, ":") == 1 {
            ep = ep + ":2379"
        }
        res[x] = ep
    }
    return res
}

/// Returns the key prefix used for the gsr registry in etcd.
func etcdKeyPrefix() string {
    return strings.TrimRight(
        EnvOrDefaultStr(
            "GSR_KEY_PREFIX",
            defaultEtcdKeyPrefix,
        ),
        "/",
    ) + "/"
}

// Returns the timeout, as time.Duration, of number of seconds gsr will try to
// connect to etcd. This value is the entire length of time, including all
// retries, that gsr will attempt to connect.
func etcdConnectTimeout() time.Duration {
    return time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_CONNECT_TIMEOUT_SECONDS",
            defaultEtcdConnectTimeoutSeconds,
        ),
    ) * time.Second
}

// Returns the timeout, as time.Duration, of number of seconds gsr will use for
// the dial timeout connecting to etcd3 endpoints.
func etcdDialTimeout() time.Duration {
    return time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_DIAL_TIMEOUT_SECONDS",
            defaultEtcdDialTimeoutSeconds,
        ),
    ) * time.Second
}

// Returns the timeout, as time.Duration, of number of seconds gsr will use
// when calling etcd.
func etcdRequestTimeout() time.Duration {
    return time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_REQUEST_TIMEOUT_SECONDS",
            defaultEtcdRequestTimeoutSeconds,
        ),
    ) * time.Second
}

// Returns the etcd key prefix representing the top-level "services" directory.
func servicesKey() string {
    return etcdKeyPrefix() + "services/"
}

// Returns the etcd key prefix for a specific service.
func serviceKey(service string) string {
    return servicesKey() + service
}

// Returns the etcd key for an endpoint within a service.
func endpointKey(service string, endpoint string) string {
    return serviceKey(service) + "/" + endpoint
}

// Returns the logging level for gsr.
func logLevel() int {
    return EnvOrDefaultInt(
        "GSR_LOG_LEVEL",
        defaultLogLevel,
    )
}

// Returns the currently-configured endpoint key lease timeout in seconds
func leaseTimeout() int64 {
    return int64(EnvOrDefaultInt(
        "GSR_LEASE_SECONDS",
        defaultLeaseSeconds,
    ))
}
