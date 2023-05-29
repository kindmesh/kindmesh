package netdevice

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

var interfaceName = ""

func EnsureDevice(deviceName string) error {
	interfaceName = deviceName
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, i := range interfaces {
		if i.Name == interfaceName {
			return nil
		}
	}
	return exec.Command("ip", "link", "add", interfaceName, "type", "bridge").Run()
}

func ListAddr() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range interfaces {
		if i.Name != interfaceName {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		ret := make([]string, 0)
		for _, a := range addrs {
			addr := strings.TrimSuffix(a.String(), "/32")
			ret = append(ret, addr)
		}
		return ret, nil
	}
	return nil, fmt.Errorf("network bridge %s not exist", interfaceName)
}

func AddAddr(ip string) error {
	return exec.Command("ip", "address", "add", ip+"/32", "dev", interfaceName).Run()
}

func DelAddr(ip string) error {
	return exec.Command("ip", "address", "del", ip+"/32", "dev", interfaceName).Run()
}
