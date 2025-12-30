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
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/proto"
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

func (k *Kaktus) NodeCapabilities(args *proto.KaktusNodeCapabilitiesArgs, reply *proto.KaktusNodeCapabilitiesReply) error {

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

	*reply = proto.KaktusNodeCapabilitiesReply{
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

func (k *Kaktus) CreateInstance(args *proto.KaktusCreateInstanceArgs, reply *proto.KaktusCreateInstanceReply) error {
	err := k.agent.lcs.CreateInstance(args.Name, args.XML)
	*reply = proto.KaktusCreateInstanceReply{}
	return err
}

/*
 * RPC GetInstance()
 */

func (k *Kaktus) GetInstance(args *proto.KaktusGetInstanceArgs, reply *proto.KaktusGetInstanceReply) error {
	xml, err := k.agent.lcs.GetInstanceDescription(args.Name, args.Migratable)
	*reply = proto.KaktusGetInstanceReply{
		XML: xml,
	}
	return err
}

/*
 * RPC DeleteInstance()
 */

func (k *Kaktus) DeleteInstance(args *proto.KaktusDeleteInstanceArgs, reply *proto.KaktusDeleteInstanceReply) error {
	err := k.agent.lcs.DeleteInstance(args.Name)
	*reply = proto.KaktusDeleteInstanceReply{}
	return err
}

/*
 * RPC UpdateInstance()
 */

func (k *Kaktus) UpdateInstance(args *proto.KaktusUpdateInstanceArgs, reply *proto.KaktusUpdateInstanceReply) error {
	err := k.agent.lcs.UpdateInstance(args.Name, args.XML)
	*reply = proto.KaktusUpdateInstanceReply{}
	return err
}

/*
 * RPC GetInstanceState()
 */

func (k *Kaktus) GetInstanceState(args *proto.KaktusGetInstanceStateArgs, reply *proto.KaktusGetInstanceStateReply) error {
	state, reason, err := k.agent.lcs.GetInstanceState(args.Name)
	*reply = proto.KaktusGetInstanceStateReply{
		State:  state,
		Reason: reason,
	}
	return err
}

/*
 * RPC InstanceIsRunning()
 */

func (k *Kaktus) InstanceIsRunning(args *proto.KaktusInstanceIsRunningArgs, reply *proto.KaktusInstanceIsRunningReply) error {
	state := k.agent.lcs.IsInstanceRunning(args.Name)
	*reply = proto.KaktusInstanceIsRunningReply{
		Running: state,
	}
	return nil
}

/*
 * RPC GetInstanceRemoteConnectionUrl()
 */

func (k *Kaktus) GetInstanceRemoteConnectionUrl(args *proto.KaktusGetInstanceRemoteConnectionUrlArgs, reply *proto.KaktusGetInstanceRemoteConnectionUrlReply) error {
	url, err := k.agent.lcs.GetInstanceRemoteConnectionUrl(args.Name)
	*reply = proto.KaktusGetInstanceRemoteConnectionUrlReply{
		URL: url,
	}
	return err
}

/*
 * RPC InstanceOperation()
 */

func (k *Kaktus) InstanceOperation(args *proto.KaktusInstanceOperationArgs, reply *proto.KaktusInstanceOperationReply) error {
	*reply = proto.KaktusInstanceOperationReply{}

	switch args.Action {
	case proto.KaktusInstanceOpAutoStart:
		return k.agent.lcs.AutoStartInstance(args.Name)
	case proto.KaktusInstanceOpStart:
		return k.agent.lcs.StartInstance(args.Name)
	case proto.KaktusInstanceOpSoftReboot:
		return k.agent.lcs.RebootInstance(args.Name)
	case proto.KaktusInstanceOpHardReboot:
		return k.agent.lcs.ResetInstance(args.Name)
	case proto.KaktusInstanceOpSoftShutdown:
		return k.agent.lcs.ShutdownInstance(args.Name)
	case proto.KaktusInstanceOpHardShutdown:
		return k.agent.lcs.StopInstance(args.Name)
	case proto.KaktusInstanceOpPmSuspend:
		return k.agent.lcs.SuspendInstance(args.Name)
	case proto.KaktusInstanceOpPmResume:
		return k.agent.lcs.ResumeInstance(args.Name)
	}

	return fmt.Errorf("unsupported instance operation: %d", args.Action)
}

/*
 * RPC GetStoragePoolStats()
 */

func (k *Kaktus) GetStoragePoolStats(args *proto.KaktusGetStoragePoolStatsArgs, reply *proto.KaktusGetStoragePoolStatsReply) error {
	used, available, total, err := k.agent.ceph.GetPoolStats(args.Pool)
	*reply = proto.KaktusGetStoragePoolStatsReply{
		Allocated: used,
		Available: available,
		Capacity:  total,
	}
	return err
}

/*
 * RPC CreateRawVolume()
 */

func (k *Kaktus) CreateRawVolume(args *proto.KaktusCreateRawVolumeArgs, reply *proto.KaktusCreateRawVolumeReply) error {
	err := k.agent.ceph.CreateRbdVolume(args.Pool, args.Volume, args.Size)
	*reply = proto.KaktusCreateRawVolumeReply{}
	return err
}

/*
 * RPC CreateTemplateVolume()
 */

func (k *Kaktus) CreateTemplateVolume(args *proto.KaktusCreateTemplateVolumeArgs, reply *proto.KaktusCreateTemplateVolumeReply) error {
	size, err := k.agent.ceph.CreateRbdVolumeFromUrl(args.Pool, args.Volume, args.SourceURL)
	*reply = proto.KaktusCreateTemplateVolumeReply{
		Size: size,
	}
	return err
}

/*
 * RPC CreateOsVolume()
 */

func (k *Kaktus) CreateOsVolume(args *proto.KaktusCreateOsVolumeArgs, reply *proto.KaktusCreateOsVolumeReply) error {
	err := k.agent.ceph.CloneRbdVolume(args.Pool, args.Template, args.Volume, args.Size)
	*reply = proto.KaktusCreateOsVolumeReply{}
	return err
}

/*
 * RPC CreateIsoVolume()
 */

func (k *Kaktus) CreateIsoVolume(args *proto.KaktusCreateIsoVolumeArgs, reply *proto.KaktusCreateIsoVolumeReply) error {
	err := k.agent.ceph.CreateRbdVolumeFromBinData(args.Pool, args.Volume, args.Size, args.Content)
	*reply = proto.KaktusCreateIsoVolumeReply{}
	return err
}

/*
 * RPC UpdateIsoVolume()
 */

func (k *Kaktus) UpdateIsoVolume(args *proto.KaktusUpdateIsoVolumeArgs, reply *proto.KaktusUpdateIsoVolumeReply) error {
	err := k.agent.ceph.UpdateRbdVolumeFromBinData(args.Pool, args.Volume, args.Size, args.Content)
	*reply = proto.KaktusUpdateIsoVolumeReply{}
	return err
}

/*
 * RPC GetVolumeInfos()
 */

func (k *Kaktus) GetVolumeInfos(args *proto.KaktusGetVolumeInfosArgs, reply *proto.KaktusGetVolumeInfosReply) error {
	size, err := k.agent.ceph.GetRbdVolumeInfos(args.Pool, args.Volume)
	*reply = proto.KaktusGetVolumeInfosReply{
		Size: size,
	}
	return err
}

/*
 * RPC ResizeVolume()
 */

func (k *Kaktus) ResizeVolume(args *proto.KaktusResizeVolumeArgs, reply *proto.KaktusResizeVolumeReply) error {
	err := k.agent.ceph.ResizeRbdVolume(args.Pool, args.Volume, args.Size)
	*reply = proto.KaktusResizeVolumeReply{}
	return err
}

/*
 * RPC DeleteVolume()
 */

func (k *Kaktus) DeleteVolume(args *proto.KaktusDeleteVolumeArgs, reply *proto.KaktusDeleteVolumeReply) error {
	err := k.agent.ceph.DeleteRbdVolume(args.Pool, args.Volume, args.WithSnapshots)
	*reply = proto.KaktusDeleteVolumeReply{}
	return err
}

// CephFS

/*
 * RPC ListFileSystems()
 */

func (k *Kaktus) ListFileSystems(args *proto.KaktusListFileSystemsArgs, reply *proto.KaktusListFileSystemsReply) error {
	volumes, err := k.agent.ceph.ListVolumes()
	*reply = proto.KaktusListFileSystemsReply{
		FS: volumes,
	}
	return err
}

/*
 * RPC ListFsSubVolumes()
 */

func (k *Kaktus) ListFsSubVolumes(args *proto.KaktusListFsSubVolumesArgs, reply *proto.KaktusListFsSubVolumesReply) error {
	subvolumes, err := k.agent.ceph.ListSubVolumes(args.FS)
	*reply = proto.KaktusListFsSubVolumesReply{
		SubVolumes: subvolumes,
	}
	return err
}

/*
 * RPC CreateFsSubVolume()
 */

func (k *Kaktus) CreateFsSubVolume(args *proto.KaktusCreateFsSubVolumeArgs, reply *proto.KaktusCreateFsSubVolumeReply) error {
	path, size, err := k.agent.ceph.CreateSubVolume(args.FS, args.SubVolume)
	*reply = proto.KaktusCreateFsSubVolumeReply{
		Path:      path,
		BytesUsed: size,
	}
	return err
}

/*
 * RPC DeleteFsSubVolume()
 */

func (k *Kaktus) DeleteFsSubVolume(args *proto.KaktusDeleteFsSubVolumeArgs, reply *proto.KaktusDeleteFsSubVolumeReply) error {
	err := k.agent.ceph.DeleteSubVolume(args.FS, args.SubVolume)
	*reply = proto.KaktusDeleteFsSubVolumeReply{}
	return err
}

// NFS

/*
 * RPC CreateNfsBackends()
 */

func (k *Kaktus) CreateNfsBackends(args *proto.KaktusCreateNfsBackendsArgs, reply *proto.KaktusCreateNfsBackendsReply) error {
	err := k.agent.nfs.CreateBackends(args.ID, args.Name, args.FS, args.Path, args.Access, args.Protocols, args.Clients, args.Backends, args.Port)
	*reply = proto.KaktusCreateNfsBackendsReply{}
	return err
}

/*
 * RPC UpdateNfsBackends()
 */

func (k *Kaktus) UpdateNfsBackends(args *proto.KaktusUpdateNfsBackendsArgs, reply *proto.KaktusUpdateNfsBackendsReply) error {
	err := k.agent.nfs.UpdateBackends(args.ID, args.Name, args.FS, args.Path, args.Access, args.Protocols, args.Clients, args.Backends, args.Port)
	*reply = proto.KaktusUpdateNfsBackendsReply{}
	return err
}

/*
 * RPC DeleteNfsBackends()
 */

func (k *Kaktus) DeleteNfsBackends(args *proto.KaktusDeleteNfsBackendsArgs, reply *proto.KaktusDeleteNfsBackendsReply) error {
	err := k.agent.nfs.DeleteBackends(args.ID, args.Name, args.FS, args.Path, args.Access, args.Protocols, args.Clients, args.Backends, args.Port)
	*reply = proto.KaktusDeleteNfsBackendsReply{}
	return err
}
