package discovery

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
)

const (
	TTL             = 30 * time.Second
	KeepAlivePeriod = 20 * time.Second
)

// Interface for ServiceDiscovery client
// for etcd.
type RegistryClient interface {
	// Register given service
	// to etcd instance
	Register() error
	Unregister() error
	ServicesByName(name string) ([]string, error)
}

type EtcdRegistryConfig struct {
	EtcdEndpoints   []string
	ServiceName     string
	InstanceName    string
	BaseUrl         string
	etcdKey         string
	keepAliveTicker *time.Ticker
	cancel          context.CancelFunc
}

type EtcdReigistryClient struct {
	EtcdRegistryConfig
	etcdKApi client.KeysAPI
}

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

func (e *EtcdReigistryClient) Register() {
	e.etcdKey = buildKey(e.ServiceName, e.InstanceName)
	value := registerDTO{
		e.BaseUrl,
	}

	val, _ := json.Marshal(value)
	e.keepAliveTicker = time.NewTicker(KeepAlivePeriod)
	ctx, c := context.WithCancel(context.TODO())
	e.cancel = c

	insertFunc := func() {
		e.etcdKApi.Set(context.Background(), e.etcdKey, string(val), &client.SetOptions{
			TTL: TTL,
		})
	}
	insertFunc()

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

}

func (e *EtcdReigistryClient) Unregister() {
	e.cancel()
	e.keepAliveTicker.Stop()
	e.etcdKApi.Delete(context.Background(), e.etcdKey, nil)
}

func (e *EtcdReigistryClient) ServicesByName(name string) ([]string, error) {
	response, err := e.etcdKApi.Get(context.Background(), fmt.Sprintf("/%s", name), nil)
	ipList := make([]string, 0)
	if err == nil {
		for _, node := range response.Node.Nodes {
			val := &registerDTO{}
			json.Unmarshal([]byte(node.Value), val)
			ipList = append(ipList, val.BaseUrl)
		}
	}
	return ipList, err
}

type registerDTO struct {
	BaseUrl string
}

func buildKey(servicetype, instanceName string) string {
	return fmt.Sprintf("%s/%s", servicetype, instanceName)
}
