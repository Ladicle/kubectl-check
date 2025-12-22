package client

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
)

// Manager manages all Kubernetes clients
type Manager struct {
	RESTConfig     *rest.Config
	Clientset      kubernetes.Interface
	DynamicClient  dynamic.Interface
	MetadataClient metadata.Interface
}

// NewManager creates a new Manager and initializes all clients
func NewManager(config *rest.Config) (*Manager, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Manager{
		RESTConfig:     config,
		Clientset:      clientset,
		DynamicClient:  dynamicClient,
		MetadataClient: metadataClient,
	}, nil
}
