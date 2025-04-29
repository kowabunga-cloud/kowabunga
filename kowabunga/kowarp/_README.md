## Local testing

To test `kowarp` in a non-kowabunga environment and load a virtual router manually :
Replace the following `(*Kowarp) loadConfig` with the following code and build it (replace yours Ips and config up to your desires):

```
func (*Kowarp) loadTestConfig() ([]*virtualRouter, error) {
	var (
		peer_ips []net.IP
		priority uint8
		vrs      []*virtualRouter
	)
	ips := []string{"10.69.64.139", "10.69.64.140"}
	localip := getLocalIP()

	for _, ip := range ips {
		if ip != localip {
			peer_ips = append(peer_ips, net.ParseIP(ip))
		}
	}

	if localip == ips[0] {
		priority = 255
	} else {
		priority = 0
	}
	ip, err := netlink.ParseAddr("10.69.68.141/24")
	vips := []netlink.Addr{*ip}

	advitf, _ := findFirstPrivateInterface()
	vr, err := NewVirtualRouter(1, peer_ips, priority, vips, 100, advitf.Name, advitf.Name, net.ParseIP(localip), false, nil)
	if err != nil {
		return nil, err
	}
	vrs = append(vrs, vr)

	klog.Infof("Loaded VR : \n%#v", *vr)
	return vrs, nil
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.IsPrivate() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
```