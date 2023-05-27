package processor

import (
	"fmt"

	"github.com/kindmesh/kindmesh/internal/watch/netdevice"
)

var gwIPSource = map[string]bool{}

type defaultMalloctor struct {
	allocated   map[string]string
	allocatedIP map[string]bool
}

func init() {
	for i := 100; i < 200; i++ {
		for j := 100; j < 256; j++ {
			gwIPSource[fmt.Sprintf("169.254.%d.%d", i, j)] = true
		}
	}
}

func newDefaultMalloctor() *defaultMalloctor {
	return &defaultMalloctor{
		allocated:   map[string]string{},
		allocatedIP: map[string]bool{},
	}
}

func (h *defaultMalloctor) AllocateForNames(names map[string]bool) (map[string]string, error) {
	ret := map[string]string{}
	needDelete := map[string]string{}
	needAdd := map[string]bool{}
	for name, ip := range h.allocated {
		needDelete[name] = ip
	}
	for name := range names {
		delete(needDelete, name)
		ip, ok := h.allocated[name]
		if ok {
			ret[name] = ip
		} else {
			needAdd[name] = true
		}
	}

	// reused ip from deleted
	for name := range needAdd {
		for nn, ip := range needDelete {
			ret[name] = ip
			delete(needDelete, nn)
			delete(needAdd, name)
			break
		}
	}

	if len(needDelete) == 0 && len(needAdd) == 0 {
		return ret, nil
	}

	for name, ip := range needDelete {
		err := netdevice.DelAddr(ip)
		if err != nil {
			return ret, err
		}
		delete(h.allocated, name)
		delete(h.allocatedIP, ip)
	}

	if len(needAdd) == 0 {
		return ret, nil
	}

	// list all
	existsAddrs, err := netdevice.ListAddr()
	if err != nil {
		return ret, err
	}

	for name := range needAdd {
		// try from exists ip
		targetIP := ""
		useExists := false
		for _, ip := range existsAddrs {
			if !gwIPSource[ip] {
				continue
			}
			if _, ok := h.allocatedIP[ip]; !ok {
				targetIP = ip
				useExists = true
				break
			}
		}
		// try from source
		if targetIP == "" {
			for ip := range gwIPSource {
				if _, ok := h.allocatedIP[ip]; !ok {
					targetIP = ip
					break
				}
			}
		}
		if targetIP == "" {
			return nil, fmt.Errorf("no availdable ip")
		}
		if !useExists {
			// add one ip
			err = netdevice.AddAddr(targetIP)
			if err != nil {
				return ret, err
			}
		}
		h.allocated[name] = targetIP
		h.allocatedIP[targetIP] = true
		ret[name] = targetIP
	}
	return ret, nil
}
