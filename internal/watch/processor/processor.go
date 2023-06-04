package processor

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/kindmesh/kindmesh/internal/spec"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type processor struct {
	eventChan chan *metaEvent
	hostIP    string

	services       map[string]*spec.L7Service
	ns2GwIP        map[string]string
	label2Pod      map[string]map[string]bool
	clusterDomain  string
	runningPod2ns  map[string]string
	currHostPod2ns map[string]string

	gwMalloctor malloctor
	emitor      emitor

	dnsRequest   *spec.DNSRequest
	routerRequst *spec.RouterRequest

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type metaEvent struct {
	eventType watch.EventType
	object    interface{}
}

type serviceInfo struct {
	name      string
	namespace string
}

func newProcessor(hostIP, clusterDomain string, gwMalloctor malloctor, emitor emitor) *processor {
	ctx, cancel := context.WithCancel(context.Background())
	return &processor{
		eventChan:      make(chan *metaEvent, 10000),
		hostIP:         hostIP,
		runningPod2ns:  map[string]string{},
		currHostPod2ns: map[string]string{},
		clusterDomain:  clusterDomain,
		services:       make(map[string]*spec.L7Service),
		label2Pod:      map[string]map[string]bool{},
		gwMalloctor:    gwMalloctor,
		emitor:         emitor,
		ctx:            ctx,
		cancel:         cancel,
		wg:             sync.WaitGroup{},
	}
}

func (p *processor) addEvent(e *metaEvent) {
	p.eventChan <- e
}

func (p *processor) processEvent() {
	hasEvent := false
	for {
		var e *metaEvent
		select {
		case e = <-p.eventChan:
			hasEvent = true
		default:
			if hasEvent {
				return
			}
			select {
			case e = <-p.eventChan:
				hasEvent = true
			case <-p.ctx.Done():
				return
			}
		}
		if svc, ok := e.object.(*spec.L7Service); ok {
			key := svc.MetaData.Name + "." + svc.MetaData.Namespace
			switch e.eventType {
			case watch.Added, watch.Modified:
				p.services[key] = svc
			case watch.Deleted:
				delete(p.services, key)
			default:
			}
		}
		if pod, ok := e.object.(*v1.Pod); ok {
			// pod label index
			for k, v := range pod.Labels {
				label := fmt.Sprintf("%s:%s", k, v)
				vv, ok := p.label2Pod[label]
				if e.eventType == watch.Deleted {
					if ok {
						delete(vv, pod.Status.PodIP)
						if len(vv) == 0 {
							delete(p.label2Pod, label)
						}
					}
				} else {
					if !ok {
						p.label2Pod[label] = map[string]bool{}
					}
					p.label2Pod[label][pod.Status.PodIP] = true
				}
			}
			switch e.eventType {
			case watch.Added, watch.Modified:
				if pod.Status.Phase == v1.PodRunning {
					p.runningPod2ns[pod.Status.PodIP] = pod.Namespace
				}
				if pod.Status.HostIP == p.hostIP {
					p.currHostPod2ns[pod.Status.PodIP] = pod.Namespace
				}
			case watch.Deleted:
				delete(p.runningPod2ns, pod.Status.PodIP)
				if pod.Status.HostIP == p.hostIP {
					delete(p.currHostPod2ns, pod.Status.PodIP)
				}
			default:
			}
		}
	}
}

func (p *processor) processMetaEvent() {
	p.wg.Add(1)
	defer p.wg.Done()
	for {
		p.processEvent()
		select {
		case <-p.ctx.Done():
			return
		default:
		}
		p.buildMeta()
	}
}
func (p *processor) buildMeta() {
	if err := p.ensureNS2GwIP(p.currHostPod2ns); err != nil {
		log.Printf("ensureNS2GwIP error %v\n", err)
		return
	}

	if err := p.buildDNS(); err != nil {
		log.Printf("build dns error %v\n", err)
		return
	}
	if err := p.buildEnvoy(); err != nil {
		log.Printf("build envoy api error %v\n", err)
		return
	}
	p.emitor(p.dnsRequest, p.routerRequst)
}

func (p *processor) buildDNS() error {
	var serviceList []string
	for _, svc := range p.services {
		serviceList = append(serviceList, svc.MetaData.Name+"."+svc.MetaData.Namespace+".")
	}

	p.dnsRequest = &spec.DNSRequest{
		Pod2NS:        p.currHostPod2ns,
		NS2GwIP:       p.ns2GwIP,
		ServiceList:   serviceList,
		ClusterDomain: p.clusterDomain,
	}
	return nil
}

func (p *processor) buildEnvoy() error {
	p.routerRequst = &spec.RouterRequest{
		Pod2NS:        p.runningPod2ns,
		NS2GwIP:       p.ns2GwIP,
		ServiceList:   p.services,
		ClusterDomain: p.clusterDomain,
		Label2Pod:     p.label2Pod,
	}
	return nil
}

func (p *processor) ensureNS2GwIP(pod2ns map[string]string) error {
	names := make(map[string]bool)
	for _, ns := range pod2ns {
		names[ns] = true
	}
	ns2GwIP, err := p.gwMalloctor.AllocateForNames(names)
	if err != nil {
		return fmt.Errorf("allocate gw ip error %v", err)
	}
	p.ns2GwIP = ns2GwIP
	return nil
}

func (p *processor) stop() {
	p.cancel()
	p.wg.Wait()
}
