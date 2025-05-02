/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	virt "github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	virtxml "libvirt.org/go/libvirtxml"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	LibvirtConnectionTimeoutSeconds = 5

	LibvirtProtocolTCP    = "tcp"
	LibvirtDefaultPortTCP = 16509

	LibvirtProtocolTLS    = "tls"
	LibvirtDefaultPortTLS = 16514

	LibvirtNoConnectionError = "no associated libvirt connection"
)

type LibvirtConnectionSettings struct {
	// Settings
	Address       string
	Port          int
	Protocol      string
	TLSClientKey  string
	TLSClientCert string
	TLSCA         string

	// Connection
	keepRunning bool
	Conn        *virt.Libvirt
}

func NewLibvirtConnectionSettings(cfg *KaktusAgentConfig) (*LibvirtConnectionSettings, error) {

	if cfg.Libvirt.Address == "" {
		return nil, fmt.Errorf("invalid libvirt configuration, missing address")
	}

	lcs := LibvirtConnectionSettings{
		Address:       cfg.Libvirt.Address,
		Port:          cfg.Libvirt.Port,
		Protocol:      cfg.Libvirt.Protocol,
		TLSClientKey:  cfg.Libvirt.TLS.PrivateKey,
		TLSClientCert: cfg.Libvirt.TLS.PublicCert,
		TLSCA:         cfg.Libvirt.TLS.CA,
	}

	// check for supported protocol and ports
	switch lcs.Protocol {
	case LibvirtProtocolTCP:
		if lcs.Port == 0 {
			lcs.Port = LibvirtDefaultPortTCP
		}
	case LibvirtProtocolTLS:
		if lcs.Port == 0 {
			lcs.Port = LibvirtDefaultPortTLS
		}
	default:
		return nil, fmt.Errorf("transport '%s' not implemented", lcs.Protocol)
	}

	err := lcs.Connect()
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s:%d", lcs.Address, lcs.Port)
	klog.Infof("Successfully initiated libvirt %s connection to %s", lcs.Protocol, host)

	// maintain connection
	lcs.keepRunning = true
	klog.Infof("Register connection monitor for %s", host)
	go lcs.RegisterConnectionMonitor()

	return &lcs, err
}

type TLS struct {
	timeout    time.Duration
	conf       *tls.Config
	host, port string
}

func (t *TLS) Dial() (net.Conn, error) {
	netDialer := net.Dialer{
		Timeout: t.timeout,
	}
	c, err := tls.DialWithDialer(
		&netDialer,
		"tcp",
		net.JoinHostPort(t.host, t.port),
		t.conf,
	)
	if err != nil {
		return nil, err
	}

	// When running over TLS, after connection libvirt writes a single byte to
	// the socket to indicate whether the server's check of the client's
	// certificate has succeeded.
	// See https://github.com/digitalocean/go-libvirt/issues/89#issuecomment-1607300636
	// for more details.
	buf := make([]byte, 1)
	if n, err := c.Read(buf); err != nil {
		_ = c.Close()
		return nil, err
	} else if n != 1 || buf[0] != byte(1) {
		_ = c.Close()
		return nil, errors.New("server verification (of our certificate or IP address) failed")
	}

	return c, nil
}

func (lcs *LibvirtConnectionSettings) Connect() error {
	// ensure we're not already connected
	if lcs.Conn != nil && lcs.Conn.IsConnected() {
		return nil
	}

	switch lcs.Protocol {
	case LibvirtProtocolTCP:
		remote := dialers.NewRemote(lcs.Address, dialers.UsePort(strconv.Itoa(lcs.Port)), dialers.WithRemoteTimeout(2*time.Second))

		lcs.Conn = virt.NewWithDialer(remote)
		if err := lcs.Conn.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}
	case LibvirtProtocolTLS:
		keyFile, err := os.ReadFile(lcs.TLSClientKey)
		if err != nil {
			return err
		}

		certFile, err := os.ReadFile(lcs.TLSClientCert)
		if err != nil {
			return err
		}

		caFile, err := os.ReadFile(lcs.TLSCA)
		if err != nil {
			return err
		}

		cert, err := tls.X509KeyPair([]byte(certFile), []byte(keyFile))
		if err != nil {
			return err
		}

		roots := x509.NewCertPool()
		roots.AppendCertsFromPEM([]byte(caFile))

		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            roots,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}

		t := &TLS{
			timeout: LibvirtConnectionTimeoutSeconds * time.Second,
			conf:    tlsConfig,
			host:    lcs.Address,
			port:    strconv.Itoa(lcs.Port),
		}

		lcs.Conn = virt.NewWithDialer(t)
		if err := lcs.Conn.Connect(); err != nil {
			return err
		}
	}

	return nil
}

func (lcs *LibvirtConnectionSettings) Disconnect() error {
	klog.Infof("Disconnecting from libvirt ...")
	lcs.keepRunning = false
	if err := lcs.Conn.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect: %v", err)
	}
	return nil
}

func (lcs *LibvirtConnectionSettings) RegisterConnectionMonitor() {
	host := fmt.Sprintf("%s:%d", lcs.Address, lcs.Port)
	for {
		<-lcs.Conn.Disconnected()
		klog.Warningf("libvirt disconnection from %s has been detected", host)

		if !lcs.keepRunning {
			break
		}

		for {
			klog.Infof("Trying to reconnect to %s", host)
			err := lcs.Connect()
			if err != nil {
				klog.Error(err)
				time.Sleep(LibvirtConnectionTimeoutSeconds * time.Second)
				continue
			}
			klog.Infof("Successfully reconnected to libvirt on %s", host)
			break
		}
	}
}

func getGuestMachine(machines []virtxml.CapsGuestMachine, target string) string {
	for _, m := range machines {
		if m.Name == target {
			if m.Canonical != "" {
				return m.Canonical
			}
			return m.Name
		}
	}
	return ""
}

func getGuestMachineName(guest *virtxml.CapsGuest) (string, error) {
	/* Machine entries can be in the guest.Arch.Machines level as well as
	   under each guest.Arch.Domains[].Machines */
	target := guest.Arch.Machines[0].Name
	name := getGuestMachine(guest.Arch.Machines, target)
	if name != "" {
		return name, nil
	}
	for _, d := range guest.Arch.Domains {
		name := getGuestMachine(d.Machines, target)
		if name != "" {
			return name, nil
		}
	}

	return "", fmt.Errorf("can't find machine type %s in %v", target, guest)
}

func (lcs *LibvirtConnectionSettings) GetHostCapabilities() (virtxml.Caps, error) {
	caps := virtxml.Caps{}
	capsXML, err := lcs.Conn.Capabilities()
	if err != nil {
		return caps, err
	}

	err = xml.Unmarshal(capsXML, &caps)
	if err != nil {
		return caps, err
	}

	return caps, nil
}

func (lcs *LibvirtConnectionSettings) GetGuestCapabilities(caps virtxml.Caps) (string, string, error) {
	var guest *virtxml.CapsGuest
	for _, g := range caps.Guests {
		g := g // prevents implicit memory aliasing in for loop
		if g.Arch.Name == caps.Host.CPU.Arch && g.OSType == "hvm" {
			guest = &g
			break
		}
	}
	if guest == nil {
		return "", "", fmt.Errorf("can't find guest")
	}

	machineName, err := getGuestMachineName(guest)
	if err != nil {
		return "", "", err
	}

	return guest.Arch.Emulator, machineName, nil
}

func (lcs *LibvirtConnectionSettings) CreateInstance(instanceName, xml string) error {
	klog.Infof("Creating new virtual machine instance %s ...", instanceName)
	_, err := lcs.Conn.DomainDefineXML(xml)
	if err != nil {
		klog.Errorf("unable to create new virtual machine instance: %v", err)
		return err
	}

	return nil
}

func (lcs *LibvirtConnectionSettings) UpdateInstance(instanceName, xml string) error {
	klog.Infof("Updating existing virtual machine instance %s ...", instanceName)
	_, err := lcs.Conn.DomainDefineXML(xml)
	if err != nil {
		klog.Errorf("unable to update virtual machine instance: %v", err)
		return err
	}

	return nil
}

func (lcs *LibvirtConnectionSettings) DeleteInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	klog.Infof("Destroying virtual machine instance %s ...", instanceName)
	flags := virt.DomainUndefineNvram | virt.DomainUndefineSnapshotsMetadata | virt.DomainUndefineManagedSave | virt.DomainUndefineCheckpointsMetadata
	err = lcs.Conn.DomainUndefineFlags(*instance, flags)
	if err != nil {
		e := err.(virt.Error)
		if e.Code == uint32(virt.ErrNoSupport) || e.Code == uint32(virt.ErrInvalidArg) {
			// unsupported undefine flags: let's try again without flags
			err := lcs.Conn.DomainUndefine(*instance)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (lcs *LibvirtConnectionSettings) getInstance(instanceName string) (*virt.Domain, error) {
	domain, err := lcs.Conn.DomainLookupByName(instanceName)
	if err != nil {
		return nil, err
	}
	return &domain, nil
}

// Is instance running ?
func (lcs *LibvirtConnectionSettings) IsInstanceRunning(instanceName string) bool {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return false
	}

	state, _, err := lcs.Conn.DomainGetState(*instance, 0)
	if err != nil {
		return false
	}

	if state == int32(virt.DomainRunning) || state == int32(virt.DomainPaused) {
		return true
	}

	return false
}

func (lcs *LibvirtConnectionSettings) GetInstanceState(instanceName string) (string, string, error) {

	type VirtDomainState struct {
		State  string
		Reason map[int32]string
	}

	stateMap := map[int32]VirtDomainState{
		int32(virt.DomainNostate): {
			State: "No State",
			Reason: map[int32]string{
				int32(virt.DomainNostateUnknown): "Unknown",
			},
		},
		int32(virt.DomainRunning): {
			State: "Running",
			Reason: map[int32]string{
				int32(virt.DomainRunningUnknown):           "Unknown",
				int32(virt.DomainRunningBooted):            "Booted",
				int32(virt.DomainRunningMigrated):          "Migrated",
				int32(virt.DomainRunningRestored):          "Restored",
				int32(virt.DomainRunningFromSnapshot):      "From Snapshot",
				int32(virt.DomainRunningUnpaused):          "Unpaused",
				int32(virt.DomainRunningMigrationCanceled): "Migration Cancelled",
				int32(virt.DomainRunningSaveCanceled):      "Save Cancelled",
				int32(virt.DomainRunningWakeup):            "Wake Up",
				int32(virt.DomainRunningCrashed):           "Crashed",
				int32(virt.DomainRunningPostcopy):          "Post Copy",
			},
		},
		int32(virt.DomainBlocked): {
			State: "Blocked",
			Reason: map[int32]string{
				int32(virt.DomainBlockedUnknown): "Unknown",
			},
		},
		int32(virt.DomainPaused): {
			State: "Paused",
			Reason: map[int32]string{
				int32(virt.DomainPausedUnknown):        "Unknown",
				int32(virt.DomainPausedUser):           "User",
				int32(virt.DomainPausedMigration):      "Migration",
				int32(virt.DomainPausedSave):           "Save",
				int32(virt.DomainPausedDump):           "Dump",
				int32(virt.DomainPausedIoerror):        "I/O Error",
				int32(virt.DomainPausedWatchdog):       "Watchdog",
				int32(virt.DomainPausedFromSnapshot):   "From Snapshot",
				int32(virt.DomainPausedShuttingDown):   "Shutting Down",
				int32(virt.DomainPausedSnapshot):       "Snapshot",
				int32(virt.DomainPausedCrashed):        "Crashed",
				int32(virt.DomainPausedStartingUp):     "Starting Up",
				int32(virt.DomainPausedPostcopy):       "Post Copy",
				int32(virt.DomainPausedPostcopyFailed): "Post Copy Failed",
			},
		},
		int32(virt.DomainShutdown): {
			State: "Shutdown",
			Reason: map[int32]string{
				int32(virt.DomainShutdownUnknown): "Unknown",
				int32(virt.DomainShutdownUser):    "User",
			},
		},
		int32(virt.DomainShutoff): {
			State: "Shutoff",
			Reason: map[int32]string{
				int32(virt.DomainShutoffUnknown):      "Unknown",
				int32(virt.DomainShutoffShutdown):     "Shutdown",
				int32(virt.DomainShutoffDestroyed):    "Destroyed",
				int32(virt.DomainShutoffCrashed):      "Crashed",
				int32(virt.DomainShutoffMigrated):     "Migrated",
				int32(virt.DomainShutoffSaved):        "Saved",
				int32(virt.DomainShutoffFailed):       "Failed",
				int32(virt.DomainShutoffFromSnapshot): "From Snapshot",
				int32(virt.DomainShutoffDaemon):       "Daemon",
			},
		},
		int32(virt.DomainCrashed): {
			State: "Crashed",
			Reason: map[int32]string{
				int32(virt.DomainCrashedUnknown):  "Unknown",
				int32(virt.DomainCrashedPanicked): "Panicked",
			},
		},
		int32(virt.DomainPmsuspended): {
			State: "PM Suspended",
			Reason: map[int32]string{
				int32(virt.DomainPmsuspendedUnknown): "Unknown",
			},
		},
	}

	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return "", "", err
	}

	state, reason, err := lcs.Conn.DomainGetState(*instance, 0)
	if err != nil {
		return "", "", fmt.Errorf("unable to get instance state: %v", err)
	}

	sm := stateMap[state]
	return sm.State, sm.Reason[reason], nil
}

func (lcs *LibvirtConnectionSettings) GetInstanceDescription(instanceName string, migratable bool) (string, error) {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return "", err
	}

	var flags virt.DomainXMLFlags = 0
	if migratable {
		flags = virt.DomainXMLMigratable
	}

	return lcs.Conn.DomainGetXMLDesc(*instance, flags)
}

func (lcs *LibvirtConnectionSettings) GetInstanceRemoteConnectionUrl(instanceName string) (string, error) {

	port := 0

	xml, err := lcs.GetInstanceDescription(instanceName, false)
	if err != nil {
		return "", fmt.Errorf("unable to get instance XML description: %v", err)
	}

	d := virtxml.Domain{}
	err = xmlUnmarshal(xml, &d)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal instance XML description: %v", err)
	}

	graphics := d.Devices.Graphics
	for _, g := range graphics {
		if g.Spice != nil && g.Spice.Port != 0 {
			port = g.Spice.Port
			break
		}
	}

	return fmt.Sprintf("spice://%s:%d", lcs.Address, port), nil
}

// Software OS reboot
func (lcs *LibvirtConnectionSettings) RebootInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	klog.Infof("Rebooting instance %s ...", instance.Name)
	return lcs.Conn.DomainReboot(*instance, virt.DomainRebootDefault)
}

// Hardware Reset
func (lcs *LibvirtConnectionSettings) ResetInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	klog.Infof("Hardware reset of instance %s ...", instance.Name)
	return lcs.Conn.DomainReset(*instance, 0)
}

// Software PM Suspend
func (lcs *LibvirtConnectionSettings) SuspendInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	klog.Infof("Suspending instance %s ...", instance.Name)
	return lcs.Conn.DomainSuspend(*instance)
}

// Software PM Resume
func (lcs *LibvirtConnectionSettings) ResumeInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	klog.Infof("Resuming instance %s ...", instance.Name)
	return lcs.Conn.DomainResume(*instance)
}

// Enable auto-start
func (lcs *LibvirtConnectionSettings) AutoStartInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	err = lcs.Conn.DomainSetAutostart(*instance, 1)
	if err != nil {
		return err
	}

	return lcs.StartInstance(instanceName)
}

// Hardware Boot
func (lcs *LibvirtConnectionSettings) StartInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	klog.Infof("Starting instance %s ...", instance.Name)
	return lcs.Conn.DomainCreate(*instance)
}

// Hardware Shutdown
func (lcs *LibvirtConnectionSettings) StopInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	if lcs.IsInstanceRunning(instanceName) {
		klog.Infof("Stopping instance %s ...", instance.Name)
		return lcs.Conn.DomainDestroy(*instance)
	}

	return nil
}

// Software Shutdown
func (lcs *LibvirtConnectionSettings) ShutdownInstance(instanceName string) error {
	instance, err := lcs.getInstance(instanceName)
	if err != nil {
		return err
	}

	if lcs.IsInstanceRunning(instanceName) {
		klog.Infof("Shuting down instance %s ...", instance.Name)
		return lcs.Conn.DomainShutdown(*instance)
	}

	return nil
}

func xmlUnmarshal(input string, v any) error {
	return xml.Unmarshal([]byte(input), v)
}
