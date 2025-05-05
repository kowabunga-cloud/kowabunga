package konvey

import (
	"fmt"

	"github.com/inetaf/tcpproxy"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
)

type TcpEndpoint struct {
	name            string
	port            int
	upstreamServers []string
}

type NetworkLoadBalancer struct {
	tcpproxy.Proxy
	tcpBackends []string
}

func NewNetworkLoadBalancer() (*NetworkLoadBalancer, error) {
	return &NetworkLoadBalancer{
		Proxy:       tcpproxy.Proxy{},
		tcpBackends: []string{},
	}, nil
}

func (lb *NetworkLoadBalancer) Start() error {
	klog.Infof("Starting up load-balancer ...")
	return lb.Proxy.Run()
}

func (lb *NetworkLoadBalancer) Reload(meta *metadata.InstanceMetadata) error {
	klog.Infof("Reloading configuration ...")

	lb.Stop()

	for _, e := range meta.Konvey.Endpoints {
		switch e.Protocol {
		case "tcp":
			for _, b := range e.Backends {
				backend := fmt.Sprintf("%s:%d", b.Host, b.Port)
				lb.tcpBackends = append(lb.tcpBackends, backend)
			}
		}
	}

	for _, b := range lb.tcpBackends {
		lb.Proxy.AddRoute(":8080", tcpproxy.To(b))
	}

	lb.Start()

	return nil
}

func (lb *NetworkLoadBalancer) Stop() error {
	klog.Infof("Stopping down load-balancer ...")
	return lb.Proxy.Close()
}

func (lb *NetworkLoadBalancer) Shutdown() error {
	klog.Infof("Shutting down load-balancer ...")
	return nil
}
