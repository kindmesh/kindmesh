package processor

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kindmesh/kindmesh/internal/spec"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

var GlobalCache *MetaCache
var Emitor emitor = func(dns *spec.DNSRequest, router *spec.RouterRequst) {
	log.Println("emit", *dns, *router)
}

type malloctor interface {
	AllocateForNames(names map[string]bool) (map[string]string, error)
}

type emitor func(dns *spec.DNSRequest, router *spec.RouterRequst)

func Init() {
	hostIP := os.Getenv("HOST_IP")
	if hostIP == "" {
		hostIP = "127.0.0.1"
	}
	clusetrDomain := os.Getenv("CLUSTER_DOMAIN")
	if clusetrDomain == "" {
		clusetrDomain = "svc.cluster.local"
	}
	GlobalCache = &MetaCache{p: newProcessor(hostIP, clusetrDomain, newDefaultMalloctor(), Emitor)}
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
	svc := &spec.L7Service{}
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
