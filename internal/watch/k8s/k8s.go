package k8s

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kindmesh/kindmesh/internal/watch/processor"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func Watch() {
	go watchCRDs()
	go watchPods()
	go processor.Init()
}

func getRestConfig() *rest.Config {
	configPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// TODO: use in cluster mode
		log.Fatal(err)
	}
	//Load kubernetes config
	cfg, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		log.Fatal(err)
	}
	return cfg
}
