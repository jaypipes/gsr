package gsr

import (
    "crypto/tls"
    "fmt"
    "io/ioutil"
    "os"
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
    defaultUseTLS = false
    defaultLogLevel = 0
    defaultLeaseSeconds = 60
)

var (
    defaultTLSCertPath = filepath.Join(cfgPath, "server.pem")
    defaultTLSKeyPath = filepath.Join(cfgPath, "server.key")
)

type Config struct {
    EtcdEndpoints []string
    EtcdKeyPrefix string
    EtcdConnectTimeoutSeconds time.Duration
    EtcdRequestTimeoutSeconds time.Duration
    EtcdDialTimeoutSeconds time.Duration
    UseTLS bool
    TLSCertPath string
    TLSKeyPath string
    LogLevel int
    LeaseSeconds int64
}

func configFromEnv() *Config {
    endpoints := etcdEndpoints()
    keyPrefix := strings.TrimRight(
        EnvOrDefaultStr(
            "GSR_KEY_PREFIX",
            defaultEtcdKeyPrefix,
        ),
        "/",
    ) + "/"
    connectTimeout := time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_CONNECT_TIMEOUT_SECONDS",
            defaultEtcdConnectTimeoutSeconds,
        ),
    ) * time.Second
    requestTimeout := time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_REQUEST_TIMEOUT_SECONDS",
            defaultEtcdRequestTimeoutSeconds,
        ),
    ) * time.Second
    dialTimeout := time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_DIAL_TIMEOUT_SECONDS",
            defaultEtcdDialTimeoutSeconds,
        ),
    ) * time.Second

    useTLS := EnvOrDefaultBool(
        "GSR_USE_TLS",
        defaultUseTLS,
    )
    certPath := EnvOrDefaultStr(
        "GSR_TLS_CERT_PATH",
        defaultTLSCertPath,
    )
    keyPath := EnvOrDefaultStr(
        "GSR_TLS_KEY_PATH",
        defaultTLSKeyPath,
    )

    logLevel := EnvOrDefaultInt(
        "GSR_LOG_LEVEL",
        defaultLogLevel,
    )
    leaseSeconds := int64(EnvOrDefaultInt(
        "GSR_LEASE_SECONDS",
        defaultLeaseSeconds,
    ))
    cfg := &Config{
        EtcdEndpoints: endpoints,
        EtcdKeyPrefix: keyPrefix,
        EtcdConnectTimeoutSeconds: connectTimeout,
        EtcdRequestTimeoutSeconds: requestTimeout,
        EtcdDialTimeoutSeconds: dialTimeout,
        UseTLS: useTLS,
        TLSCertPath: certPath,
        TLSKeyPath: keyPath,
        LogLevel: logLevel,
        LeaseSeconds: leaseSeconds,
    }
    return cfg
}

// Returns an etcd configuration struct populated with all configured options.
func (c *Config) EtcdConfig() *etcd.Config {
    return &etcd.Config{
        Endpoints: c.EtcdEndpoints,
        DialTimeout: c.EtcdDialTimeoutSeconds,
        TLS: c.TLSConfig(),
    }
}

// Returns the TLS configuration struct to use with etcd client.
func (c *Config) TLSConfig() *tls.Config {
    cfg := &tls.Config{}

    if ! c.UseTLS {
        return nil
    }
    certPath := c.TLSCertPath
    keyPath := c.TLSKeyPath

    if certPath == "" || keyPath == "" {
        fmt.Fprintf(
            os.Stderr,
            "error setting up TLS configuration. Either cert or " +
            "key path not specified.",
        )
        return nil
    }

    certContent, err := ioutil.ReadFile(certPath)
    if err != nil {
        fmt.Fprintf(
            os.Stderr,
            "error getting cert content: %v",
            err,
        )
        return nil
    }

    keyContent, err := ioutil.ReadFile(keyPath)
    if err != nil {
        fmt.Fprintf(
            os.Stderr,
            "error getting key content: %v",
            err,
        )
        return nil
    }

    kp, err := tls.X509KeyPair(certContent, keyContent)
    if err != nil {
        fmt.Fprintf(
            os.Stderr,
            "error setting up TLS cert: %v.",
            err,
        )
        return nil
    }

    cfg.MinVersion = tls.VersionTLS10
    cfg.InsecureSkipVerify = false
    cfg.Certificates = []tls.Certificate{kp}
    return cfg
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
