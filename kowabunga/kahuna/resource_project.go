/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"slices"
	"sort"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionProjectSchemaVersion = 2
	MongoCollectionProjectName          = "project"

	ErrProjectNoSuchInstance  = "no such instance in project"
	ErrProjectNoSuchVolume    = "no such volume in project"
	ErrProjectNoSuchKompute   = "no such Kompute virtual machine in project"
	ErrProjectNoSuchKawaii    = "no such Kowabunga Gateway in project"
	ErrProjectNoSuchKonvey    = "no such Kowabunga Konvey in project"
	ErrProjectNoSuchKylo      = "no such Kylo storage in project"
	ErrProjectNoSuchDnsRecord = "no such DNS record in project"

	VirtualRouterIdMin = 1
	VirtualRouterIdMax = 255
)

type Project struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents

	// properties
	Domain          string             `bson:"domain"`
	RootPassword    string             `bson:"default_root_password"`
	BootstrapUser   string             `bson:"bootstrap_user"`
	BootstrapPubkey string             `bson:"bootstrap_pubkey"`
	Tags            []string           `bson:"tags"`
	Meta            []ResourceMetadata `bson:"metadatas"`
	Quotas          ProjectResources   `bson:"quotas"` // limits, 0 for unlimited
	Usage           ProjectResources   `bson:"usage"`  // usage, 0 for un-used
	Cost            ProjectCost        `bson:"cost"`
	TeamIDs         []string           `bson:"team_ids"`
	RegionIDs       []string           `bson:"region_ids"`
	VrrpIDs         []int              `bson:"reserved_vrrp_ids"`

	// children references
	InstanceIDs    []string          `bson:"instance_ids"`
	VolumeIDs      []string          `bson:"volume_ids"`
	KomputeIDs     []string          `bson:"kompute_ids"`
	KawaiiIDs      []string          `bson:"kawaii_ids"`
	KonveyIDs      []string          `bson:"konvey_ids"`
	KyloIDs        []string          `bson:"kylo_ids"`
	RecordIDs      []string          `bson:"record_ids"`
	PrivateSubnets map[string]string `bson:"private_subnets"`
	ZoneGateways   map[string]string `bson:"zone_gateways"`
}

type ProjectResources struct {
	VCPUs          uint16 `bson:"vcpus"`     // sum of all instances vCPUs
	MemorySize     uint64 `bson:"memory"`    // sum of all instances memory (bytes)
	StorageSize    uint64 `bson:"storage"`   // sum of all disks size (bytes)
	InstancesCount uint16 `bson:"instances"` // count of associated instances (regardless of their state)
}

func (pr *ProjectResources) Update(quotas sdk.ProjectResources) {
	pr.VCPUs = uint16(quotas.Vcpus)
	pr.MemorySize = uint64(quotas.Memory)
	pr.StorageSize = uint64(quotas.Storage)
	pr.InstancesCount = uint16(quotas.Instances)
}

func (pr *ProjectResources) Model() sdk.ProjectResources {
	return sdk.ProjectResources{
		Instances: int32(pr.InstancesCount),
		Memory:    int64(pr.MemorySize),
		Storage:   int64(pr.StorageSize),
		Vcpus:     int32(pr.VCPUs),
	}
}

type ProjectCost struct {
	Price    float32 `bson:"price,truncate"`
	Currency string  `bson:"currency"`
}

func getMetadatas(meta map[string]string) []ResourceMetadata {
	metas := []ResourceMetadata{}
	for k, v := range meta {
		m := ResourceMetadata{k, v}
		metas = append(metas, m)
	}
	return metas
}

func ProjectMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("projects", MongoCollectionProjectName)
	if err != nil {
		return err
	}

	for _, project := range FindProjects() {
		if project.SchemaVersion == 0 || project.SchemaVersion == 1 {
			err = project.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewProject(name, desc, domain, pwd, user, pubkey string, teams, regions, tags []string, meta map[string]string, quotas sdk.ProjectResources, subnetSize int) (*Project, error) {
	// ensure we have a rightful domain, if any
	if domain != "" && !VerifyDomain(domain) {
		err := fmt.Errorf("Invalid domain name: %s", domain)
		return nil, err
	}

	// if unspecified by project, use Kowabunga's global settings
	if user == "" {
		user = GetCfg().Global.Bootstrap.User
	}
	if pubkey == "" {
		pubkey = GetCfg().Global.Bootstrap.Pubkey
	}

	p := Project{
		Resource:        NewResource(name, desc, MongoCollectionProjectSchemaVersion),
		Domain:          domain,
		RootPassword:    pwd,
		BootstrapUser:   user,
		BootstrapPubkey: pubkey,
		Tags:            tags,
		Meta:            getMetadatas(meta),
		Quotas:          ProjectResources{},
		Usage:           ProjectResources{},
		Cost: ProjectCost{
			Currency: CostCurrencyDefault,
		},
		TeamIDs:        teams,
		RegionIDs:      regions,
		VrrpIDs:        []int{},
		InstanceIDs:    []string{},
		VolumeIDs:      []string{},
		KomputeIDs:     []string{},
		KyloIDs:        []string{},
		KawaiiIDs:      []string{},
		KonveyIDs:      []string{},
		RecordIDs:      []string{},
		PrivateSubnets: map[string]string{},
		ZoneGateways:   map[string]string{},
	}
	p.Quotas.Update(quotas)

	err := p.AllocatePrivateSubnets(subnetSize)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	err = p.AssignZoneGatewayAddresses()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	err = p.CreateDnsZone()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionProjectName, p)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new project %s", p.String())

	// notify project's users
	for _, u := range p.NotifiableUsers() {
		err := NewEmailProjectCreated(&p, u)
		if err != nil {
			klog.Error(err)
			// not a blocker
		}
	}

	return &p, nil
}

func FindProjects() []Project {
	return FindResources[Project](MongoCollectionProjectName)
}

func FindProjectByID(id string) (*Project, error) {
	return FindResourceByID[Project](MongoCollectionProjectName, id)
}

func FindProjectByName(name string) (*Project, error) {
	return FindResourceByName[Project](MongoCollectionProjectName, name)
}

func (p *Project) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionProjectName, p.ID, from, to)
}

func (p *Project) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionProjectName, p.ID, version)
}

func (p *Project) migrateSchemaV2() error {
	err := p.renameDbField("groups", "team_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("regions", "region_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("instances", "instance_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("volumes", "volume_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("kces", "kompute_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("kgws", "kawaii_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("konveys", "konvey_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("kfs", "kylo_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("records", "record_ids")
	if err != nil {
		return err
	}

	err = p.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (p *Project) Resources() []string {
	res := []string{p.String()}
	res = append(res, p.InstanceIDs...)
	res = append(res, p.VolumeIDs...)
	res = append(res, p.KomputeIDs...)
	res = append(res, p.KyloIDs...)
	res = append(res, p.KawaiiIDs...)
	res = append(res, p.KonveyIDs...)
	res = append(res, p.RecordIDs...)
	return res
}

func (p *Project) HasChildren() bool {
	return HasChildRefs(p.InstanceIDs, p.VolumeIDs, p.KomputeIDs, p.KyloIDs, p.KawaiiIDs, p.KonveyIDs, p.RecordIDs)
}

func (p *Project) FindInstances() ([]Instance, error) {
	return FindInstancesByProject(p.String())
}

func (p *Project) FindVolumes() ([]Volume, error) {
	return FindVolumesByProject(p.String())
}

func (p *Project) FindKomputes() ([]Kompute, error) {
	return FindKomputesByProject(p.String())
}

func (p *Project) FindKylo() ([]Kylo, error) {
	return FindKyloByProject(p.String())
}

func (p *Project) NotifiableUsers() []*User {
	users := []*User{}

	for _, teamId := range p.TeamIDs {
		t, err := FindTeamByID(teamId)
		if err != nil {
			klog.Error(err)
			// not a blocker
			continue
		}

		for _, userId := range t.Users() {
			u, err := FindUserByID(userId)
			if err != nil {
				klog.Error(err)
				// not a blocker
				continue
			}
			if !u.NotificationsEnabled {
				continue
			}

			users = append(users, u)
		}
	}

	return users
}

func (p *Project) Update(name, desc, pwd, user, pubkey string, teams, regions, tags []string, meta map[string]string, quotas sdk.ProjectResources) {
	p.UpdateResourceDefaults(name, desc)
	SetFieldStr(&p.RootPassword, pwd)
	SetFieldStr(&p.BootstrapUser, user)
	SetFieldStr(&p.BootstrapPubkey, pubkey)
	p.Tags = tags
	p.Meta = getMetadatas(meta)
	p.Quotas.Update(quotas)
	p.TeamIDs = teams
	p.RegionIDs = regions
	err := p.AssignZoneGatewayAddresses()
	if err != nil {
		klog.Error(err)
	}
	p.Save()
}

func (p *Project) AllocatePrivateSubnets(subnetSize int) error {
	// reserve a subnet in each of project's eligible region
	for _, regionId := range p.RegionIDs {
		r, err := FindRegionByID(regionId)
		if err != nil {
			return err
		}

		s, err := ReservePrivateSubnet(r.String(), p.String(), subnetSize)
		if err != nil {
			return err
		}
		p.PrivateSubnets[r.Name] = s
	}

	return nil
}

func (p *Project) FreePrivateSubnet() error {
	if len(p.PrivateSubnets) == 0 {
		return nil
	}

	for r, subnetId := range p.PrivateSubnets {
		s, err := FindSubnetByID(subnetId)
		if err != nil {
			return err
		}

		s.SetProject("")
		delete(p.PrivateSubnets, r)
	}
	p.Save()

	return nil
}

func (p *Project) GetPrivateSubnet(regionId string) (string, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return "", err
	}

	subnet, ok := p.PrivateSubnets[r.Name]
	if !ok {
		return "", fmt.Errorf("no assigned subnet")
	}
	return subnet, nil
}

func (p *Project) AssignZoneGatewayAddresses() error {
	// reserve local-zone gateways in each subnet in of project's eligible region
	for _, regionId := range p.RegionIDs {
		r, err := FindRegionByID(regionId)
		if err != nil {
			return err
		}

		s, err := FindSubnetByID(p.PrivateSubnets[r.Name])
		if err != nil {
			return err
		}

		gwPoolIPs := s.FindGwPoolIPs()
		klog.Debugf("Project %s subnet for region %s has the following local-zone gateway IPs: %s", p.Name, r.Name, gwPoolIPs)

		if len(gwPoolIPs) < len(r.Zones()) {
			err := fmt.Errorf("Too few gateway IPs in subnet to map all zone's regions")
			klog.Error(err)
			return err
		}

		// alpha-sort zones
		zones := []string{}
		for _, zoneId := range r.Zones() {
			z, err := FindZoneByID(zoneId)
			if err != nil {
				return err
			}
			zones = append(zones, z.Name)
		}
		sort.Strings(zones)

		gwId := 0
		for _, z := range zones {
			klog.Infof("Project %s will use %s as local-zone gateway for zone %s", p.Name, gwPoolIPs[gwId], z)
			p.ZoneGateways[z] = gwPoolIPs[gwId]
			gwId += 1
		}
	}

	return nil
}

func (p *Project) GetZoneGatewayAddress(zoneId string) (string, error) {
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return "", err
	}

	gw, ok := p.ZoneGateways[z.Name]
	if !ok {
		return "", fmt.Errorf("no assigned network gateway")
	}
	return gw, nil
}

func (p *Project) CreateDnsZone() error {
	if p.Domain == "" {
		klog.Warningf("No project domain can be found, ignoring DNS zone creation")
		return nil
	}

	// create zone on all possible network gateways (for now)
	for _, gw := range FindKiwis() {
		err := gw.CreateDnsZone(p.Domain)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Project) DeleteDnsZone() error {
	if p.Domain == "" {
		klog.Warningf("No project domain can be found, ignoring DNS zone deletion")
		return nil
	}

	// delete zone from all possible network gateways (for now)
	for _, gw := range FindKiwis() {
		err := gw.DeleteDnsZone(p.Domain)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Project) Save() {
	p.Updated()
	_, err := GetDB().Update(MongoCollectionProjectName, p.ID, p)
	if err != nil {
		klog.Error(err)
	}
}

func (p *Project) Delete() error {
	klog.Debugf("Deleting project %s", p.String())

	if p.String() == ResourceUnknown {
		return nil
	}

	err := p.FreePrivateSubnet()
	if err != nil {
		klog.Error(err)
		return err
	}

	err = p.DeleteDnsZone()
	if err != nil {
		klog.Error(err)
		// not a blocker
	}

	return GetDB().Delete(MongoCollectionProjectName, p.ID)
}

func (p *Project) Model() sdk.Project {
	metas := []sdk.Metadata{}
	for _, m := range p.Meta {
		meta := m.Model()
		metas = append(metas, meta)
	}
	quotas := p.Quotas.Model()
	privateSubnets := []sdk.RegionSubnet{}
	for regionName, subnetId := range p.PrivateSubnets {
		rs := sdk.RegionSubnet{
			Key:   regionName,
			Value: subnetId,
		}
		privateSubnets = append(privateSubnets, rs)
	}
	vrrp_ids := []int32{}
	for _, id := range p.VrrpIDs {
		vrrp_ids = append(vrrp_ids, int32(id))
	}
	return sdk.Project{
		Id:              p.String(),
		Name:            p.Name,
		Description:     p.Description,
		RootPassword:    p.RootPassword,
		BootstrapUser:   p.BootstrapUser,
		BootstrapPubkey: p.BootstrapPubkey,
		Domain:          p.Domain,
		Teams:           p.TeamIDs,
		Regions:         p.RegionIDs,
		ReservedVrrpIds: vrrp_ids,
		Tags:            p.Tags,
		Metadatas:       metas,
		Quotas:          quotas,
		PrivateSubnets:  privateSubnets,
	}
}

func (p *Project) GetCost() sdk.Cost {

	var price float32 = 0

	// sum all project's instances costs
	for _, instanceId := range p.InstanceIDs {
		i, err := p.Instance(instanceId)
		if err != nil {
			continue
		}
		price += i.Cost.Price
		p.Cost.Currency = i.Cost.Currency
	}

	// sum all project's volumes costs
	for _, volumeId := range p.VolumeIDs {
		v, err := p.Volume(volumeId)
		if err != nil {
			continue
		}
		price += v.Cost.Price
		p.Cost.Currency = v.Cost.Currency
	}

	// save cost back to DB, if changed
	if price != p.Cost.Price {
		p.Cost.Price = price
		p.Save()
	}

	return sdk.Cost{
		Price:    p.Cost.Price,
		Currency: p.Cost.Currency,
	}
}

func (p *Project) GetUsage() sdk.ProjectResources {
	return p.Usage.Model()
}

// VRRP (Virtual Router) IDs
func (p *Project) AllocateVRID() (int, error) {
	for vrrpId := VirtualRouterIdMin; vrrpId <= VirtualRouterIdMax; vrrpId++ {
		if slices.Contains(p.VrrpIDs, vrrpId) {
			continue
		}
		p.VrrpIDs = append(p.VrrpIDs, vrrpId)
		sort.Ints(p.VrrpIDs)
		p.Save()
		return vrrpId, nil
	}

	return 0, fmt.Errorf("Exhausted pool of virtual router IDs")
}

func (p *Project) RemoveVRID(vrrpId int) {
	klog.Debugf("Removing VRRP ID %d from project %s", vrrpId, p.String())
	for idx, id := range p.VrrpIDs {
		if id == vrrpId {
			p.VrrpIDs = append((p.VrrpIDs)[:idx], (p.VrrpIDs)[idx+1:]...)
			break
		}
	}
	p.Save()
}

// Instances

func (p *Project) Instances() []string {
	return p.InstanceIDs
}

func (p *Project) Instance(id string) (*Instance, error) {
	return FindChildByID[Instance](&p.InstanceIDs, id, MongoCollectionInstanceName, ErrProjectNoSuchInstance)
}

func (p *Project) AddInstance(id string) {
	klog.Debugf("Adding instance %s to project %s", id, p.String())
	AddChildRef(&p.InstanceIDs, id)
	p.Save() // save DB before looking back

	// find instance again
	i, err := p.Instance(id)
	if err != nil {
		klog.Error(err)
		return
	}

	// increase usage counters
	p.Usage.InstancesCount += 1
	p.Usage.VCPUs += uint16(i.CPU)
	p.Usage.MemorySize += uint64(i.Memory)

	p.Save()
}

func (p *Project) UpdateInstanceUsage(cpu, mem int64) {
	// increase usage counters
	p.Usage.VCPUs += uint16(cpu)
	p.Usage.MemorySize += uint64(mem)
	p.Save()
}

func (p *Project) RemoveInstance(id string) {
	klog.Debugf("Removing instance %s from project %s", id, p.String())

	// ensure instance exists in project
	ist, err := p.Instance(id)
	if err != nil {
		klog.Error(err)
		return
	}

	// decrease usage counters
	p.Usage.InstancesCount -= 1
	p.Usage.VCPUs -= uint16(ist.CPU)
	p.Usage.MemorySize -= uint64(ist.Memory)

	RemoveChildRef(&p.InstanceIDs, id)
	p.Save()
}

func (p *Project) AllowInstanceCreationOrUpdate(instances, cpu, mem int64) bool {
	// ensure the new instance characteristics comply with the project quota
	if p.Quotas.InstancesCount > 0 {
		if p.Usage.InstancesCount+uint16(instances) > p.Quotas.InstancesCount {
			return false
		}
	}
	if p.Quotas.VCPUs > 0 {
		if p.Usage.VCPUs+uint16(cpu) > p.Quotas.VCPUs {
			return false
		}
	}
	if p.Quotas.MemorySize > 0 {
		if p.Usage.MemorySize+uint64(mem) > p.Quotas.MemorySize {
			return false
		}
	}
	return true
}

// Volumes

func (p *Project) Volumes() []string {
	return p.VolumeIDs
}

func (p *Project) Volume(id string) (*Volume, error) {
	return FindChildByID[Volume](&p.VolumeIDs, id, MongoCollectionVolumeName, ErrProjectNoSuchVolume)
}

func (p *Project) AddVolume(id string) {
	klog.Debugf("Adding volume %s to project %s", id, p.String())
	AddChildRef(&p.VolumeIDs, id)
	p.Save() // save DB before looking back

	// find volume again
	v, err := p.Volume(id)
	if err != nil {
		klog.Error(err)
		return
	}

	// increase usage counters
	p.Usage.StorageSize += uint64(v.Size)

	p.Save()
}

func (p *Project) UpdateVolumeUsage(size int64) {
	// increase usage counters
	p.Usage.StorageSize += uint64(size)
	p.Save()
}

func (p *Project) RemoveVolume(id string) {
	klog.Debugf("Removing volume %s from project %s", id, p.String())

	// ensure volume exists in project
	vol, err := p.Volume(id)
	if err != nil {
		return
	}

	// decrease usage counters
	p.Usage.StorageSize -= uint64(vol.Size)

	RemoveChildRef(&p.VolumeIDs, id)
	p.Save()
}

func (p *Project) AllowVolumeCreationOrUpdate(vol int64) bool {
	// ensure the new volume characteristics comply with the project quota
	if p.Quotas.StorageSize > 0 {
		if p.Usage.StorageSize+uint64(vol) > p.Quotas.StorageSize {
			return false
		}
	}
	return true
}

// Komputes

func (p *Project) Komputes() []string {
	return p.KomputeIDs
}

func (p *Project) FindKomputeByID(id string) (*Kompute, error) {
	return FindChildByID[Kompute](&p.KomputeIDs, id, MongoCollectionKomputeName, ErrProjectNoSuchKompute)
}

func (p *Project) AddKompute(id string) {
	klog.Debugf("Adding Kompute %s to project %s", id, p.String())
	AddChildRef(&p.KomputeIDs, id)
	p.Save()
}

func (p *Project) RemoveKompute(id string) {
	klog.Debugf("Removing Kompute %s from project %s", id, p.String())
	RemoveChildRef(&p.KomputeIDs, id)
	p.Save()
}

// Kawaiis

func (p *Project) Kawaiis() []string {
	return p.KawaiiIDs
}

func (p *Project) FindKawaiiByID(id string) (*Kawaii, error) {
	return FindChildByID[Kawaii](&p.KawaiiIDs, id, MongoCollectionKawaiiName, ErrProjectNoSuchKawaii)
}

func (p *Project) AddKawaii(id string) {
	klog.Debugf("Adding Kawaii %s to project %s", id, p.String())
	AddChildRef(&p.KawaiiIDs, id)
	p.Save()
}

func (p *Project) RemoveKawaii(id string) {
	klog.Debugf("Removing Kawaii %s from project %s", id, p.String())
	RemoveChildRef(&p.KawaiiIDs, id)
	p.Save()
}

// Konvey

func (p *Project) Konveys() []string {
	return p.KonveyIDs
}

func (p *Project) FindKonveyByID(id string) (*Konvey, error) {
	return FindChildByID[Konvey](&p.KonveyIDs, id, MongoCollectionKonveyName, ErrProjectNoSuchKonvey)
}

func (p *Project) AddKonvey(id string) {
	klog.Debugf("Adding Konvey %s to project %s", id, p.String())
	AddChildRef(&p.KonveyIDs, id)
	p.Save()
}

func (p *Project) RemoveKonvey(id string) {
	klog.Debugf("Removing Konvey %s from project %s", id, p.String())
	RemoveChildRef(&p.KonveyIDs, id)
	p.Save()
}

// Kylo

func (p *Project) Kylos() []string {
	return p.KyloIDs
}

func (p *Project) FindKyloByID(id string) (*Kylo, error) {
	return FindChildByID[Kylo](&p.KyloIDs, id, MongoCollectionKyloName, ErrProjectNoSuchKylo)
}

func (p *Project) AddKylo(id string) {
	klog.Debugf("Adding Kylo %s to project %s", id, p.String())
	AddChildRef(&p.KyloIDs, id)
	p.Save()
}

func (p *Project) RemoveKylo(id string) {
	klog.Debugf("Removing Kylo %s from project %s", id, p.String())
	RemoveChildRef(&p.KyloIDs, id)
	p.Save()
}

// DNS Records

func (p *Project) DnsRecords() []string {
	return p.RecordIDs
}

func (p *Project) FindDnsRecordByID(id string) (*DnsRecord, error) {
	return FindChildByID[DnsRecord](&p.RecordIDs, id, MongoCollectionDnsRecordName, ErrProjectNoSuchDnsRecord)
}

func (p *Project) AddDnsRecord(id string) {
	klog.Debugf("Adding DNS Record %s to project %s", id, p.String())
	AddChildRef(&p.RecordIDs, id)
	p.Save()
}

func (p *Project) RemoveDnsRecord(id string) {
	klog.Debugf("Removing DNS Record %s from project %s", id, p.String())
	RemoveChildRef(&p.RecordIDs, id)
	p.Save()
}
