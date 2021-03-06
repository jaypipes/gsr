package gsr

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaypipes/envutil"
	etcd "go.etcd.io/etcd/clientv3"
)

const (
	cfgPath                          = "/etc/gsr"
	defaultEtcdEndpoints             = "http://127.0.0.1:2379"
	defaultEtcdKeyPrefix             = "gsr/"
	defaultEtcdConnectTimeoutSeconds = 300
	defaultEtcdRequestTimeoutSeconds = 1
	defaultEtcdDialTimeoutSeconds    = 1
	defaultUseTLS                    = false
	defaultLogLevel                  = 0
	defaultLogMicroseconds           = false
	defaultLogFileTrace              = false
	defaultLeaseSeconds              = 60
)

var (
	defaultTLSCertPath = filepath.Join(cfgPath, "server.pem")
	defaultTLSKeyPath  = filepath.Join(cfgPath, "server.key")
)

type Config struct {
	EtcdEndpoints             []string
	EtcdKeyPrefix             string
	EtcdConnectTimeoutSeconds time.Duration
	EtcdRequestTimeoutSeconds time.Duration
	EtcdDialTimeoutSeconds    time.Duration
	UseTLS                    bool
	TLSCertPath               string
	TLSKeyPath                string
	LogLevel                  int
	LogMicroseconds           bool
	LogFileTrace              bool
	LeaseSeconds              int64
}

func configFromEnv() *Config {
	endpoints := etcdEndpoints()
	keyPrefix := strings.TrimRight(
		envutil.WithDefault(
			"GSR_KEY_PREFIX",
			defaultEtcdKeyPrefix,
		),
		"/",
	) + "/"
	connectTimeout := time.Duration(
		envutil.WithDefaultInt(
			"GSR_ETCD_CONNECT_TIMEOUT_SECONDS",
			defaultEtcdConnectTimeoutSeconds,
		),
	) * time.Second
	requestTimeout := time.Duration(
		envutil.WithDefaultInt(
			"GSR_ETCD_REQUEST_TIMEOUT_SECONDS",
			defaultEtcdRequestTimeoutSeconds,
		),
	) * time.Second
	dialTimeout := time.Duration(
		envutil.WithDefaultInt(
			"GSR_ETCD_DIAL_TIMEOUT_SECONDS",
			defaultEtcdDialTimeoutSeconds,
		),
	) * time.Second

	useTLS := envutil.WithDefaultBool(
		"GSR_USE_TLS",
		defaultUseTLS,
	)
	certPath := envutil.WithDefault(
		"GSR_TLS_CERT_PATH",
		defaultTLSCertPath,
	)
	keyPath := envutil.WithDefault(
		"GSR_TLS_KEY_PATH",
		defaultTLSKeyPath,
	)

	logLevel := envutil.WithDefaultInt(
		"GSR_LOG_LEVEL",
		defaultLogLevel,
	)
	logMicroseconds := envutil.WithDefaultBool(
		"GSR_LOG_MICROSECONDS",
		defaultLogMicroseconds,
	)
	logFileTrace := envutil.WithDefaultBool(
		"GSR_LOG_FILE_TRACE",
		defaultLogFileTrace,
	)

	leaseSeconds := int64(envutil.WithDefaultInt(
		"GSR_LEASE_SECONDS",
		defaultLeaseSeconds,
	))
	cfg := &Config{
		EtcdEndpoints:             endpoints,
		EtcdKeyPrefix:             keyPrefix,
		EtcdConnectTimeoutSeconds: connectTimeout,
		EtcdRequestTimeoutSeconds: requestTimeout,
		EtcdDialTimeoutSeconds:    dialTimeout,
		UseTLS:                    useTLS,
		TLSCertPath:               certPath,
		TLSKeyPath:                keyPath,
		LogLevel:                  logLevel,
		LogMicroseconds:           logMicroseconds,
		LogFileTrace:              logFileTrace,
		LeaseSeconds:              leaseSeconds,
	}
	return cfg
}

// Returns an etcd configuration struct populated with all configured options.
func (c *Config) EtcdConfig() *etcd.Config {
	return &etcd.Config{
		Endpoints:   c.EtcdEndpoints,
		DialTimeout: c.EtcdDialTimeoutSeconds,
		TLS:         c.TLSConfig(),
	}
}

// Returns the TLS configuration struct to use with etcd client.
func (c *Config) TLSConfig() *tls.Config {
	cfg := &tls.Config{}

	if !c.UseTLS {
		return nil
	}
	certPath := c.TLSCertPath
	keyPath := c.TLSKeyPath

	if certPath == "" || keyPath == "" {
		fmt.Fprintf(
			os.Stderr,
			"error setting up TLS configuration. Either cert or "+
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
		envutil.WithDefault(
			"GSR_ETCD_ENDPOINTS", defaultEtcdEndpoints,
		),
		",",
	)
	res := make([]string, len(eps))
	// Ensure endpoints begin with http[s]:// and contain a port. If missing,
	// add default etcd port.
	for x, ep := range eps {
		if !strings.HasPrefix(ep, "http") {
			ep = "http://" + ep
		}
		if strings.Count(ep, ":") == 1 {
			ep = ep + ":2379"
		}
		res[x] = ep
	}
	return res
}
