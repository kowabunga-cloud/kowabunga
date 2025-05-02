/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

type Kaktus struct {
	agent *KaktusAgent
}

func newKaktus(agent *KaktusAgent) *Kaktus {
	return &Kaktus{
		agent: agent,
	}
}

/*
 * RPC Capabilities()
 */

func (k *Kaktus) Capabilities(args *agents.CapabilitiesArgs, reply *agents.CapabilitiesReply) error {
	*reply = agents.CapabilitiesReply{
		Version: KaktusVersion,
		Methods: k.agent.RpcServer().GetServices(),
	}
	return nil
}

/*
 * RPC NodeCapabilities()
 */

type KaktusNodeCapabilitiesArgs struct{}
type KaktusNodeCapabilitiesReply struct {
	Arch             string
	Vendor           string
	Model            string
	Sockets          uint8
	Cores            uint16
	Threads          uint16
	Memory           uint64
	GuestEmulator    string
	GuestMachineName string
}

func (k *Kaktus) NodeCapabilities(args *KaktusNodeCapabilitiesArgs, reply *KaktusNodeCapabilitiesReply) error {

	caps, err := k.agent.lcs.GetHostCapabilities()
	if err != nil {
		return err
	}

	var memory uint64
	sockets := make(map[string]bool)
	cores := make(map[string]bool)
	threads := make(map[string]bool)
	for _, c := range caps.Host.NUMA.Cells.Cells {
		if c.Memory != nil {
			// some NUMA cells might have no memory attached
			mem := uint64(c.Memory.Size)
			switch c.Memory.Unit {
			case "KiB":
				mem *= common.KiB
			case "MiB":
				mem *= common.MiB
			}
			memory += mem
		}
		for _, cc := range c.CPUS.CPUs {
			sockets[fmt.Sprintf("%d", *cc.SocketID)] = true
			cores[fmt.Sprintf("%d", *cc.CoreID)] = true
			threads[fmt.Sprintf("%d", cc.ID)] = true
		}
	}
	klog.Infof("Detected host memory: %d GB", memory/common.GiB)
	klog.Infof("Detected host CPU %d socket(s), %d cores, %d threads", len(sockets), len(cores), len(threads))

	emulator, machine, err := k.agent.lcs.GetGuestCapabilities(caps)
	if err != nil {
		return err
	}

	*reply = KaktusNodeCapabilitiesReply{
		Arch:             caps.Host.CPU.Arch,
		Vendor:           caps.Host.CPU.Vendor,
		Model:            caps.Host.CPU.Model,
		Sockets:          uint8(len(sockets)),
		Cores:            uint16(len(cores)),
		Threads:          uint16(len(threads)),
		Memory:           memory,
		GuestEmulator:    emulator,
		GuestMachineName: machine,
	}

	return nil
}

/*
 * RPC CreateInstance()
 */

type KaktusCreateInstanceArgs struct {
	Name string
	XML  string
}
type KaktusCreateInstanceReply struct{}

func (k *Kaktus) CreateInstance(args *KaktusCreateInstanceArgs, reply *KaktusCreateInstanceReply) error {
	err := k.agent.lcs.CreateInstance(args.Name, args.XML)
	*reply = KaktusCreateInstanceReply{}
	return err
}

/*
 * RPC GetInstance()
 */

type KaktusGetInstanceArgs struct {
	Name       string
	Migratable bool
}

type KaktusGetInstanceReply struct {
	XML string
}

func (k *Kaktus) GetInstance(args *KaktusGetInstanceArgs, reply *KaktusGetInstanceReply) error {
	xml, err := k.agent.lcs.GetInstanceDescription(args.Name, args.Migratable)
	*reply = KaktusGetInstanceReply{
		XML: xml,
	}
	return err
}

/*
 * RPC DeleteInstance()
 */

type KaktusDeleteInstanceArgs struct {
	Name string
}

type KaktusDeleteInstanceReply struct{}

func (k *Kaktus) DeleteInstance(args *KaktusDeleteInstanceArgs, reply *KaktusDeleteInstanceReply) error {
	err := k.agent.lcs.DeleteInstance(args.Name)
	*reply = KaktusDeleteInstanceReply{}
	return err
}

/*
 * RPC UpdateInstance()
 */

type KaktusUpdateInstanceArgs struct {
	Name string
	XML  string
}

type KaktusUpdateInstanceReply struct{}

func (k *Kaktus) UpdateInstance(args *KaktusUpdateInstanceArgs, reply *KaktusUpdateInstanceReply) error {
	err := k.agent.lcs.UpdateInstance(args.Name, args.XML)
	*reply = KaktusUpdateInstanceReply{}
	return err
}

/*
 * RPC GetInstanceState()
 */

type KaktusGetInstanceStateArgs struct {
	Name string
}

type KaktusGetInstanceStateReply struct {
	State  string
	Reason string
}

func (k *Kaktus) GetInstanceState(args *KaktusGetInstanceStateArgs, reply *KaktusGetInstanceStateReply) error {
	state, reason, err := k.agent.lcs.GetInstanceState(args.Name)
	*reply = KaktusGetInstanceStateReply{
		State:  state,
		Reason: reason,
	}
	return err
}

/*
 * RPC InstanceIsRunning()
 */

type KaktusInstanceIsRunningArgs struct {
	Name string
}

type KaktusInstanceIsRunningReply struct {
	Running bool
}

func (k *Kaktus) InstanceIsRunning(args *KaktusInstanceIsRunningArgs, reply *KaktusInstanceIsRunningReply) error {
	state := k.agent.lcs.IsInstanceRunning(args.Name)
	*reply = KaktusInstanceIsRunningReply{
		Running: state,
	}
	return nil
}

/*
 * RPC GetInstanceRemoteConnectionUrl()
 */

type KaktusGetInstanceRemoteConnectionUrlArgs struct {
	Name string
}

type KaktusGetInstanceRemoteConnectionUrlReply struct {
	URL string
}

func (k *Kaktus) GetInstanceRemoteConnectionUrl(args *KaktusGetInstanceRemoteConnectionUrlArgs, reply *KaktusGetInstanceRemoteConnectionUrlReply) error {
	url, err := k.agent.lcs.GetInstanceRemoteConnectionUrl(args.Name)
	*reply = KaktusGetInstanceRemoteConnectionUrlReply{
		URL: url,
	}
	return err
}

/*
 * RPC InstanceOperation()
 */

type KaktusInstanceOperationArgs struct {
	Name   string
	Action KaktusInstanceOperation
}

type KaktusInstanceOperationReply struct{}

type KaktusInstanceOperation int

const (
	KaktusInstanceOpAutoStart KaktusInstanceOperation = iota
	KaktusInstanceOpStart
	KaktusInstanceOpSoftReboot
	KaktusInstanceOpHardReboot
	KaktusInstanceOpHardShutdown
	KaktusInstanceOpSoftShutdown
	KaktusInstanceOpPmSuspend
	KaktusInstanceOpPmResume
)

func (k *Kaktus) InstanceOperation(args *KaktusInstanceOperationArgs, reply *KaktusInstanceOperationReply) error {
	*reply = KaktusInstanceOperationReply{}

	switch args.Action {
	case KaktusInstanceOpAutoStart:
		return k.agent.lcs.AutoStartInstance(args.Name)
	case KaktusInstanceOpStart:
		return k.agent.lcs.StartInstance(args.Name)
	case KaktusInstanceOpSoftReboot:
		return k.agent.lcs.RebootInstance(args.Name)
	case KaktusInstanceOpHardReboot:
		return k.agent.lcs.ResetInstance(args.Name)
	case KaktusInstanceOpSoftShutdown:
		return k.agent.lcs.ShutdownInstance(args.Name)
	case KaktusInstanceOpHardShutdown:
		return k.agent.lcs.StopInstance(args.Name)
	case KaktusInstanceOpPmSuspend:
		return k.agent.lcs.SuspendInstance(args.Name)
	case KaktusInstanceOpPmResume:
		return k.agent.lcs.ResumeInstance(args.Name)
	}

	return fmt.Errorf("unsupported instance operation: %d", args.Action)
}

/*
 * RPC GetStoragePoolStats()
 */

type KaktusGetStoragePoolStatsArgs struct {
	Pool string
}

type KaktusGetStoragePoolStatsReply struct {
	Allocated uint64
	Available uint64
	Capacity  uint64
}

func (k *Kaktus) GetStoragePoolStats(args *KaktusGetStoragePoolStatsArgs, reply *KaktusGetStoragePoolStatsReply) error {
	used, available, total, err := k.agent.ceph.GetPoolStats(args.Pool)
	*reply = KaktusGetStoragePoolStatsReply{
		Allocated: used,
		Available: available,
		Capacity:  total,
	}
	return err
}

/*
 * RPC CreateRawVolume()
 */

type KaktusCreateRawVolumeArgs struct {
	Pool   string
	Volume string
	Size   uint64
}

type KaktusCreateRawVolumeReply struct{}

func (k *Kaktus) CreateRawVolume(args *KaktusCreateRawVolumeArgs, reply *KaktusCreateRawVolumeReply) error {
	err := k.agent.ceph.CreateRbdVolume(args.Pool, args.Volume, args.Size)
	*reply = KaktusCreateRawVolumeReply{}
	return err
}

/*
 * RPC CreateTemplateVolume()
 */

type KaktusCreateTemplateVolumeArgs struct {
	Pool      string
	Volume    string
	SourceURL string
}

type KaktusCreateTemplateVolumeReply struct {
	Size uint64
}

func (k *Kaktus) CreateTemplateVolume(args *KaktusCreateTemplateVolumeArgs, reply *KaktusCreateTemplateVolumeReply) error {
	size, err := k.agent.ceph.CreateRbdVolumeFromUrl(args.Pool, args.Volume, args.SourceURL)
	*reply = KaktusCreateTemplateVolumeReply{
		Size: size,
	}
	return err
}

/*
 * RPC CreateOsVolume()
 */

type KaktusCreateOsVolumeArgs struct {
	Pool     string
	Volume   string
	Size     uint64
	Template string
}

type KaktusCreateOsVolumeReply struct{}

func (k *Kaktus) CreateOsVolume(args *KaktusCreateOsVolumeArgs, reply *KaktusCreateOsVolumeReply) error {
	err := k.agent.ceph.CloneRbdVolume(args.Pool, args.Template, args.Volume, args.Size)
	*reply = KaktusCreateOsVolumeReply{}
	return err
}

/*
 * RPC CreateIsoVolume()
 */

type KaktusCreateIsoVolumeArgs struct {
	Pool    string
	Volume  string
	Size    uint64
	Content []byte
}

type KaktusCreateIsoVolumeReply struct{}

func (k *Kaktus) CreateIsoVolume(args *KaktusCreateIsoVolumeArgs, reply *KaktusCreateIsoVolumeReply) error {
	err := k.agent.ceph.CreateRbdVolumeFromBinData(args.Pool, args.Volume, args.Size, args.Content)
	*reply = KaktusCreateIsoVolumeReply{}
	return err
}

/*
 * RPC UpdateIsoVolume()
 */

type KaktusUpdateIsoVolumeArgs struct {
	Pool    string
	Volume  string
	Size    uint64
	Content []byte
}

type KaktusUpdateIsoVolumeReply struct{}

func (k *Kaktus) UpdateIsoVolume(args *KaktusUpdateIsoVolumeArgs, reply *KaktusUpdateIsoVolumeReply) error {
	err := k.agent.ceph.UpdateRbdVolumeFromBinData(args.Pool, args.Volume, args.Size, args.Content)
	*reply = KaktusUpdateIsoVolumeReply{}
	return err
}

/*
 * RPC GetVolumeInfos()
 */

type KaktusGetVolumeInfosArgs struct {
	Pool   string
	Volume string
}

type KaktusGetVolumeInfosReply struct {
	Size uint64
}

func (k *Kaktus) GetVolumeInfos(args *KaktusGetVolumeInfosArgs, reply *KaktusGetVolumeInfosReply) error {
	size, err := k.agent.ceph.GetRbdVolumeInfos(args.Pool, args.Volume)
	*reply = KaktusGetVolumeInfosReply{
		Size: size,
	}
	return err
}

/*
 * RPC ResizeVolume()
 */

type KaktusResizeVolumeArgs struct {
	Pool   string
	Volume string
	Size   uint64
}

type KaktusResizeVolumeReply struct{}

func (k *Kaktus) ResizeVolume(args *KaktusResizeVolumeArgs, reply *KaktusResizeVolumeReply) error {
	err := k.agent.ceph.ResizeRbdVolume(args.Pool, args.Volume, args.Size)
	*reply = KaktusResizeVolumeReply{}
	return err
}

/*
 * RPC DeleteVolume()
 */

type KaktusDeleteVolumeArgs struct {
	Pool          string
	Volume        string
	WithSnapshots bool
}

type KaktusDeleteVolumeReply struct{}

func (k *Kaktus) DeleteVolume(args *KaktusDeleteVolumeArgs, reply *KaktusDeleteVolumeReply) error {
	err := k.agent.ceph.DeleteRbdVolume(args.Pool, args.Volume, args.WithSnapshots)
	*reply = KaktusDeleteVolumeReply{}
	return err
}

// CephFS

/*
 * RPC ListFileSystems()
 */

type KaktusListFileSystemsArgs struct{}
type KaktusListFileSystemsReply struct {
	FS []string
}

func (k *Kaktus) ListFileSystems(args *KaktusListFileSystemsArgs, reply *KaktusListFileSystemsReply) error {
	volumes, err := k.agent.ceph.ListVolumes()
	*reply = KaktusListFileSystemsReply{
		FS: volumes,
	}
	return err
}

/*
 * RPC ListFsSubVolumes()
 */

type KaktusListFsSubVolumesArgs struct {
	FS string
}

type KaktusListFsSubVolumesReply struct {
	SubVolumes []string
}

func (k *Kaktus) ListFsSubVolumes(args *KaktusListFsSubVolumesArgs, reply *KaktusListFsSubVolumesReply) error {
	subvolumes, err := k.agent.ceph.ListSubVolumes(args.FS)
	*reply = KaktusListFsSubVolumesReply{
		SubVolumes: subvolumes,
	}
	return err
}

/*
 * RPC CreateFsSubVolume()
 */

type KaktusCreateFsSubVolumeArgs struct {
	FS        string
	SubVolume string
}

type KaktusCreateFsSubVolumeReply struct {
	Path      string
	BytesUsed int64
}

func (k *Kaktus) CreateFsSubVolume(args *KaktusCreateFsSubVolumeArgs, reply *KaktusCreateFsSubVolumeReply) error {
	path, size, err := k.agent.ceph.CreateSubVolume(args.FS, args.SubVolume)
	*reply = KaktusCreateFsSubVolumeReply{
		Path:      path,
		BytesUsed: size,
	}
	return err
}

/*
 * RPC DeleteFsSubVolume()
 */

type KaktusDeleteFsSubVolumeArgs struct {
	FS        string
	SubVolume string
}
type KaktusDeleteFsSubVolumeReply struct{}

func (k *Kaktus) DeleteFsSubVolume(args *KaktusDeleteFsSubVolumeArgs, reply *KaktusDeleteFsSubVolumeReply) error {
	err := k.agent.ceph.DeleteSubVolume(args.FS, args.SubVolume)
	*reply = KaktusDeleteFsSubVolumeReply{}
	return err
}

// NFS

/*
 * RPC CreateNfsBackends()
 */

type KaktusCreateNfsBackendsArgs struct {
	ID        string
	Name      string
	FS        string
	Path      string
	Access    string
	Protocols []int32
	Clients   []string
	Backends  []string
	Port      int
}
type KaktusCreateNfsBackendsReply struct{}

func (k *Kaktus) CreateNfsBackends(args *KaktusCreateNfsBackendsArgs, reply *KaktusCreateNfsBackendsReply) error {
	err := k.agent.nfs.CreateBackends(args.ID, args.Name, args.FS, args.Path, args.Access, args.Protocols, args.Clients, args.Backends, args.Port)
	*reply = KaktusCreateNfsBackendsReply{}
	return err
}

/*
 * RPC UpdateNfsBackends()
 */

type KaktusUpdateNfsBackendsArgs struct {
	ID        string
	Name      string
	FS        string
	Path      string
	Access    string
	Protocols []int32
	Clients   []string
	Backends  []string
	Port      int
}
type KaktusUpdateNfsBackendsReply struct{}

func (k *Kaktus) UpdateNfsBackends(args *KaktusUpdateNfsBackendsArgs, reply *KaktusUpdateNfsBackendsReply) error {
	err := k.agent.nfs.UpdateBackends(args.ID, args.Name, args.FS, args.Path, args.Access, args.Protocols, args.Clients, args.Backends, args.Port)
	*reply = KaktusUpdateNfsBackendsReply{}
	return err
}

/*
 * RPC DeleteNfsBackends()
 */

type KaktusDeleteNfsBackendsArgs struct {
	ID        string
	Name      string
	FS        string
	Path      string
	Access    string
	Protocols []int32
	Clients   []string
	Backends  []string
	Port      int
}
type KaktusDeleteNfsBackendsReply struct{}

func (k *Kaktus) DeleteNfsBackends(args *KaktusDeleteNfsBackendsArgs, reply *KaktusDeleteNfsBackendsReply) error {
	err := k.agent.nfs.DeleteBackends(args.ID, args.Name, args.FS, args.Path, args.Access, args.Protocols, args.Clients, args.Backends, args.Port)
	*reply = KaktusDeleteNfsBackendsReply{}
	return err
}
