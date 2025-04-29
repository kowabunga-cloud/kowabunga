/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/kaktus"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionKaktusSchemaVersion = 2
	MongoCollectionKaktusName          = "kaktus"

	ErrKaktusNoSuchInstance = "no such instance in kaktus"

	KaktusMaxScore                  = 999999
	KaktusScoreFactorInstancesCount = 2
	KaktusScoreFactorVCPUs          = 1
	KaktusScoreFactorMemory         = 1
	KaktusScoreMaxLevenshteinDist   = 2 // maximum distance between two hostnames to be considered as siblings
	KaktusScoreSiblingMalus         = 1000

	KaktusCpuOverCommmitRatio   = 3
	KaktusMemoryOverCommitRatio = 2
)

type Kaktus struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ZoneID string `bson:"zone_id"`

	// properties
	Costs            KaktusCosts            `bson:"costs"`
	Usage            KaktusResources        `bson:"usage"` // 0 for un-used
	Capabilities     KaktusCapabilities     `bson:"capabilities"`
	OverCommit       KaktusOverCommitRatio  `bson:"overcommit"`
	VirtualResources KaktusVirtualResources `bson:"virtual_resources"`
	AgentIDs         []string               `bson:"agent_ids"`

	// children references
	InstanceIDs []string `bson:"instance_ids"`
}

type KaktusResources struct {
	VCPUs          uint16 `bson:"vcpus"`     // sum of all instances vCPUs
	MemorySize     uint64 `bson:"memory"`    // sum of all instances memory (bytes)
	InstancesCount uint16 `bson:"instances"` // count of associated instances (regardless of their state)
}

type KaktusCapabilities struct {
	CPU    KaktusCPU `bson:"cpu"`
	Memory int64     `bson:"memory"`
}

type KaktusOverCommitRatio struct {
	CPU    int64 `bson:"cpu"`
	Memory int64 `bson:"memory"`
}

type KaktusCosts struct {
	CPU    ResourceCost `bson:"cpu"`
	Memory ResourceCost `bson:"memory"`
}

type KaktusVirtualResources struct {
	VCPU   KaktusVirtualResource `bson:"vcpu"`
	VMemGB KaktusVirtualResource `bson:"vmem_gb"`
}

type KaktusVirtualResource struct {
	Count    int64   `bson:"count"`
	Price    float32 `bson:"price,truncate"`
	Currency string  `bson:"currency"`
}

func (kc *KaktusCapabilities) Model() sdk.KaktusCaps {

	cpu := kc.CPU.Model()
	return sdk.KaktusCaps{
		Cpu:    cpu,
		Memory: kc.Memory,
	}
}

type KaktusCPU struct {
	Arch    string `bson:"arch"`
	Cores   int64  `bson:"cores"`
	Modele  string `bson:"model"`
	Sockets int64  `bson:"sockets"`
	Threads int64  `bson:"threads"`
	Vendor  string `bson:"vendor"`
}

func (kc *KaktusCPU) Model() sdk.KaktusCpu {
	return sdk.KaktusCpu{
		Arch:    kc.Arch,
		Cores:   kc.Cores,
		Model:   kc.Modele,
		Sockets: kc.Sockets,
		Threads: kc.Threads,
		Vendor:  kc.Vendor,
	}
}

func KaktusMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("hosts", MongoCollectionKaktusName)
	if err != nil {
		return err
	}

	for _, kaktus := range FindKaktuses() {
		if kaktus.SchemaVersion == 0 || kaktus.SchemaVersion == 1 {
			err := kaktus.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewKaktus(zoneId, name, desc string, cpu_price float32, cpu_currency string, memory_price float32, memory_currency string, overcommit_cpu, overcommit_memory int64, agts []string) (*Kaktus, error) {
	h := Kaktus{
		Resource: NewResource(name, desc, MongoCollectionKaktusSchemaVersion),
		ZoneID:   zoneId,
		Costs: KaktusCosts{
			CPU:    NewResourceCost(cpu_price, cpu_currency),
			Memory: NewResourceCost(memory_price, memory_currency),
		},
		InstanceIDs: []string{},
		OverCommit: KaktusOverCommitRatio{
			CPU:    overcommit_cpu,
			Memory: overcommit_memory,
		},
		VirtualResources: KaktusVirtualResources{
			VCPU:   KaktusVirtualResource{},
			VMemGB: KaktusVirtualResource{},
		},
		AgentIDs: VerifyAgents(agts, common.KowabungaKaktusAgent),
	}

	z, err := h.Zone()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionKaktusName, h)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new kaktus %s", h.String())

	// add kaktus to zone
	z.AddKaktus(h.String())

	return &h, nil
}

func FindKaktuses() []Kaktus {
	return FindResources[Kaktus](MongoCollectionKaktusName)
}

func FindKaktusByZone(zoneId string) ([]Kaktus, error) {
	return FindResourcesByKey[Kaktus](MongoCollectionKaktusName, "zone_id", zoneId)
}

func FindKaktusByID(id string) (*Kaktus, error) {
	return FindResourceByID[Kaktus](MongoCollectionKaktusName, id)
}

func FindKaktusByName(name string) (*Kaktus, error) {
	return FindResourceByName[Kaktus](MongoCollectionKaktusName, name)
}

func (k *Kaktus) RPC(method string, args, reply any) error {
	return RPC(k.AgentIDs, method, args, reply)
}

func (k *Kaktus) Zone() (*Zone, error) {
	return FindZoneByID(k.ZoneID)
}

func (k *Kaktus) Agents() []string {
	return k.AgentIDs
}

func (k *Kaktus) HasChildren() bool {
	return HasChildRefs(k.InstanceIDs)
}

func (k *Kaktus) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionKaktusName, k.ID, from, to)
}

func (k *Kaktus) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionKaktusName, k.ID, version)
}

func (k *Kaktus) migrateSchemaV2() error {
	err := k.renameDbField("zone", "zone_id")
	if err != nil {
		return err
	}

	err = k.renameDbField("agents", "agent_ids")
	if err != nil {
		return err
	}

	err = k.renameDbField("instances", "instance_ids")
	if err != nil {
		return err
	}

	err = k.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kaktus) FindInstances() ([]Instance, error) {
	return FindInstancesByKaktus(k.String())
}

func (k *Kaktus) AverageZoneResources() (*ZoneVirtualResources, error) {
	z, err := k.Zone()
	if err != nil {
		return nil, err
	}

	return z.AverageVirtualResources(), nil
}

func (k *Kaktus) Scan() {
	klog.Debugf("Scanning Kaktus %s", k)

	args := kaktus.KaktusNodeCapabilitiesArgs{}
	var reply kaktus.KaktusNodeCapabilitiesReply

	err := k.RPC("NodeCapabilities", args, &reply)
	if err != nil {
		klog.Errorf("Unable to get remote kaktus capabilities: %v", err)
		return
	}

	hc := KaktusCapabilities{
		CPU: KaktusCPU{
			Arch:    reply.Arch,
			Cores:   int64(reply.Cores),
			Modele:  reply.Model,
			Sockets: int64(reply.Sockets),
			Threads: int64(reply.Threads),
			Vendor:  reply.Vendor,
		},
		Memory: int64(reply.Memory),
	}

	k.Capabilities = hc

	k.VirtualResourcesComputation()
	k.Save()
}

func (k *Kaktus) VirtualResourcesComputation() {
	updated := false

	vcpuCount := int64(k.Capabilities.CPU.Threads * k.OverCommit.CPU)
	if vcpuCount != k.VirtualResources.VCPU.Count {
		k.VirtualResources.VCPU.Count = vcpuCount
		updated = true
	}
	vcpuPrice := k.Costs.CPU.Price / float32(k.VirtualResources.VCPU.Count)
	if vcpuPrice != k.VirtualResources.VCPU.Price {
		k.VirtualResources.VCPU.Price = vcpuPrice
		updated = true
	}
	k.VirtualResources.VCPU.Currency = k.Costs.CPU.Currency
	klog.Debugf("Kaktus %s has %d overcommited vCPUs capability, with a %f %s per vCPU price", k, k.VirtualResources.VCPU.Count, k.VirtualResources.VCPU.Price, k.VirtualResources.VCPU.Currency)

	memCount := int64(bytesToGB(k.Capabilities.Memory) * k.OverCommit.Memory)
	if memCount != k.VirtualResources.VMemGB.Count {
		k.VirtualResources.VMemGB.Count = memCount
		updated = true
	}
	memPrice := k.Costs.Memory.Price / float32(k.VirtualResources.VMemGB.Count)
	if memPrice != k.VirtualResources.VMemGB.Price {
		k.VirtualResources.VMemGB.Price = memPrice
		updated = true
	}
	k.VirtualResources.VMemGB.Currency = k.Costs.Memory.Currency
	klog.Debugf("Kaktus %s has %d overcommited vGiB Memory capability, with a %f %s per vGiB Memory price", k, k.VirtualResources.VMemGB.Count, k.VirtualResources.VMemGB.Price, k.VirtualResources.VMemGB.Currency)

	// if kaktus settings have changed, trigger a zone capability update
	if updated {
		z, err := k.Zone()
		if err != nil {
			return
		}

		go func() {
			err := z.UpdateCapabilities()
			if err != nil {
				klog.Error(err.Error())
			}
		}()
	}
}

func (k *Kaktus) Update(name, desc string, cpu_price float32, cpu_currency string, memory_price float32, memory_currency string, overcommit_cpu, overcommit_memory int64, agts []string) {
	k.UpdateResourceDefaults(name, desc)

	k.Costs.CPU.Price = cpu_price
	SetFieldStr(&k.Costs.CPU.Currency, cpu_currency)
	k.Costs.Memory.Price = memory_price
	SetFieldStr(&k.Costs.Memory.Currency, memory_currency)
	k.OverCommit.CPU = overcommit_cpu
	k.OverCommit.Memory = overcommit_memory
	k.AgentIDs = VerifyAgents(agts, common.KowabungaKaktusAgent)
	k.VirtualResourcesComputation()

	k.Save()
}

func (k *Kaktus) Save() {
	k.Updated()
	_, err := GetDB().Update(MongoCollectionKaktusName, k.ID, k)
	if err != nil {
		klog.Error(err)
	}
}

func (k *Kaktus) Delete() error {
	klog.Debugf("Deleting kaktus %s", k.String())

	if k.String() == ResourceUnknown {
		return nil
	}

	// remove kaktus' reference from parents
	z, err := k.Zone()
	if err != nil {
		return err
	}
	z.RemoveKaktus(k.String())

	return GetDB().Delete(MongoCollectionKaktusName, k.ID)
}

func (k *Kaktus) Model() sdk.Kaktus {
	cpu_cost := k.Costs.CPU.Model()
	memory_cost := k.Costs.Memory.Model()

	return sdk.Kaktus{
		Id:                    k.String(),
		Name:                  k.Name,
		Description:           k.Description,
		CpuCost:               cpu_cost,
		MemoryCost:            memory_cost,
		OvercommitCpuRatio:    k.OverCommit.CPU,
		OvercommitMemoryRatio: k.OverCommit.Memory,
		Agents:                k.AgentIDs,
	}
}

func (k *Kaktus) UsageScore() int {
	score := 0
	score += int(k.Usage.InstancesCount) * KaktusScoreFactorInstancesCount
	score += int(k.Usage.VCPUs) * KaktusScoreFactorVCPUs
	memGB := int(bytesToGB(int64(k.Usage.MemorySize)))
	score += memGB * KaktusScoreFactorMemory
	return score
}

func (k *Kaktus) Instances() []string {
	return k.InstanceIDs
}

func (k *Kaktus) Instance(id string) (*Instance, error) {
	return FindChildByID[Instance](&k.InstanceIDs, id, MongoCollectionInstanceName, ErrKaktusNoSuchInstance)
}

func (k *Kaktus) AddInstance(id string) {
	klog.Debugf("Adding instance %s to kaktus %s", id, k.String())
	AddChildRef(&k.InstanceIDs, id)
	k.Save() // save DB before looking back

	// find instance again
	instance, err := k.Instance(id)
	if err != nil {
		klog.Error(err)
		return
	}

	// increase usage counters
	k.Usage.InstancesCount += 1
	k.Usage.VCPUs += uint16(instance.CPU)
	k.Usage.MemorySize += uint64(instance.Memory)

	k.Save()
}

func (k *Kaktus) UpdateInstanceUsage(cpu, mem int64) {
	// increase usage counters
	k.Usage.VCPUs += uint16(cpu)
	k.Usage.MemorySize += uint64(mem)
	k.Save()
}

func (k *Kaktus) RemoveInstance(id string) {
	klog.Debugf("Removing instance %s from kaktus %s", id, k.String())

	// ensure instance exists in kaktus
	ist, err := k.Instance(id)
	if err != nil {
		klog.Error(err)
		return
	}

	// decrease usage counters
	k.Usage.InstancesCount -= 1
	k.Usage.VCPUs -= uint16(ist.CPU)
	k.Usage.MemorySize -= uint64(ist.Memory)

	RemoveChildRef(&k.InstanceIDs, id)
	k.Save()
}
