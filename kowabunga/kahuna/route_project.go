/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewProjectRouter() sdk.Router {
	return sdk.NewProjectAPIController(&ProjectService{})
}

type ProjectService struct{}

func (s *ProjectService) CreateProject(ctx context.Context, project sdk.Project, subnetSize int32) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("project", project), RA("subnetSize", subnetSize))

	// check for params
	if project.Name == "" {
		return HttpBadParams(nil)
	}

	// ensure project does not already exists
	_, err := FindProjectByName(project.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create project
	metas := map[string]string{}
	for _, m := range project.Metadatas {
		metas[m.Key] = m.Value
	}

	p, err := NewProject(project.Name, project.Description, project.Domain, project.RootPassword, project.BootstrapUser, project.BootstrapPubkey, project.Teams, project.Regions, project.Tags, metas, project.Quotas, int(subnetSize))
	if err != nil {
		klog.Error(err)
		return HttpServerError(err)
	}

	payload := p.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ProjectService) CreateProjectDnsRecord(ctx context.Context, projectId string, dnsRecord sdk.DnsRecord) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("projectId", projectId), RA("dnsRecord", dnsRecord))

	// ensure project exists
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if dnsRecord.Name == "" || len(dnsRecord.Addresses) == 0 {
		return HttpBadParams(nil)
	}

	// ensure DNS record does not already exists
	_, err = FindDnsRecordByDomainAndName(prj.Domain, dnsRecord.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create DNS record
	r, err := NewDnsRecord(prj.String(), prj.Domain, dnsRecord.Name, dnsRecord.Description, dnsRecord.Addresses)
	if err != nil {
		return HttpServerError(err)
	}

	payload := r.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ProjectService) CreateProjectZoneInstance(ctx context.Context, projectId string, zoneId string, instance sdk.Instance) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("projectId", projectId), RA("zoneId", zoneId), RA("instance", instance))

	// ensure project exists
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure zone exists
	zone, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if instance.Name == "" || instance.Memory == 0 || instance.Vcpus == 0 {
		return HttpBadParams(nil)
	}

	// ensure we're allowed by quotas
	if !prj.AllowInstanceCreationOrUpdate(1, instance.Vcpus, instance.Memory) {
		return HttpQuota(nil)
	}

	// ensure instance does not already exists (globally, across all projects)
	_, err = FindInstanceByName(instance.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// now find the best-suited kaktus node
	h, err := zone.ElectMostFavorableKaktus(instance.Name, zone.Kaktuses())
	if err != nil {
		return HttpServerError(err)
	}

	// create instance
	i, err := NewInstance(prj.String(), h.String(), instance.Name, instance.Description, "", "", instance.Vcpus, instance.Memory, instance.Adapters, instance.Volumes)
	if err != nil {
		return HttpServerError(err)
	}

	payload := i.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ProjectService) CreateProjectZoneKompute(ctx context.Context, projectId string, zoneId string, kompute sdk.Kompute, poolId string, templateId string, public bool) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("projectId", projectId), RA("zoneId", zoneId), RA("kompute", kompute), RA("poolId", poolId), RA("templateId", templateId), RA("public", public))

	// ensure project exists
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure zone exists
	zone, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure region exists
	region, err := zone.Region()
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if kompute.Name == "" || kompute.Memory == 0 || kompute.Vcpus == 0 || kompute.Disk == 0 {
		return HttpBadParams(nil)
	}

	// ensure we're allowed by quotas
	if !prj.AllowInstanceCreationOrUpdate(1, kompute.Vcpus, kompute.Memory) {
		return HttpQuota(nil)
	}
	size := kompute.Disk + kompute.DataDisk
	if !prj.AllowVolumeCreationOrUpdate(size) {
		return HttpQuota(nil)
	}

	// use region's default storage pool unless specified
	pid := region.Defaults.StoragePoolID
	if poolId != "" {
		pid = poolId
	}

	// ensure storage pool exists
	p, err := FindStoragePoolByID(pid)
	if err != nil {
		return HttpNotFound(err)
	}

	// use pool's default template unless specified
	tid := p.Defaults.TemplateIDs.OS
	if templateId != "" {
		tid = templateId
	}

	// ensure template exists
	t, err := FindTemplateByID(tid)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure instance does not already exists (globally, across all projects), this would validate auto-named volumes as well
	_, err = FindInstanceByName(kompute.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// now find the best-suited kaktus node
	h, err := zone.ElectMostFavorableKaktus(kompute.Name, zone.Kaktuses())
	if err != nil {
		return HttpServerError(err)
	}

	//
	// FINALLY, we're done with preflight, let's create something
	//

	// create Kompute
	k, err := NewKompute(prj.String(), zone.String(), h.String(), p.String(), t.String(), kompute.Name, kompute.Description, "", "", kompute.Vcpus, kompute.Memory, kompute.Disk, kompute.DataDisk, public, []string{})
	if err != nil {
		return HttpServerError(err)
	}
	payload := k.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func CreateProjectKonvey(projectId, regionId, name string, konvey sdk.Konvey, kaktusIds []string) (sdk.ImplResponse, error) {
	// converts from model to object
	endpoints := []KonveyEndpoint{}
	for _, ep := range konvey.Endpoints {
		e := KonveyEndpoint{
			Name:     ep.Name,
			Port:     ep.Port,
			Protocol: ep.Protocol,
			Backends: []KonveyBackend{},
		}

		err := IsValidPortListExpression(fmt.Sprintf("%d", ep.Port))
		if err != nil {
			return HttpBadParams(err)
		}

		for _, h := range ep.Backends.Hosts {
			e.Backends = append(e.Backends, KonveyBackend{
				Host: h,
				Port: ep.Backends.Port,
			})
		}

		endpoints = append(endpoints, e)
	}

	// Create Konvey
	k, err := NewKonvey(projectId, regionId, name, konvey.Description, endpoints, kaktusIds)
	if err != nil {
		return HttpServerError(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ProjectService) CreateProjectZoneKonvey(ctx context.Context, projectId string, zoneId string, konvey sdk.Konvey) (sdk.ImplResponse, error) {
	// Create a set of Konvey instances spread across a single zone
	LogHttpRequest(RA("projectId", projectId), RA("zoneId", zoneId), RA("konvey", konvey))

	// ensure project exists
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure zone exists
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	r, err := z.Region()
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure Konvey does not already exists (globally, across all projects)
	kName := konvey.Name
	if kName == "" {
		kName = p.Name
	}
	konveyName := KonveyDefaultNamePrefix + "-" + kName
	_, err = FindKonveyByName(konveyName)
	if err != nil {
		return HttpConflict(err)
	}

	count := 1
	if konvey.Failover {
		count = 2
	}

	// now find the best-suited kaktus nodes
	kaktuses, err := z.ElectMostFavorableKaktuses(konveyName, count)
	if err != nil {
		return HttpServerError(err)
	}

	kaktusIds := []string{}
	for id, h := range kaktuses {
		klog.Debugf("Will use kaktus node %s from zone %s to spawn new Konvey %s #%d", h.Name, z.Name, konveyName, id+1)
		kaktusIds = append(kaktusIds, h.String())
	}

	return CreateProjectKonvey(p.String(), r.String(), konveyName, konvey, kaktusIds)
}

func (s *ProjectService) CreateProjectRegionKylo(ctx context.Context, projectId string, regionId string, kylo sdk.Kylo, nfsId string) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("projectId", projectId), RA("regionId", regionId), RA("kylo", kylo), RA("nfsId", nfsId))

	// ensure project exists
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure region exists
	region, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if kylo.Name == "" || kylo.Access == "" {
		return HttpBadParams(nil)
	}

	// use zone's default NFS storage unless specified
	nid := region.Defaults.NfsID
	if nfsId != "" {
		nid = nfsId
	}

	// ensure NFS storage exists
	nfs, err := FindNfsByID(nid)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure storage does not already exists (globally, across all projects)
	_, err = FindKyloByName(kylo.Name)
	if err == nil {
		return HttpConflict(err)
	}

	//
	// FINALLY, we're done with preflight, let's create something
	//

	// create Kylo
	k, err := NewKylo(prj.String(), region.String(), nfs.String(), kylo.Name, kylo.Description, kylo.Access, kylo.Protocols)
	if err != nil {
		return HttpServerError(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ProjectService) CreateProjectRegionKawaii(ctx context.Context, projectId string, regionId string, kawaii sdk.Kawaii) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("projectId", projectId), RA("regionId", regionId), RA("kawaii", kawaii))

	// ensure project exists
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure Kawaii does not already exists (globally, across all projects)
	kawaiiName := KawaiiDefaultNamePrefix + "-" + p.Name
	_, err = FindKawaiiByName(kawaiiName)
	if err == nil {
		return HttpConflict(err)
	}

	// converts kawaii from model to object
	fw := KawaiiFirewall{
		EgressPolicy: kawaii.Firewall.EgressPolicy,
	}
	if fw.EgressPolicy == "" {
		fw.EgressPolicy = KawaiiFirewallPolicyAccept
	}
	for _, in := range kawaii.Firewall.Ingress {
		err := IsValidPortListExpression(in.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiFirewallIngressRule{
			Source:   in.Source,
			Protocol: in.Protocol,
			Ports:    in.Ports,
		}
		if rule.Source == "" {
			rule.Source = KawaiiFirewallWildcardNetwork
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		fw.Ingress = append(fw.Ingress, rule)
	}
	for _, out := range kawaii.Firewall.Egress {
		err := IsValidPortListExpression(out.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiFirewallEgressRule{
			Destination: out.Destination,
			Protocol:    out.Protocol,
			Ports:       out.Ports,
		}
		if rule.Destination == "" {
			rule.Destination = KawaiiFirewallWildcardNetwork
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		fw.Egress = append(fw.Egress, rule)
	}

	natRules := []KawaiiDNatRule{}
	for _, rule := range kawaii.Dnat {
		err := IsValidPortListExpression(rule.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiDNatRule{
			PrivateIP: rule.Destination,
			Protocol:  rule.Protocol,
			Ports:     rule.Ports,
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		natRules = append(natRules, rule)
	}

	vpcPeerings := []KawaiiVpcPeering{}
	for _, peer := range kawaii.VpcPeerings {
		vpp := KawaiiVpcPeering{
			SubnetID: peer.Subnet,
			Policy:   peer.Policy,
		}
		if vpp.Policy == "" {
			vpp.Policy = KawaiiFirewallPolicyDrop
		}

		vpcSubnet, err := FindSubnetByID(peer.Subnet)
		if err != nil {
			return HttpBadParams(err)
		}

		// enforce application's ports if specified at subnet's level
		if vpcSubnet.Application == SubnetApplicationCeph {
			rule := KawaiiVpcForwardRule{
				Protocol: KawaiiFirewallProtocolTCP,
				Ports:    SubnetApplicationCephPorts,
			}
			vpp.Ingress = append(vpp.Ingress, rule)
			vpp.Egress = append(vpp.Egress, rule)
		}

		for _, in := range peer.Ingress {
			err := IsValidPortListExpression(in.Ports)
			if err != nil {
				return HttpBadParams(err)
			}

			rule := KawaiiVpcForwardRule{
				Protocol: in.Protocol,
				Ports:    in.Ports,
			}
			if rule.Protocol == "" {
				rule.Protocol = KawaiiFirewallProtocolTCP
			}
			vpp.Ingress = append(vpp.Ingress, rule)
		}

		for _, out := range peer.Egress {
			err := IsValidPortListExpression(out.Ports)
			if err != nil {
				return HttpBadParams(err)
			}

			rule := KawaiiVpcForwardRule{
				Protocol: out.Protocol,
				Ports:    out.Ports,
			}
			if rule.Protocol == "" {
				rule.Protocol = KawaiiFirewallProtocolTCP
			}
			vpp.Egress = append(vpp.Egress, rule)
		}

		vpcPeerings = append(vpcPeerings, vpp)
	}

	// Create Kawaii
	k, err := NewKawaii(p.String(), r.String(), kawaiiName, kawaii.Description, fw, natRules, vpcPeerings)
	if err != nil {
		return HttpServerError(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ProjectService) CreateProjectRegionKonvey(ctx context.Context, projectId string, regionId string, konvey sdk.Konvey) (sdk.ImplResponse, error) {
	// Create a set of Konvey instances spread across multiple zones, when applicable
	LogHttpRequest(RA("projectId", projectId), RA("regionId", regionId), RA("konvey", konvey))

	// ensure project exists
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure Konvey does not already exists (globally, across all projects)
	kName := konvey.Name
	if kName == "" {
		kName = p.Name
	}
	konveyName := KonveyDefaultNamePrefix + "-" + kName
	_, err = FindKonveyByName(konveyName)
	if err == nil {
		return HttpConflict(err)
	}

	count := 1
	if konvey.Failover {
		count = 2
	}

	// find the best-suited zones(s)
	bestZones, err := r.ElectMostFavorableZones(count)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if single-zoned
	if len(bestZones) < count {
		for i := 0; i < count-len(bestZones); i++ {
			bestZones = append(bestZones, bestZones[0])
		}
	}

	// now find the best-suited kaktus node
	kaktusIds := []string{}
	for id, zoneId := range bestZones {
		z, err := FindZoneByID(zoneId)
		if err != nil {
			return HttpNotFound(err)
		}

		// consider kaktus nodes that were not already assigned in the loop
		eligibleKaktuses := z.Kaktuses()
		for _, kaktusId := range kaktusIds {
			RemoveChildRef(&eligibleKaktuses, kaktusId)
		}
		if len(eligibleKaktuses) == 0 {
			eligibleKaktuses = z.Kaktuses()
		}

		h, err := z.ElectMostFavorableKaktus(konveyName, eligibleKaktuses)
		if err != nil {
			return HttpServerError(err)
		}

		klog.Debugf("Will use kaktus node %s from zone %s to spawn new Konvey %s #%d", h.Name, z.Name, konveyName, id+1)
		kaktusIds = append(kaktusIds, h.String())
	}

	return CreateProjectKonvey(p.String(), r.String(), konveyName, konvey, kaktusIds)
}

func (s *ProjectService) CreateProjectRegionVolume(ctx context.Context, projectId string, regionId string, volume sdk.Volume, poolId string, templateId string) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("projectId", projectId), RA("regionId", regionId), RA("volume", volume), RA("pooId", poolId), RA("templateId", templateId))

	// ensure project exists
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure region exists
	region, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// use region's default storage pool unless specified
	pid := region.Defaults.StoragePoolID
	if poolId != "" {
		pid = poolId
	}

	// check for storage pool
	p, err := FindStoragePoolByID(pid)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if volume.Name == "" || volume.Type == "" || volume.Size == 0 {
		return HttpBadParams(nil)
	}

	// ensure we're allowed by quotas
	if !prj.AllowVolumeCreationOrUpdate(volume.Size) {
		return HttpQuota(nil)
	}

	// use pool's default template unless specified
	tid := p.Defaults.TemplateIDs.OS
	if templateId != "" {
		tid = templateId
	}

	// ensure volume does not already exists
	_, err = FindVolumeByName(volume.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create volume
	v, err := NewVolume(prj.String(), p.String(), tid, volume.Name, volume.Description, volume.Type, volume.Size)
	if err != nil {
		return HttpServerError(err)
	}

	payload := v.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ProjectService) DeleteProject(ctx context.Context, projectId string) (sdk.ImplResponse, error) {
	// ensure project exists
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if project still has children referenced
	if len(p.Instances()) != 0 || len(p.Volumes()) != 0 {
		return HttpConflict(nil)
	}

	// remove project
	err = p.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *ProjectService) ListProjectDnsRecords(ctx context.Context, projectId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.DnsRecords()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjectZoneInstances(ctx context.Context, projectId string, zoneId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Instances()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjectZoneKomputes(ctx context.Context, projectId string, zoneId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Komputes()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjectZoneKonveys(ctx context.Context, projectId string, zoneId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Konveys()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjectRegionKylos(ctx context.Context, projectId string, regionId string, nfsId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Kylos()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjectRegionKawaiis(ctx context.Context, projectId string, regionId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Kawaiis()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjectRegionKonveys(ctx context.Context, projectId string, regionId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Konveys()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjectRegionVolumes(ctx context.Context, projectId string, regionId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Volumes()
	return HttpOK(payload)
}

func (s *ProjectService) ListProjects(ctx context.Context, subnetSize int32) (sdk.ImplResponse, error) {
	projects := FindProjects()
	var payload []string
	for _, t := range projects {
		payload = append(payload, t.String())
	}

	return HttpOK(payload)
}

func (s *ProjectService) ReadProject(ctx context.Context, projectId string) (sdk.ImplResponse, error) {
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *ProjectService) ReadProjectCost(ctx context.Context, projectId string) (sdk.ImplResponse, error) {
	// ensure project exists
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// get project cost model
	payload := p.GetCost()
	return HttpOK(payload)
}

func (s *ProjectService) ReadProjectUsage(ctx context.Context, projectId string) (sdk.ImplResponse, error) {
	// ensure project exists
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// get project usage model
	payload := p.GetUsage()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *ProjectService) UpdateProject(ctx context.Context, projectId string, project sdk.Project) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("projectId", projectId), RA("project", project))

	// check for params
	if project.Name == "" {
		return HttpBadParams(nil)
	}

	// ensure project exists
	p, err := FindProjectByID(projectId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update project
	metas := map[string]string{}
	for _, m := range project.Metadatas {
		metas[m.Key] = m.Value
	}
	p.Update(project.Name, project.Description, project.RootPassword, project.BootstrapUser, project.BootstrapPubkey, project.Teams, project.Regions, project.Tags, metas, project.Quotas)

	payload := p.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
