package processor

import (
	"encoding/json"
	"log"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

var GlobalCache *MetaCache

func Init() {
	hostIP := os.Getenv("HOST_IP")
	if hostIP == "" {
		hostIP = "127.0.0.1"
	}
	clusetrDomain := os.Getenv("CLUSTER_DOMAIN")
	if clusetrDomain == "" {
		clusetrDomain = "svc.cluster.local"
	}
	GlobalCache = &MetaCache{p: newProcessor(hostIP, clusetrDomain)}
	go GlobalCache.p.processMetaEvent()
}

type MetaCache struct {
	p *processor
}

func (m *MetaCache) NewCRDEvent(eventType watch.EventType, obj interface{}) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Println("not unstructured object:", obj)
		return
	}
	bs, err := u.MarshalJSON()
	if err != nil {
		log.Println("MarshalJSON error:", err, obj)
		return
	}
	svc := &L7Service{}
	if err := json.Unmarshal(bs, svc); err != nil {
		log.Println("MarshalJSON error:", err, obj)
		return
	}
	m.p.addEvent(&metaEvent{eventType: eventType, object: svc})
}

func (m *MetaCache) NewPodEvent(event watch.Event) {
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		log.Println("not pod object:", event.Object)
		return
	}
	m.p.addEvent(&metaEvent{eventType: event.Type, object: pod})
}
