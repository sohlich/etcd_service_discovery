# Small service discovery client for etcd

This is small library, that wraps the oficial etcd go client, to simplify
service discovery.

[![GoDoc](https://godoc.org/github.com/sohlich/etcd_service_discovery?status.svg)](https://godoc.org/github.com/sohlich/etcd_service_discovery)


## How to register a service

```
// Create config
registryConfig := discovery.EtcdRegistryConfig{
		EtcdEndpoints: []string{"http://127.0.0.1:4001"}, //etcd instances
		ServiceName:   "core", //service type/name
		InstanceName:  "core1", //service instance name
		BaseUrl:       "127.0.0.1:8080", //base url to use to access service
	}

// Create client
registryClient, registryErr := discovery.New(registryConfig)
	if registryErr != nil {
		log.Panic(registryErr)
	}

// Register service to etcd
registryClient.Register()

```

## How to obtain a instances by service type/name

```
response, err := client.ServicesByName("test")

if err != nil {
	log.Panic(err)
}

log.Println(response)

// Obtained output:
// [127.0.0.1:8080]

```
