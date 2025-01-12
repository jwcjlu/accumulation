package main

import (
	"github.com/cilium/ebpf"
	"github.com/vishvananda/netlink"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf/rlimit"
)

const (
	BpfMapPath = "/sys/fs/bpf/backend_map"

	XDP_FLAGS_AUTO_MODE = 0 // custom
	XDP_FLAGS_DRV_MODE  = 1 << 2
)

func main() {
	// 移除资源限制
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("Failed to remove memlock limit: %v", err)
	}

	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %v", err)
	}
	defer objs.Close()

	// 绑定 XDP 程序到网络接口
	link, err := netlink.LinkByName("eth0")
	if err != nil {
		panic(err)
	}
	err = netlink.LinkSetXdpFdWithFlags(link, objs.XdpLoadBalancer.FD(), XDP_FLAGS_AUTO_MODE)
	if err != nil {
		err = netlink.LinkSetXdpFdWithFlags(link, objs.XdpLoadBalancer.FD(), XDP_FLAGS_DRV_MODE)
		if err != nil {
			panic(err)
		}
	}

	err = objs.BackendMap.Pin(BpfMapPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := objs.BackendMap.Put(uint32(8080), bpfBackendInfo{Ip: ipToInt("45.113.192.101"),
		Port: 80}); err != nil {
		log.Fatalf("Failed to update backend_map: %v", err)
	}
	m, err := ebpf.LoadPinnedMap(BpfMapPath, &ebpf.LoadPinOptions{})
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Put(uint32(8080), bpfBackendInfo{Ip: ipToInt("45.113.192.101"),
		Port: 80}); err != nil {
		log.Fatalf("Failed to update backend map: %v", err)
	}

	log.Println("XDP load balancer is running. Press Ctrl+C to stop.")

	// 等待中断信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
func ipToInt(ip string) uint32 {
	parsedIP := net.ParseIP(ip).To4()
	return uint32(parsedIP[0])<<24 | uint32(parsedIP[1])<<16 | uint32(parsedIP[2])<<8 | uint32(parsedIP[3])
}

func combineIPPort(ip string, port int) uint32 {
	ipInt := ipToInt(ip)
	return (ipInt << 16) | uint32(port)
}
