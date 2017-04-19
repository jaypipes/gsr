package gsr

import (
    "strings"
    "time"
)

const (
    defaultEtcdEndpoints = "http://127.0.0.1:2379"
    defaultEtcdKeyPrefix = "gsr/"
    defaultEtcdConnectTimeoutSeconds = 300 // 5 minutes
    defaultEtcdRequestTimeoutSeconds = 1
    defaultLogLevel = 0
)

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
    for _, ep := range(eps) {
        if ! strings.HasPrefix(ep, "http") {
            ep = "http://" + ep
        }
        if strings.Count(ep, ":") == 1 {
            ep = ep + ":2379"
        }
        res = append(res, ep)
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
// connect to etcd.
func etcdConnectTimeout() time.Duration {
    return time.Duration(
        EnvOrDefaultInt(
            "GSR_ETCD_CONNECT_TIMEOUT_SECONDS",
            defaultEtcdConnectTimeoutSeconds,
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

// Returns the logging level for gsr.
func logLevel() int {
    return EnvOrDefaultInt(
        "GSR_LOG_LEVEL",
        defaultLogLevel,
    )
}
