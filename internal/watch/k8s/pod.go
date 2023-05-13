package k8s

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

func watchPods() {
	clientSet, err := kubernetes.NewForConfig(getRestConfig())
	if err != nil {
		log.Fatal(err)
	}

	//create an api object
	api := clientSet.CoreV1()

	opts := metav1.ListOptions{}

	//Create a watcher on pods

	podWatcher, err := api.Pods("").Watch(context.Background(), opts)
	if err != nil {
		log.Fatal(err)
	}

	//Watch loop
	podChannel := podWatcher.ResultChan()
	for event := range podChannel {
		pod, ok := event.Object.(*v1.Pod)
		if !ok {
			log.Fatal(err)
		}

		switch event.Type {
		case watch.Added:
			log.Printf(" Pod %s-%s added %s %v \n", pod.Spec.NodeName, pod.Name, pod.Kind, pod.Labels)

		case watch.Modified:
			log.Printf(" Pod %s update \n", pod.Name)

		case watch.Deleted:
			log.Printf(" Pod %s deleted \n", pod.Name)

		}
	}
}
