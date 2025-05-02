/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"net"
	"reflect"
	"sort"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/kaktus"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionInstanceSchemaVersion = 2
	MongoCollectionInstanceName          = "instance"

	ErrInstanceNoSuchAdapter = "no such adapter connected to instance"
	ErrInstanceNoSuchVolume  = "no such volume connected to instance"
)

type Instance struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`
	KaktusID  string `bson:"kaktus_id"`
	ProfileID string `bson:"profile_id"` // optional, only for as-a-service things like kawaii
	AgentID   string `bson:"agent_id"`   // optional, only for as-a-service things like kawaii

	// properties
	OS                string       `bson:"os"`
	CPU               int64        `bson:"vcpus"`
	Memory            int64        `bson:"memory"`
	RootPassword      string       `bson:"initial_root_password"`
	CloudInitVolumeId string       `bson:"cloudinit"`
	Profile           string       `bson:"profile"`
	Cost              InstanceCost `bson:"cost"`
	LocalIP           string       `bson:"local_ip"`

	// children references
	Interfaces map[string]string `bson:"interfaces"`
	Disks      map[string]string `bson:"disks"`
}

func InstanceMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("instances", MongoCollectionInstanceName)
	if err != nil {
		return err
	}

	for _, instance := range FindInstances() {
		if instance.SchemaVersion == 0 || instance.SchemaVersion == 1 {
			err := instance.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type InstanceCost struct {
	Price    float32 `bson:"price,truncate"`
	Currency string  `bson:"currency"`
}

func (i *Instance) NewInterfaceMap(adapters []string) (map[string]string, error) {
	interfaces := map[string]string{}

	// check if there's any public adapter
	var publicAdapter *Adapter = nil
	privateAdapters := []Adapter{}
	for _, adapterId := range adapters {
		a, err := FindAdapterByID(adapterId)
		if err != nil {
			return interfaces, err
		}

		s, err := a.Subnet()
		if err != nil {
			return interfaces, err
		}

		ip, _, err := net.ParseCIDR(s.CIDR)
		if err != nil {
			return interfaces, err
		}

		if !ip.IsPrivate() {
			if publicAdapter == nil {
				publicAdapter = a
				continue
			} else {
				return interfaces, fmt.Errorf("multiple public adapters have been found. Unsupported")
			}
		}

		privateAdapters = append(privateAdapters, *a)
	}

	index := 0
	// if a public adapter has been found, always flag it as the primary network interface
	if publicAdapter != nil {
		device := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, index+AdapterOsNicLinuxStartIndex)
		if i.OS == TemplateOsWindows {
			device = fmt.Sprintf("%s%d", AdapterOsNicWindowsPrefix, index+AdapterOsNicWindowsStartIndex)
		}
		interfaces[device] = publicAdapter.String()
		index += 1
	}

	// loop over remaining private adapters
	for _, a := range privateAdapters {
		device := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, index+AdapterOsNicLinuxStartIndex)
		if i.OS == TemplateOsWindows {
			device = fmt.Sprintf("%s%d", AdapterOsNicWindowsPrefix, index+AdapterOsNicWindowsStartIndex)
		}
		interfaces[device] = a.String()
		index += 1
	}

	// set local IP settings, used as FindBy() key for metadata
	if len(privateAdapters) > 0 {
		if len(privateAdapters[0].Addresses) > 0 {
			i.LocalIP = privateAdapters[0].Addresses[0]
		}
	}

	return interfaces, nil
}

func (i *Instance) NewDiskMap(volumes []string) (map[string]string, error) {
	disks := map[string]string{}

	// check if there's an OS volume
	var osVolume *Volume = nil
	dataVolumes := []Volume{}
	for _, volumeId := range volumes {
		v, err := FindVolumeByID(volumeId)
		if err != nil {
			return disks, err
		}

		if v.Type == VolumeTypeOs {
			if osVolume == nil {
				osVolume = v
				continue
			} else {
				return disks, fmt.Errorf("multiple OS volumes have been found. Unsupported")
			}
		}

		dataVolumes = append(dataVolumes, *v)
	}

	if osVolume == nil {
		return disks, fmt.Errorf("no OS volume can be found. Unsupported")
	}

	index := 0
	// register OS disk, always flag it as the primary disk, otherwise system can't be booted up
	device := fmt.Sprintf("%s%s", VolumeOsDiskPrefix, diskLetterForIndex(index))
	disks[device] = osVolume.String()
	index += 1

	// loop over remaining data volumes
	for _, v := range dataVolumes {
		device := fmt.Sprintf("%s%s", VolumeOsDiskPrefix, diskLetterForIndex(index))
		disks[device] = v.String()
		index += 1
	}

	return disks, nil
}

func NewInstance(projectId, kaktusId, name, desc, profile, profileId string, cpu, mem int64, adapters, volumes []string) (*Instance, error) {
	// ensure we have a rightful hostname, if any
	if !VerifyHostname(name) {
		err := fmt.Errorf("invalid host name: %s", name)
		klog.Error(err)
		return nil, err
	}

	instance := Instance{
		Resource:  NewResource(name, desc, MongoCollectionInstanceSchemaVersion),
		ProjectID: projectId,
		KaktusID:  kaktusId,
		Profile:   profile,
		ProfileID: profileId,
		CPU:       cpu,
		Memory:    mem,
		Cost:      InstanceCost{},
	}

	// find associated OS
	var osType string
	for _, vid := range volumes {
		v, err := FindVolumeByID(vid)
		if err != nil {
			return nil, err
		}
		if v.Type == VolumeTypeOs {
			t, err := v.Template()
			if err != nil {
				return nil, err
			}
			osType = t.OS
			break
		}
	}
	instance.OS = osType

	interfaces, err := instance.NewInterfaceMap(adapters)
	if err != nil {
		return nil, err
	}
	instance.Interfaces = interfaces

	disks, err := instance.NewDiskMap(volumes)
	if err != nil {
		return nil, err
	}
	instance.Disks = disks

	prj, err := FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}
	rootPassword := prj.RootPassword
	if rootPassword == "" {
		// no default bootstrap password has been set at project-level, so generate a random one
		rootPassword = common.GenerateRandomPassword(32)
	}
	instance.RootPassword = rootPassword

	k, err := instance.Kaktus()
	if err != nil {
		return nil, err
	}

	err = instance.CreateDnsRecord()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	switch instance.Profile {
	case CloudinitProfileKawaii, CloudinitProfileKonvey:
		err = instance.CreateAgent()
		if err != nil {
			klog.Error(err)
			// cleanup
			_ = instance.Delete()
			return nil, err
		}
	}

	err = instance.CreateCloudInitVolume()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	err = instance.CreateInstance()
	if err != nil {
		klog.Error(err)
		// cleanup
		_ = instance.Delete()
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionInstanceName, instance)
	if err != nil {
		// cleanup
		_ = instance.Delete()
		return nil, err
	}
	klog.Infof("Created new instance %s (%s)", instance.String(), instance.Name)

	// setup initial cost
	azr, err := instance.AverageZoneResources()
	if err != nil {
		return nil, err
	}

	err = instance.ComputeCost(azr)
	if err != nil {
		return nil, err
	}

	// add instance to project
	prj.AddInstance(instance.String())

	// add instance to kaktus node
	k.AddInstance(instance.String())

	// notify project's users
	for _, u := range prj.NotifiableUsers() {
		err := NewEmailInstanceCreated(&instance, u)
		if err != nil {
			klog.Error(err)
			// not a blocker
		}
	}

	return &instance, nil
}

func FindInstances() []Instance {
	return FindResources[Instance](MongoCollectionInstanceName)
}

func FindInstancesByProject(projectId string) ([]Instance, error) {
	return FindResourcesByKey[Instance](MongoCollectionInstanceName, "project_id", projectId)
}

func FindInstancesByKaktus(kaktusId string) ([]Instance, error) {
	return FindResourcesByKey[Instance](MongoCollectionKaktusName, "kaktus_id", kaktusId)
}

func FindInstanceByID(id string) (*Instance, error) {
	return FindResourceByID[Instance](MongoCollectionInstanceName, id)
}

func FindInstanceByName(name string) (*Instance, error) {
	return FindResourceByName[Instance](MongoCollectionInstanceName, name)
}

func FindInstanceByIP(ip string) (*Instance, error) {
	return FindResourceByIP[Instance](MongoCollectionInstanceName, ip)
}

func (i *Instance) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionInstanceName, i.ID, from, to)
}

func (i *Instance) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionInstanceName, i.ID, version)
}

func (i *Instance) migrateSchemaV2() error {
	err := i.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = i.renameDbField("host", "kaktus_id")
	if err != nil {
		return err
	}

	err = i.renameDbField("instance_type_id", "profile_id")
	if err != nil {
		return err
	}

	err = i.renameDbField("agent", "agent_id")
	if err != nil {
		return err
	}

	err = i.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (i *Instance) RPC(method string, args, reply any) error {
	k, err := i.Kaktus()
	if err != nil {
		return err
	}

	return k.RPC(method, args, reply)
}

func (i *Instance) InstanceRPC(method string, args, reply any) error {
	return RPC([]string{i.AgentID}, method, args, reply)
}

var diskLetters = []rune("abcdefghijklmnopqrstuvwxyz")

// diskLetterForIndex return diskLetters for index
func diskLetterForIndex(i int) string {

	q := i / len(diskLetters)
	r := i % len(diskLetters)
	letter := diskLetters[r]

	if q == 0 {
		return fmt.Sprintf("%c", letter)
	}

	return fmt.Sprintf("%s%c", diskLetterForIndex(q-1), letter)
}

func (i *Instance) DeviceIDs(m map[string]string) []string {
	devices := []string{}
	for _, v := range m {
		devices = append(devices, v)
	}
	sort.Strings(devices)
	return devices
}

func (i *Instance) Adapters() []string {
	return i.DeviceIDs(i.Interfaces)
}

func (i *Instance) Volumes() []string {
	return i.DeviceIDs(i.Disks)
}

func (i *Instance) GetOsVolume() (*Volume, error) {
	osCount := 0
	var osVolume *Volume

	for _, volumeId := range i.Disks {
		v, err := FindVolumeByID(volumeId)
		if err != nil {
			return nil, err
		}

		// find OS volume
		if v.Type == VolumeTypeOs {
			osCount += 1
			osVolume = v
		}
	}

	// ensure we only have one OS volume (no more, no less)
	if osCount != 1 || osVolume == nil {
		return nil, fmt.Errorf("more than one OS volume. Unsupported")
	}

	return osVolume, nil
}

func (i *Instance) CreateCloudInitVolume() error {
	osVolume, err := i.GetOsVolume()
	if err != nil {
		return err
	}

	// finally, create a new cloud-init ISO image volume
	osTemplate, err := osVolume.Template()
	if err != nil {
		return err
	}

	k, err := i.Kaktus()
	if err != nil {
		return err
	}

	z, err := k.Zone()
	if err != nil {
		return err
	}

	// if no cloud-init volume exists for this instance, create one
	if (osTemplate.OS == TemplateOsLinux || osTemplate.OS == TemplateOsWindows) && i.CloudInitVolumeId == "" {
		v, err := NewCloudInitVolume(osVolume.ProjectID, z.String(), osVolume.StoragePoolID, i.String(), i.AgentID, i.Name, i.LocalIP, osTemplate.OS, i.RootPassword, i.Profile, i.Adapters())
		if err != nil {
			return err
		}
		i.CloudInitVolumeId = v.String()
		i.Save()
	}

	return nil
}

func (i *Instance) create(xml string) error {
	args := kaktus.KaktusCreateInstanceArgs{
		Name: i.Name,
		XML:  xml,
	}
	var reply kaktus.KaktusCreateInstanceReply
	return i.RPC("CreateInstance", args, &reply)
}

func (i *Instance) CreateInstance() error {
	// TODO: migrate once and for all in host settings instead of querying at each instance creation ?
	args := kaktus.KaktusNodeCapabilitiesArgs{}
	var reply kaktus.KaktusNodeCapabilitiesReply

	err := i.RPC("NodeCapabilities", args, &reply)
	if err != nil {
		klog.Errorf("unable to get host capabilities: %v", err)
		return err
	}

	d := NewVirtualInstanceDescription(i.OS, i.Name, i.Description,
		reply.Arch, reply.GuestMachineName,
		reply.GuestEmulator, i.Memory, i.CPU)

	// discover and attach disks/volumes
	d.SetDisks(i.Disks, i.CloudInitVolumeId)

	// discover and attach interfaces/adapters
	d.SetInterfaces(i.Interfaces)

	data, err := d.XML()
	if err != nil {
		return err
	}
	klog.Debugf("Generated XML for libvirt domain:\n%s", data)

	// create libvirt domain
	err = i.create(data)
	if err != nil {
		return err
	}

	return i.AutoStart()
}

func (i *Instance) GetIpAddress(private bool) string {
	for _, adapterId := range i.Interfaces {
		a, err := FindAdapterByID(adapterId)
		if err != nil {
			continue
		}

		if len(a.Addresses) == 0 {
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

		if private == v.Private {
			return a.Addresses[0]
		}
	}
	return ""
}

func (i *Instance) CreateAgent() error {
	// ensure agent does not already exists, if so, re-use
	a, err := FindAgentByName(i.Name)
	if err == nil {
		i.AgentID = a.String()
	}

	// create agent
	a, err = NewAgent(i.Name, "", common.KowabungaControllerAgent)
	if err != nil {
		return err
	}
	i.AgentID = a.String()

	return nil
}

func (i *Instance) CreateDnsRecord() error {
	prj, err := i.Project()
	if err != nil {
		return err
	}

	if prj.Domain == "" {
		klog.Warningf("No associated project domain can be found, ignoring instance DNS record creation")
		return nil
	}

	addresses := []string{}
	for _, adapterId := range i.Adapters() {
		a, err := FindAdapterByID(adapterId)
		if err != nil {
			return err
		}

		s, err := a.Subnet()
		if err != nil {
			return err
		}

		v, err := s.VNet()
		if err != nil {
			return err
		}

		// only add private addresses
		if v.Private {
			addresses = append(addresses, a.Addresses...)
		}
	}

	// create record on all possible network gateways (for now)
	if len(addresses) > 0 {
		for _, gw := range FindKiwis() {
			klog.Infof("Creating DNS Record for %s ...", i.Name)
			err := gw.CreateDnsRecord(prj.Domain, i.Name, addresses)
			if err != nil {
				klog.Error(err)
				return err
			}
		}
	}

	return nil
}

func (i *Instance) DeleteDnsRecord() error {
	prj, err := i.Project()
	if err != nil {
		return err
	}

	if prj.Domain == "" {
		klog.Warningf("No associated project domain can be found, ignoring instance DNS record deletion")
		return nil
	}

	// delete zone from all possible network gateways (for now)
	for _, gw := range FindKiwis() {
		err := gw.DeleteDnsRecord(prj.Domain, i.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Instance) get(migratable bool) (string, error) {
	args := kaktus.KaktusGetInstanceArgs{
		Name:       i.Name,
		Migratable: migratable,
	}
	var reply kaktus.KaktusGetInstanceReply

	err := i.RPC("GetInstance", args, &reply)
	if err != nil {
		return "", err
	}
	return reply.XML, nil
}

func (i *Instance) update(data string) error {
	args := kaktus.KaktusUpdateInstanceArgs{
		Name: i.Name,
		XML:  data,
	}
	var reply kaktus.KaktusUpdateInstanceReply
	return i.RPC("UpdateInstance", args, &reply)
}

func (i *Instance) Update(name, desc string, cpu, mem int64, adapters, volumes []string) error {
	i.UpdateResourceDefaults(name, desc)

	prj, err := i.Project()
	if err != nil {
		return err
	}

	k, err := i.Kaktus()
	if err != nil {
		return err
	}

	xml, err := i.get(true)
	if err != nil {
		klog.Errorf("unable to get instance %s description: %v", i.Name, err)
		return err
	}

	d, err := NewVirtualInstanceFromXml(xml)
	if err != nil {
		return err
	}

	hasChanged := false
	cpuDelta := cpu - i.CPU
	memDelta := mem - i.Memory
	if cpu != i.CPU {
		i.CPU = cpu
		d.SetCPU(i.CPU)
		hasChanged = true
	}

	if mem != i.Memory {
		i.Memory = mem
		d.SetMemory(i.Memory)
		hasChanged = true
	}

	// check if list of adapters has changed
	interfaces, err := i.NewInterfaceMap(adapters)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(i.Interfaces, interfaces) {
		i.Interfaces = interfaces
		// discover and attach interfaces/adapters
		d.SetInterfaces(i.Interfaces)
		hasChanged = true
	}

	// check if list of volumes has changed
	disks, err := i.NewDiskMap(volumes)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(i.Disks, disks) {
		i.Disks = disks
		// discover and attach disks/volumes
		d.SetDisks(i.Disks, i.CloudInitVolumeId)
		hasChanged = true
	}

	if hasChanged {
		data, err := VirtualInstanceToXml(d)
		if err != nil {
			return err
		}
		klog.Debugf("Updated XML for libvirt domain:\n%s", data)

		err = i.update(data)
		if err != nil {
			klog.Errorf("unable to update instance %s: %v", i.Name, err)
			return err
		}

		// reboot instance for settings to take effect
		err = i.Shutdown()
		if err != nil {
			klog.Error(err)
		}

		err = i.Stop()
		if err != nil {
			klog.Error(err)
		}

		err = i.Start()
		if err != nil {
			klog.Error(err)
		}

		// update instance cost details
		azr, err := i.AverageZoneResources()
		if err != nil {
			klog.Error(err)
		}

		err = i.ComputeCost(azr)
		if err != nil {
			klog.Error(err)
		}

		// update project usage counter
		prj.UpdateInstanceUsage(cpuDelta, memDelta)

		// update host usage counter
		k.UpdateInstanceUsage(cpuDelta, memDelta)
	}

	i.Save()
	return nil
}

func (i *Instance) Project() (*Project, error) {
	return FindProjectByID(i.ProjectID)
}

func (i *Instance) Kaktus() (*Kaktus, error) {
	return FindKaktusByID(i.KaktusID)
}

func (i *Instance) HasChildren() bool {
	// TODO: need to add instance reference to adapters and volumes and get them removed from list when deleted ?
	// return HasChildRefs(i.Adapters(), i.Volumes())
	return false
}

func (i *Instance) Save() {
	i.Updated()
	_, err := GetDB().Update(MongoCollectionInstanceName, i.ID, i)
	if err != nil {
		klog.Error(err)
	}
}

func (i *Instance) delete() error {
	args := kaktus.KaktusDeleteInstanceArgs{
		Name: i.Name,
	}
	var reply kaktus.KaktusDeleteInstanceReply

	return i.RPC("DeleteInstance", args, &reply)
}

func (i *Instance) Delete() error {
	klog.Infof("Deleting instance %s (%s)", i.String(), i.Name)

	if i.String() == ResourceUnknown {
		return nil
	}

	err := i.Stop()
	if err != nil {
		klog.Error(err)
		// nevermind, kill it
	}

	err = i.delete()
	if err != nil {
		klog.Errorf("unable to delete instance %s: %v", i.Name, err)
		// nevermind, already gone
	}

	// delete associated cloud-init volume, if any
	if i.CloudInitVolumeId != "" {
		v, err := FindVolumeByID(i.CloudInitVolumeId)
		if err != nil {
			klog.Error(err)
		}
		if v.Type == VolumeTypeIso {
			err := v.Delete()
			if err != nil {
				return err
			}
		}
	}

	// remove agent, if any
	switch i.Profile {
	case CloudinitProfileKawaii, CloudinitProfileKonvey:
		if i.AgentID != "" {
			a, err := FindAgentByID(i.AgentID)
			if err != nil {
				return err
			}

			// remove agent
			err = a.Delete()
			if err != nil {
				return err
			}

			// disconnect any live agent WebSocket, if any
			DisconnectAgent(i.AgentID)
		}
	}

	// delete associated DNS record, if any
	err = i.DeleteDnsRecord()
	if err != nil {
		klog.Error(err)
		// should not be a blocker
	}

	// remove zone's reference from parents
	prj, err := i.Project()
	if err != nil {
		return err
	}
	prj.RemoveInstance(i.String())

	k, err := i.Kaktus()
	if err != nil {
		return err
	}
	k.RemoveInstance(i.String())

	return GetDB().Delete(MongoCollectionInstanceName, i.ID)
}

func (i *Instance) Model() sdk.Instance {
	return sdk.Instance{
		Id:          i.String(),
		Name:        i.Name,
		Description: i.Description,
		Vcpus:       i.CPU,
		Memory:      i.Memory,
		Adapters:    i.Adapters(),
		Volumes:     i.Volumes(),
	}
}

func (i *Instance) AverageZoneResources() (*ZoneVirtualResources, error) {
	k, err := i.Kaktus()
	if err != nil {
		return nil, err
	}

	return k.AverageZoneResources()
}

func (i *Instance) ComputeCost(res *ZoneVirtualResources) error {
	vcpus := i.CPU
	vmem_gb := float64(bytesToGB(i.Memory))
	currency := res.Computing.Currency
	price := float32(vcpus)*res.Computing.Price + float32(vmem_gb)*res.Memory.Price
	klog.Debugf("Instance %s features %d vCPUs and %f GB RAM for %f %s", i, vcpus, vmem_gb, price, currency)

	i.Cost.Price = price
	i.Cost.Currency = currency
	i.Save()

	return nil
}

func (i *Instance) GetState() (sdk.InstanceState, error) {
	args := kaktus.KaktusGetInstanceStateArgs{
		Name: i.Name,
	}
	var reply kaktus.KaktusGetInstanceStateReply

	err := i.RPC("GetInstanceState", args, &reply)
	if err != nil {
		klog.Errorf("Unable to get instance %s state: %v", i.Name, err)
		return sdk.InstanceState{}, err
	}

	return sdk.InstanceState{
		State:  reply.State,
		Reason: reply.Reason,
	}, nil
}

func (i *Instance) GetRemoteConnectionURL() (sdk.InstanceRemoteAccess, error) {
	args := kaktus.KaktusGetInstanceRemoteConnectionUrlArgs{
		Name: i.Name,
	}
	var reply kaktus.KaktusGetInstanceRemoteConnectionUrlReply

	err := i.RPC("GetInstanceRemoteConnectionUrl", args, &reply)
	if err != nil {
		klog.Errorf("Unable to get instance %s remote connection URL: %v", i.Name, err)
		return sdk.InstanceRemoteAccess{}, err
	}

	return sdk.InstanceRemoteAccess{
		Url: reply.URL,
	}, nil
}

// Is instance running ?
func (i *Instance) IsRunning() bool {
	args := kaktus.KaktusInstanceIsRunningArgs{
		Name: i.Name,
	}
	var reply kaktus.KaktusInstanceIsRunningReply

	err := i.RPC("InstanceIsRunning", args, &reply)
	if err != nil {
		return false
	}

	return reply.Running
}

func (i *Instance) operation(action string, op kaktus.KaktusInstanceOperation) error {
	if action != "" {
		klog.Infof("%s instance %s (%s)", action, i.String(), i.Name)
	}

	args := kaktus.KaktusInstanceOperationArgs{
		Name:   i.Name,
		Action: op,
	}
	var reply kaktus.KaktusInstanceOperationReply

	err := i.RPC("InstanceOperation", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

// Software OS reboot
func (i *Instance) Reboot() error {
	return i.operation("Rebooting", kaktus.KaktusInstanceOpSoftReboot)
}

// Hardware Reset
func (i *Instance) Reset() error {
	return i.operation("Hardware reset of", kaktus.KaktusInstanceOpHardReboot)
}

// Software PM Suspend
func (i *Instance) Suspend() error {
	return i.operation("Suspending", kaktus.KaktusInstanceOpPmSuspend)
}

// Software PM Resume
func (i *Instance) Resume() error {
	return i.operation("Resuming", kaktus.KaktusInstanceOpPmResume)
}

// Enable auto-start
func (i *Instance) AutoStart() error {
	return i.operation("", kaktus.KaktusInstanceOpAutoStart)
}

// Hardware Boot
func (i *Instance) Start() error {
	return i.operation("Starting", kaktus.KaktusInstanceOpStart)
}

// Hardware Shutdown
func (i *Instance) Stop() error {
	return i.operation("Stopping", kaktus.KaktusInstanceOpHardShutdown)
}

// Software Shutdown
func (i *Instance) Shutdown() error {
	return i.operation("Shutting down", kaktus.KaktusInstanceOpSoftShutdown)
}
