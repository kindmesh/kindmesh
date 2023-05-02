package k8s

import (
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClient() dynamic.Interface {
	//Define kubeconfig file
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	//Load kubernetes config
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	return clientset
}
