package discovery

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

const (
	// TTL is a time to live
	// for record in etcd
	TTL = 30 * time.Second

	// KeepAlivePeriod is period of
	// goroutine to
	// refresh the record in etcd.
	KeepAlivePeriod = 20 * time.Second
)

// RegistryClient is the interface for service
// discovery based on etcd.
type RegistryClient interface {
	// Register given service
	// to etcd instance
	Register() error

	// Unregister the service
	// from etcd.
	Unregister() error

	// ServiceByName
	// return the base addresses
	// for service instances.
	ServicesByName(name string) ([]string, error)
}

// EtcdRegistryConfig is configuration structure
// for connectiong to etcd instance
// and identify the service.
type EtcdRegistryConfig struct {
	// EtcdEndpoints that service
	// registry client connects to.
	EtcdEndpoints []string

	// ServiceName is the
	// the name of
	// service in application.
	ServiceName string

	// InstanceName is the identification
	// for service instance.
	InstanceName string

	// BaseURL is the url
	// that the service is
	// acessible on.
	BaseURL string

	etcdKey         string
	keepAliveTicker *time.Ticker
	cancel          context.CancelFunc
}

// EtcdReigistryClient  structure implements the
// basic functionality for service registration
// in etcd.
// After the Reguster method is called, the client
// periodically refreshes the record about the
// service.
type EtcdReigistryClient struct {
	EtcdRegistryConfig
	etcdKApi client.KeysAPI
}

// New creates the EtcdRegistryClient
// with service paramters defined by config.
func New(config EtcdRegistryConfig) (*EtcdReigistryClient, error) {
	cfg := client.Config{
		Endpoints:               config.EtcdEndpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)

	if err != nil {
		return nil, err
	}

	etcdClient := &EtcdReigistryClient{
		config,
		client.NewKeysAPI(c),
	}
	return etcdClient, nil
}

// Register register service
// to configured etcd instance.
// Once the Register is called, the client
// also periodically
// calls the refresh goroutine.
func (e *EtcdReigistryClient) Register() error {
	e.etcdKey = buildKey(e.ServiceName, e.InstanceName)
	value := registerDTO{
		e.BaseURL,
	}

	val, _ := json.Marshal(value)
	e.keepAliveTicker = time.NewTicker(KeepAlivePeriod)
	ctx, c := context.WithCancel(context.TODO())
	e.cancel = c

	insertFunc := func() error {
		_, err := e.etcdKApi.Set(context.Background(), e.etcdKey, string(val), &client.SetOptions{
			TTL: TTL,
		})
		return err
	}
	err := insertFunc()
	if err != nil {
		return err
	}

	// Exec the keep alive goroutine
	go func() {
		for {
			select {
			case <-e.keepAliveTicker.C:
				insertFunc()
				log.Printf("Keep alive routine for %s", e.ServiceName)
			case <-ctx.Done():
				log.Printf("Shutdown keep alive routine for %s", e.ServiceName)
				return
			}
		}
	}()
	return nil
}

// Unregister removes the service instance from
// etcd. Once the Unregister method is called,
// the periodicall refresh goroutine is cancelled.
func (e *EtcdReigistryClient) Unregister() error {
	e.cancel()
	e.keepAliveTicker.Stop()
	_, err := e.etcdKApi.Delete(context.Background(), e.etcdKey, nil)
	return err
}

// ServicesByName query the
// etcd instance for service nodes for service
// by given name.
func (e *EtcdReigistryClient) ServicesByName(name string) ([]string, error) {
	response, err := e.etcdKApi.Get(context.Background(), fmt.Sprintf("/%s", name), nil)
	ipList := make([]string, 0)
	if err == nil {
		for _, node := range response.Node.Nodes {
			val := &registerDTO{}
			json.Unmarshal([]byte(node.Value), val)
			ipList = append(ipList, val.BaseURL)
		}
	}
	return ipList, err
}

type registerDTO struct {
	BaseURL string
}

func buildKey(servicetype, instanceName string) string {
	return fmt.Sprintf("%s/%s", servicetype, instanceName)
}
