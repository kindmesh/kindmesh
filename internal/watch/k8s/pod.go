package k8s

import (
	"context"
	"log"

	"github.com/kindmesh/kindmesh/internal/watch/processor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func watchPods() {
	clientSet, err := kubernetes.NewForConfig(getRestConfig())
	if err != nil {
		log.Fatal(err)
	}

	api := clientSet.CoreV1()
	opts := metav1.ListOptions{}
	podWatcher, err := api.Pods("").Watch(context.Background(), opts)
	if err != nil {
		log.Fatal(err)
	}

	podChannel := podWatcher.ResultChan()
	for event := range podChannel {
		processor.GlobalCache.NewPodEvent(event)
	}
}
