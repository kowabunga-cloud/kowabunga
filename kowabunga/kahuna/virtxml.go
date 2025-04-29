/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"sort"

	virtxml "libvirt.org/go/libvirtxml"
)

type VirtualInstanceDescription struct {
	domain *virtxml.Domain
}

func (desc *VirtualInstanceDescription) XML() (string, error) {
	return XmlMarshal(desc.domain)
}

func (desc *VirtualInstanceDescription) SetCPU(cpuCount int64) {
	desc.domain.VCPU.Value = uint(cpuCount)
}

func (desc *VirtualInstanceDescription) SetMemory(memBytes int64) {
	desc.domain.CurrentMemory.Unit = "B"
	desc.domain.CurrentMemory.Value = uint(memBytes)
	desc.domain.Memory.Unit = "B"
	desc.domain.Memory.Value = uint(memBytes)
}

func (desc *VirtualInstanceDescription) SetInterfaces(interfaces map[string]string) {
	desc.domain.Devices.Interfaces = newVirtualInterfaces(interfaces)
}

func (desc *VirtualInstanceDescription) SetDisks(disks map[string]string, cloudInitVolumeId string) {
	desc.domain.Devices.Disks = newVirtualDisks(disks, cloudInitVolumeId)
}

var ptySerialPort uint = 0
var ptyVirtioPort uint = 1

func virtOsType(arch, machine string) *virtxml.DomainOSType {
	return &virtxml.DomainOSType{
		Arch:    arch,
		Machine: machine,
		Type:    "hvm",
	}
}

func virtBootDevices() []virtxml.DomainBootDevice {
	return []virtxml.DomainBootDevice{
		{
			Dev: "hd",
		},
	}
}

func virtOs(arch, machine string) *virtxml.DomainOS {
	return &virtxml.DomainOS{
		Type:        virtOsType(arch, machine),
		BootDevices: virtBootDevices(),
	}
}

func virtMemory(mem int64) *virtxml.DomainMemory {
	return &virtxml.DomainMemory{
		Unit:  "B",
		Value: uint(mem),
	}
}

func virtVcpus(cpu int64) *virtxml.DomainVCPU {
	return &virtxml.DomainVCPU{
		Placement: "static",
		Value:     uint(cpu),
	}
}

func virtClock() *virtxml.DomainClock {
	return &virtxml.DomainClock{
		Offset: "utc",
	}
}

func virtCPU() *virtxml.DomainCPU {
	return &virtxml.DomainCPU{
		Mode:       "host-passthrough",
		Check:      "none",
		Migratable: "on",
		Model: &virtxml.DomainCPUModel{
			Fallback: "allow",
		},
	}
}

func virtFeatures() *virtxml.DomainFeatureList {
	return &virtxml.DomainFeatureList{
		PAE:  &virtxml.DomainFeature{},
		ACPI: &virtxml.DomainFeature{},
		APIC: &virtxml.DomainFeatureAPIC{},
	}
}

func virtWindowsFeatureHyperV() *virtxml.DomainFeatureHyperV {
	return &virtxml.DomainFeatureHyperV{
		Relaxed: &virtxml.DomainFeatureState{
			State: "on",
		},
		VAPIC: &virtxml.DomainFeatureState{
			State: "on",
		},
		Spinlocks: &virtxml.DomainFeatureHyperVSpinlocks{
			Retries: 8191,
		},
	}
}

func virtInputDevices() []virtxml.DomainInput {
	return []virtxml.DomainInput{
		{
			Type: "mouse",
			Bus:  "ps2",
		},
		{
			Type: "keyboard",
			Bus:  "ps2",
		},
	}
}

func virtGraphicDevices() []virtxml.DomainGraphic {
	return []virtxml.DomainGraphic{
		{
			Spice: &virtxml.DomainGraphicSpice{
				AutoPort: "yes",
				Listeners: []virtxml.DomainGraphicListener{
					{
						Address: &virtxml.DomainGraphicListenerAddress{
							Address: "0.0.0.0",
						},
					},
				},
			},
		},
	}
}

func virtAudioDevices() []virtxml.DomainAudio {
	return []virtxml.DomainAudio{
		{
			ID:    1,
			SPICE: &virtxml.DomainAudioSPICE{},
		},
	}
}

func virtWindowsSoundDevices() []virtxml.DomainSound {
	return []virtxml.DomainSound{
		{
			Model: "ich9",
		},
	}
}

func virtLinuxVideoDevices() []virtxml.DomainVideo {
	return []virtxml.DomainVideo{
		{
			Model: virtxml.DomainVideoModel{
				Type:    "cirrus",
				VRam:    16384,
				Heads:   1,
				Primary: "yes",
			},
		},
	}
}

func virtWindowsVideoDevices() []virtxml.DomainVideo {
	return []virtxml.DomainVideo{
		{
			Model: virtxml.DomainVideoModel{
				Type:    "qxl",
				Ram:     65536,
				VRam:    65536,
				VGAMem:  65536,
				Heads:   1,
				Primary: "yes",
			},
		},
	}
}

func virtSerialDevices() []virtxml.DomainSerial {
	return []virtxml.DomainSerial{
		{
			Target: &virtxml.DomainSerialTarget{
				Type: "isa-serial",
				Port: &ptySerialPort,
				Model: &virtxml.DomainSerialTargetModel{
					Name: "isa-serial",
				},
			},
		},
	}
}

func virtConsoleDevice(mode string, port *uint) virtxml.DomainConsole {
	return virtxml.DomainConsole{
		Source: &virtxml.DomainChardevSource{
			Pty: &virtxml.DomainChardevSourcePty{
				Path: "pty",
			},
		},
		Target: &virtxml.DomainConsoleTarget{
			Type: mode,
			Port: port,
		},
	}
}

func virtConsoleDevices() []virtxml.DomainConsole {
	return []virtxml.DomainConsole{
		virtConsoleDevice("serial", &ptySerialPort),
		virtConsoleDevice("virtio", &ptyVirtioPort),
	}
}
func virtMemBallonDevice() *virtxml.DomainMemBalloon {
	return &virtxml.DomainMemBalloon{
		Model: "virtio",
	}
}

func virtDevice(emulator string) *virtxml.DomainDeviceList {
	return &virtxml.DomainDeviceList{
		Emulator:   emulator,
		Inputs:     virtInputDevices(),
		Graphics:   virtGraphicDevices(),
		Audios:     virtAudioDevices(),
		Serials:    virtSerialDevices(),
		Consoles:   virtConsoleDevices(),
		MemBalloon: virtMemBallonDevice(),
	}
}

func virtLinuxChannelDevices() []virtxml.DomainChannel {
	return []virtxml.DomainChannel{
		{
			Source: &virtxml.DomainChardevSource{
				UNIX: &virtxml.DomainChardevSourceUNIX{},
			},
			Target: &virtxml.DomainChannelTarget{
				VirtIO: &virtxml.DomainChannelTargetVirtIO{
					Name: "org.qemu.guest_agent.0",
				},
			},
		},
	}
}

func virtWindowsChannelDevices() []virtxml.DomainChannel {
	return []virtxml.DomainChannel{
		{
			Source: &virtxml.DomainChardevSource{
				SpiceVMC: &virtxml.DomainChardevSourceSpiceVMC{},
			},
			Target: &virtxml.DomainChannelTarget{
				VirtIO: &virtxml.DomainChannelTargetVirtIO{
					Name: "com.redhat.spice.0",
				},
			},
		},
	}
}

func virtRngDevices() []virtxml.DomainRNG {
	return []virtxml.DomainRNG{
		{
			Model: "virtio",
			Backend: &virtxml.DomainRNGBackend{
				Random: &virtxml.DomainRNGBackendRandom{
					Device: "/dev/urandom",
				},
			},
		},
	}
}

func virtRedirDevices() []virtxml.DomainRedirDev {
	return []virtxml.DomainRedirDev{
		{
			Bus: "usb",
			Source: &virtxml.DomainChardevSource{
				SpiceVMC: &virtxml.DomainChardevSourceSpiceVMC{},
			},
		},
	}
}

func virtWindowsTimers() []virtxml.DomainTimer {
	return []virtxml.DomainTimer{
		{
			Name:       "rtc",
			TickPolicy: "catchup",
		},
		{
			Name:       "pit",
			TickPolicy: "delay",
		},
		{
			Name:    "hpet",
			Present: "no",
		},
		{
			Name:    "hypervclock",
			Present: "yes",
		},
	}
}

func virtWindowsPM() *virtxml.DomainPM {
	return &virtxml.DomainPM{
		SuspendToMem: &virtxml.DomainPMPolicy{
			Enabled: "no",
		},
		SuspendToDisk: &virtxml.DomainPMPolicy{
			Enabled: "no",
		},
	}
}

func NewVirtualInstanceDescription(os, name, desc, arch, machine, emulator string, memory, vcpus int64) *VirtualInstanceDescription {
	d := &virtxml.Domain{
		Name:        name,
		Description: desc,
		Type:        "kvm",
		OS:          virtOs(arch, machine),
		Memory:      virtMemory(memory),
		VCPU:        virtVcpus(vcpus),
		Clock:       virtClock(),
		CPU:         virtCPU(),
		OnPoweroff:  "destroy",
		OnReboot:    "restart",
		OnCrash:     "destroy",
		Features:    virtFeatures(),
		Devices:     virtDevice(emulator),
	}

	switch os {
	case TemplateOsLinux:
		// Linux-instance specifics
		d.Devices.Channels = virtLinuxChannelDevices()
		d.Devices.RNGs = virtRngDevices()
		d.Devices.Videos = virtLinuxVideoDevices()
	case TemplateOsWindows:
		// Windows-instance specifics
		d.Features.HyperV = virtWindowsFeatureHyperV()
		d.Features.HyperV.Spinlocks.State = "on"
		d.Clock.Timer = virtWindowsTimers()
		d.PM = virtWindowsPM()
		d.Devices.Channels = virtWindowsChannelDevices()
		d.Devices.Sounds = virtWindowsSoundDevices()
		d.Devices.Videos = virtWindowsVideoDevices()
		d.Devices.RedirDevs = virtRedirDevices()
	}

	instance := VirtualInstanceDescription{
		domain: d,
	}

	return &instance
}

func NewVirtualInstanceFromXml(xml string) (*VirtualInstanceDescription, error) {
	d := virtxml.Domain{}
	err := XmlUnmarshal(xml, &d)
	if err != nil {
		return nil, err
	}

	instance := VirtualInstanceDescription{
		domain: &d,
	}

	return &instance, nil
}

func VirtualInstanceToXml(desc *VirtualInstanceDescription) (string, error) {
	return XmlMarshal(desc.domain)
}

func virtInterface(address, iface string) virtxml.DomainInterface {
	return virtxml.DomainInterface{
		Model: &virtxml.DomainInterfaceModel{
			Type: "virtio",
		},
		MAC: &virtxml.DomainInterfaceMAC{
			Address: address,
		},
		Source: &virtxml.DomainInterfaceSource{
			Bridge: &virtxml.DomainInterfaceSourceBridge{
				Bridge: iface,
			},
		},
	}
}

func newVirtualInterfaces(interfaces map[string]string) []virtxml.DomainInterface {
	ifaces := []virtxml.DomainInterface{}

	// ensure we sort out interfaces by name, reflecting correct insertion order and appropriate XML generation
	keys := make([]string, 0, len(interfaces))
	for k := range interfaces {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// process adapters, if any
	for _, k := range keys {
		adapterId, ok := interfaces[k]
		if !ok {
			continue
		}

		a, err := FindAdapterByID(adapterId)
		if err != nil {
			continue
		}

		s, err := a.Subnet()
		if err != nil {
			continue
		}

		v, err := s.VNet()
		if err != nil {
			continue
		}

		iface := virtInterface(a.MAC, v.Interface)
		ifaces = append(ifaces, iface)
	}

	return ifaces
}

func newVirtualDisks(disks map[string]string, cloudInitVolumeId string) []virtxml.DomainDisk {
	disksList := []virtxml.DomainDisk{}

	// copy existing disks map
	disksMap := make(map[string]string)
	for dev, volumeId := range disks {
		disksMap[dev] = volumeId
	}

	// add cloud-init volume to list, if any
	if cloudInitVolumeId != "" {
		disksMap[VolumeCloudInitDisk] = cloudInitVolumeId
	}

	// ensure we sort out disks by name, reflecting correct insertion order and appropriate XML generation
	keys := make([]string, 0, len(disksMap))
	for k := range disksMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, dev := range keys {
		volumeId, ok := disksMap[dev]
		if !ok {
			continue
		}

		v, err := FindVolumeByID(volumeId)
		if err != nil {
			continue
		}

		p, err := v.StoragePool()
		if err != nil {
			continue
		}

		// create libvirt disk
		d := NewVirtualDisk(v.Type, p.Pool, dev, v.Name, p.Address, p.Auth, p.Port)
		disksList = append(disksList, d)
	}

	return disksList
}

func virtDisk(device string) virtxml.DomainDisk {
	return virtxml.DomainDisk{
		Device: "disk",
		Target: &virtxml.DomainDiskTarget{
			Bus: "virtio",
			Dev: device,
		},
		Driver: &virtxml.DomainDiskDriver{
			Name: "qemu",
			Type: "raw",
		},
	}
}

func virtDiskNetworkSource(srv, pool, volume string, port int) *virtxml.DomainDiskSource {
	return &virtxml.DomainDiskSource{
		Network: &virtxml.DomainDiskSourceNetwork{
			Protocol: "rbd",
			Name:     pool + "/" + volume,
			Hosts: []virtxml.DomainDiskSourceHost{
				{
					Name: srv,
					Port: fmt.Sprintf("%d", port),
				},
			},
		},
	}
}

func virtDiskSecret(uuid string) *virtxml.DomainDiskAuth {
	return &virtxml.DomainDiskAuth{
		Username: "libvirt",
		Secret: &virtxml.DomainDiskSecret{
			Type: "ceph",
			UUID: uuid,
		},
	}
}

func NewVirtualDisk(vType, pool, device, name, address, auth string, port int) virtxml.DomainDisk {
	disk := virtDisk(device)
	disk.Source = virtDiskNetworkSource(address, pool, name, port)
	disk.Auth = virtDiskSecret(auth)

	switch vType {
	case VolumeTypeIso:
		disk.Device = "cdrom"
		disk.Target.Bus = "ide"
		disk.ReadOnly = &virtxml.DomainDiskReadOnly{}
		disk.Serial = "cloudinit"
	default:
		disk.Driver.Cache = "writeback"
	}

	return disk
}
