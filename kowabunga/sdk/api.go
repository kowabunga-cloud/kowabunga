// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

/*
 * Kowabunga API documentation
 *
 * Kvm Orchestrator With A BUNch of Goods Added
 *
 * API version: 0.52.5
 * Contact: maintainers@kowabunga.cloud
 */

package sdk

import (
	"context"
	"net/http"
)



// AdapterAPIRouter defines the required methods for binding the api requests to a responses for the AdapterAPI
// The AdapterAPIRouter implementation should parse necessary information from the http request,
// pass the data to a AdapterAPIServicer to perform the required actions, then write the service results to the http response.
type AdapterAPIRouter interface { 
	ListAdapters(http.ResponseWriter, *http.Request)
	ReadAdapter(http.ResponseWriter, *http.Request)
	UpdateAdapter(http.ResponseWriter, *http.Request)
	DeleteAdapter(http.ResponseWriter, *http.Request)
}
// AgentAPIRouter defines the required methods for binding the api requests to a responses for the AgentAPI
// The AgentAPIRouter implementation should parse necessary information from the http request,
// pass the data to a AgentAPIServicer to perform the required actions, then write the service results to the http response.
type AgentAPIRouter interface { 
	ListAgents(http.ResponseWriter, *http.Request)
	CreateAgent(http.ResponseWriter, *http.Request)
	ReadAgent(http.ResponseWriter, *http.Request)
	UpdateAgent(http.ResponseWriter, *http.Request)
	DeleteAgent(http.ResponseWriter, *http.Request)
	SetAgentApiToken(http.ResponseWriter, *http.Request)
}
// InstanceAPIRouter defines the required methods for binding the api requests to a responses for the InstanceAPI
// The InstanceAPIRouter implementation should parse necessary information from the http request,
// pass the data to a InstanceAPIServicer to perform the required actions, then write the service results to the http response.
type InstanceAPIRouter interface { 
	ListInstances(http.ResponseWriter, *http.Request)
	ReadInstance(http.ResponseWriter, *http.Request)
	UpdateInstance(http.ResponseWriter, *http.Request)
	DeleteInstance(http.ResponseWriter, *http.Request)
	ReadInstanceState(http.ResponseWriter, *http.Request)
	RebootInstance(http.ResponseWriter, *http.Request)
	ResetInstance(http.ResponseWriter, *http.Request)
	SuspendInstance(http.ResponseWriter, *http.Request)
	ResumeInstance(http.ResponseWriter, *http.Request)
	StartInstance(http.ResponseWriter, *http.Request)
	StopInstance(http.ResponseWriter, *http.Request)
	ShutdownInstance(http.ResponseWriter, *http.Request)
	ReadInstanceRemoteConnection(http.ResponseWriter, *http.Request)
}
// KaktusAPIRouter defines the required methods for binding the api requests to a responses for the KaktusAPI
// The KaktusAPIRouter implementation should parse necessary information from the http request,
// pass the data to a KaktusAPIServicer to perform the required actions, then write the service results to the http response.
type KaktusAPIRouter interface { 
	ListKaktuss(http.ResponseWriter, *http.Request)
	ReadKaktus(http.ResponseWriter, *http.Request)
	UpdateKaktus(http.ResponseWriter, *http.Request)
	DeleteKaktus(http.ResponseWriter, *http.Request)
	ReadKaktusCaps(http.ResponseWriter, *http.Request)
	ListKaktusInstances(http.ResponseWriter, *http.Request)
}
// KawaiiAPIRouter defines the required methods for binding the api requests to a responses for the KawaiiAPI
// The KawaiiAPIRouter implementation should parse necessary information from the http request,
// pass the data to a KawaiiAPIServicer to perform the required actions, then write the service results to the http response.
type KawaiiAPIRouter interface { 
	ListKawaiis(http.ResponseWriter, *http.Request)
	ReadKawaii(http.ResponseWriter, *http.Request)
	UpdateKawaii(http.ResponseWriter, *http.Request)
	DeleteKawaii(http.ResponseWriter, *http.Request)
	ListKawaiiIpSecs(http.ResponseWriter, *http.Request)
	CreateKawaiiIpSec(http.ResponseWriter, *http.Request)
	ReadKawaiiIpSec(http.ResponseWriter, *http.Request)
	UpdateKawaiiIpSec(http.ResponseWriter, *http.Request)
	DeleteKawaiiIpSec(http.ResponseWriter, *http.Request)
}
// KiwiAPIRouter defines the required methods for binding the api requests to a responses for the KiwiAPI
// The KiwiAPIRouter implementation should parse necessary information from the http request,
// pass the data to a KiwiAPIServicer to perform the required actions, then write the service results to the http response.
type KiwiAPIRouter interface { 
	ListKiwis(http.ResponseWriter, *http.Request)
	ReadKiwi(http.ResponseWriter, *http.Request)
	UpdateKiwi(http.ResponseWriter, *http.Request)
	DeleteKiwi(http.ResponseWriter, *http.Request)
}
// KomputeAPIRouter defines the required methods for binding the api requests to a responses for the KomputeAPI
// The KomputeAPIRouter implementation should parse necessary information from the http request,
// pass the data to a KomputeAPIServicer to perform the required actions, then write the service results to the http response.
type KomputeAPIRouter interface { 
	ListKomputes(http.ResponseWriter, *http.Request)
	ReadKompute(http.ResponseWriter, *http.Request)
	UpdateKompute(http.ResponseWriter, *http.Request)
	DeleteKompute(http.ResponseWriter, *http.Request)
	ReadKomputeState(http.ResponseWriter, *http.Request)
	RebootKompute(http.ResponseWriter, *http.Request)
	ResetKompute(http.ResponseWriter, *http.Request)
	SuspendKompute(http.ResponseWriter, *http.Request)
	ResumeKompute(http.ResponseWriter, *http.Request)
	StartKompute(http.ResponseWriter, *http.Request)
	StopKompute(http.ResponseWriter, *http.Request)
	ShutdownKompute(http.ResponseWriter, *http.Request)
}
// KonveyAPIRouter defines the required methods for binding the api requests to a responses for the KonveyAPI
// The KonveyAPIRouter implementation should parse necessary information from the http request,
// pass the data to a KonveyAPIServicer to perform the required actions, then write the service results to the http response.
type KonveyAPIRouter interface { 
	ListKonveys(http.ResponseWriter, *http.Request)
	ReadKonvey(http.ResponseWriter, *http.Request)
	UpdateKonvey(http.ResponseWriter, *http.Request)
	DeleteKonvey(http.ResponseWriter, *http.Request)
}
// KyloAPIRouter defines the required methods for binding the api requests to a responses for the KyloAPI
// The KyloAPIRouter implementation should parse necessary information from the http request,
// pass the data to a KyloAPIServicer to perform the required actions, then write the service results to the http response.
type KyloAPIRouter interface { 
	ListKylos(http.ResponseWriter, *http.Request)
	ReadKylo(http.ResponseWriter, *http.Request)
	UpdateKylo(http.ResponseWriter, *http.Request)
	DeleteKylo(http.ResponseWriter, *http.Request)
}
// NfsAPIRouter defines the required methods for binding the api requests to a responses for the NfsAPI
// The NfsAPIRouter implementation should parse necessary information from the http request,
// pass the data to a NfsAPIServicer to perform the required actions, then write the service results to the http response.
type NfsAPIRouter interface { 
	ListStorageNFSs(http.ResponseWriter, *http.Request)
	ReadStorageNFS(http.ResponseWriter, *http.Request)
	UpdateStorageNFS(http.ResponseWriter, *http.Request)
	DeleteStorageNFS(http.ResponseWriter, *http.Request)
	ListStorageNFSKylos(http.ResponseWriter, *http.Request)
}
// PoolAPIRouter defines the required methods for binding the api requests to a responses for the PoolAPI
// The PoolAPIRouter implementation should parse necessary information from the http request,
// pass the data to a PoolAPIServicer to perform the required actions, then write the service results to the http response.
type PoolAPIRouter interface { 
	ListStoragePools(http.ResponseWriter, *http.Request)
	ReadStoragePool(http.ResponseWriter, *http.Request)
	UpdateStoragePool(http.ResponseWriter, *http.Request)
	DeleteStoragePool(http.ResponseWriter, *http.Request)
	ListStoragePoolVolumes(http.ResponseWriter, *http.Request)
	CreateTemplate(http.ResponseWriter, *http.Request)
	SetStoragePoolDefaultTemplate(http.ResponseWriter, *http.Request)
	ListStoragePoolTemplates(http.ResponseWriter, *http.Request)
}
// ProjectAPIRouter defines the required methods for binding the api requests to a responses for the ProjectAPI
// The ProjectAPIRouter implementation should parse necessary information from the http request,
// pass the data to a ProjectAPIServicer to perform the required actions, then write the service results to the http response.
type ProjectAPIRouter interface { 
	ListProjects(http.ResponseWriter, *http.Request)
	CreateProject(http.ResponseWriter, *http.Request)
	ReadProject(http.ResponseWriter, *http.Request)
	UpdateProject(http.ResponseWriter, *http.Request)
	DeleteProject(http.ResponseWriter, *http.Request)
	ReadProjectCost(http.ResponseWriter, *http.Request)
	ReadProjectUsage(http.ResponseWriter, *http.Request)
	CreateProjectDnsRecord(http.ResponseWriter, *http.Request)
	ListProjectDnsRecords(http.ResponseWriter, *http.Request)
	CreateProjectRegionVolume(http.ResponseWriter, *http.Request)
	ListProjectRegionVolumes(http.ResponseWriter, *http.Request)
	CreateProjectZoneInstance(http.ResponseWriter, *http.Request)
	ListProjectZoneInstances(http.ResponseWriter, *http.Request)
	CreateProjectZoneKompute(http.ResponseWriter, *http.Request)
	ListProjectZoneKomputes(http.ResponseWriter, *http.Request)
	ListProjectRegionKylos(http.ResponseWriter, *http.Request)
	CreateProjectRegionKylo(http.ResponseWriter, *http.Request)
	CreateProjectRegionKawaii(http.ResponseWriter, *http.Request)
	ListProjectRegionKawaiis(http.ResponseWriter, *http.Request)
	CreateProjectZoneKonvey(http.ResponseWriter, *http.Request)
	ListProjectZoneKonveys(http.ResponseWriter, *http.Request)
	CreateProjectRegionKonvey(http.ResponseWriter, *http.Request)
	ListProjectRegionKonveys(http.ResponseWriter, *http.Request)
}
// RecordAPIRouter defines the required methods for binding the api requests to a responses for the RecordAPI
// The RecordAPIRouter implementation should parse necessary information from the http request,
// pass the data to a RecordAPIServicer to perform the required actions, then write the service results to the http response.
type RecordAPIRouter interface { 
	ReadDnsRecord(http.ResponseWriter, *http.Request)
	UpdateDnsRecord(http.ResponseWriter, *http.Request)
	DeleteDnsRecord(http.ResponseWriter, *http.Request)
}
// RegionAPIRouter defines the required methods for binding the api requests to a responses for the RegionAPI
// The RegionAPIRouter implementation should parse necessary information from the http request,
// pass the data to a RegionAPIServicer to perform the required actions, then write the service results to the http response.
type RegionAPIRouter interface { 
	ListRegions(http.ResponseWriter, *http.Request)
	CreateRegion(http.ResponseWriter, *http.Request)
	ReadRegion(http.ResponseWriter, *http.Request)
	UpdateRegion(http.ResponseWriter, *http.Request)
	DeleteRegion(http.ResponseWriter, *http.Request)
	CreateZone(http.ResponseWriter, *http.Request)
	ListRegionZones(http.ResponseWriter, *http.Request)
	CreateStoragePool(http.ResponseWriter, *http.Request)
	CreateKiwi(http.ResponseWriter, *http.Request)
	ListRegionKiwis(http.ResponseWriter, *http.Request)
	CreateVNet(http.ResponseWriter, *http.Request)
	ListRegionVNets(http.ResponseWriter, *http.Request)
	SetRegionDefaultStoragePool(http.ResponseWriter, *http.Request)
	ListRegionStoragePools(http.ResponseWriter, *http.Request)
	ListRegionStorageNFSs(http.ResponseWriter, *http.Request)
	CreateStorageNFS(http.ResponseWriter, *http.Request)
	SetRegionDefaultStorageNFS(http.ResponseWriter, *http.Request)
}
// SubnetAPIRouter defines the required methods for binding the api requests to a responses for the SubnetAPI
// The SubnetAPIRouter implementation should parse necessary information from the http request,
// pass the data to a SubnetAPIServicer to perform the required actions, then write the service results to the http response.
type SubnetAPIRouter interface { 
	ListSubnets(http.ResponseWriter, *http.Request)
	ReadSubnet(http.ResponseWriter, *http.Request)
	UpdateSubnet(http.ResponseWriter, *http.Request)
	DeleteSubnet(http.ResponseWriter, *http.Request)
	CreateAdapter(http.ResponseWriter, *http.Request)
	ListSubnetAdapters(http.ResponseWriter, *http.Request)
}
// TeamAPIRouter defines the required methods for binding the api requests to a responses for the TeamAPI
// The TeamAPIRouter implementation should parse necessary information from the http request,
// pass the data to a TeamAPIServicer to perform the required actions, then write the service results to the http response.
type TeamAPIRouter interface { 
	ListTeams(http.ResponseWriter, *http.Request)
	CreateTeam(http.ResponseWriter, *http.Request)
	ReadTeam(http.ResponseWriter, *http.Request)
	UpdateTeam(http.ResponseWriter, *http.Request)
	DeleteTeam(http.ResponseWriter, *http.Request)
}
// TemplateAPIRouter defines the required methods for binding the api requests to a responses for the TemplateAPI
// The TemplateAPIRouter implementation should parse necessary information from the http request,
// pass the data to a TemplateAPIServicer to perform the required actions, then write the service results to the http response.
type TemplateAPIRouter interface { 
	ListTemplates(http.ResponseWriter, *http.Request)
	ReadTemplate(http.ResponseWriter, *http.Request)
	UpdateTemplate(http.ResponseWriter, *http.Request)
	DeleteTemplate(http.ResponseWriter, *http.Request)
}
// TokenAPIRouter defines the required methods for binding the api requests to a responses for the TokenAPI
// The TokenAPIRouter implementation should parse necessary information from the http request,
// pass the data to a TokenAPIServicer to perform the required actions, then write the service results to the http response.
type TokenAPIRouter interface { 
	ListApiTokens(http.ResponseWriter, *http.Request)
	ReadApiToken(http.ResponseWriter, *http.Request)
	UpdateApiToken(http.ResponseWriter, *http.Request)
	DeleteApiToken(http.ResponseWriter, *http.Request)
}
// UserAPIRouter defines the required methods for binding the api requests to a responses for the UserAPI
// The UserAPIRouter implementation should parse necessary information from the http request,
// pass the data to a UserAPIServicer to perform the required actions, then write the service results to the http response.
type UserAPIRouter interface { 
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	ResetPassword(http.ResponseWriter, *http.Request)
	ListUsers(http.ResponseWriter, *http.Request)
	CreateUser(http.ResponseWriter, *http.Request)
	ReadUser(http.ResponseWriter, *http.Request)
	UpdateUser(http.ResponseWriter, *http.Request)
	DeleteUser(http.ResponseWriter, *http.Request)
	SetUserPassword(http.ResponseWriter, *http.Request)
	ResetUserPassword(http.ResponseWriter, *http.Request)
	SetUserApiToken(http.ResponseWriter, *http.Request)
}
// VnetAPIRouter defines the required methods for binding the api requests to a responses for the VnetAPI
// The VnetAPIRouter implementation should parse necessary information from the http request,
// pass the data to a VnetAPIServicer to perform the required actions, then write the service results to the http response.
type VnetAPIRouter interface { 
	ListVNets(http.ResponseWriter, *http.Request)
	ReadVNet(http.ResponseWriter, *http.Request)
	UpdateVNet(http.ResponseWriter, *http.Request)
	DeleteVNet(http.ResponseWriter, *http.Request)
	CreateSubnet(http.ResponseWriter, *http.Request)
	SetVNetDefaultSubnet(http.ResponseWriter, *http.Request)
	ListVNetSubnets(http.ResponseWriter, *http.Request)
}
// VolumeAPIRouter defines the required methods for binding the api requests to a responses for the VolumeAPI
// The VolumeAPIRouter implementation should parse necessary information from the http request,
// pass the data to a VolumeAPIServicer to perform the required actions, then write the service results to the http response.
type VolumeAPIRouter interface { 
	ListVolumes(http.ResponseWriter, *http.Request)
	ReadVolume(http.ResponseWriter, *http.Request)
	UpdateVolume(http.ResponseWriter, *http.Request)
	DeleteVolume(http.ResponseWriter, *http.Request)
}
// ZoneAPIRouter defines the required methods for binding the api requests to a responses for the ZoneAPI
// The ZoneAPIRouter implementation should parse necessary information from the http request,
// pass the data to a ZoneAPIServicer to perform the required actions, then write the service results to the http response.
type ZoneAPIRouter interface { 
	ListZones(http.ResponseWriter, *http.Request)
	ReadZone(http.ResponseWriter, *http.Request)
	UpdateZone(http.ResponseWriter, *http.Request)
	DeleteZone(http.ResponseWriter, *http.Request)
	CreateKaktus(http.ResponseWriter, *http.Request)
	ListZoneKaktuses(http.ResponseWriter, *http.Request)
}


// AdapterAPIServicer defines the api actions for the AdapterAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type AdapterAPIServicer interface { 
	ListAdapters(context.Context) (ImplResponse, error)
	ReadAdapter(context.Context, string) (ImplResponse, error)
	UpdateAdapter(context.Context, string, Adapter) (ImplResponse, error)
	DeleteAdapter(context.Context, string) (ImplResponse, error)
}


// AgentAPIServicer defines the api actions for the AgentAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type AgentAPIServicer interface { 
	ListAgents(context.Context) (ImplResponse, error)
	CreateAgent(context.Context, Agent) (ImplResponse, error)
	ReadAgent(context.Context, string) (ImplResponse, error)
	UpdateAgent(context.Context, string, Agent) (ImplResponse, error)
	DeleteAgent(context.Context, string) (ImplResponse, error)
	SetAgentApiToken(context.Context, string, bool, string) (ImplResponse, error)
}


// InstanceAPIServicer defines the api actions for the InstanceAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type InstanceAPIServicer interface { 
	ListInstances(context.Context) (ImplResponse, error)
	ReadInstance(context.Context, string) (ImplResponse, error)
	UpdateInstance(context.Context, string, Instance) (ImplResponse, error)
	DeleteInstance(context.Context, string) (ImplResponse, error)
	ReadInstanceState(context.Context, string) (ImplResponse, error)
	RebootInstance(context.Context, string) (ImplResponse, error)
	ResetInstance(context.Context, string) (ImplResponse, error)
	SuspendInstance(context.Context, string) (ImplResponse, error)
	ResumeInstance(context.Context, string) (ImplResponse, error)
	StartInstance(context.Context, string) (ImplResponse, error)
	StopInstance(context.Context, string) (ImplResponse, error)
	ShutdownInstance(context.Context, string) (ImplResponse, error)
	ReadInstanceRemoteConnection(context.Context, string) (ImplResponse, error)
}


// KaktusAPIServicer defines the api actions for the KaktusAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type KaktusAPIServicer interface { 
	ListKaktuss(context.Context) (ImplResponse, error)
	ReadKaktus(context.Context, string) (ImplResponse, error)
	UpdateKaktus(context.Context, string, Kaktus) (ImplResponse, error)
	DeleteKaktus(context.Context, string) (ImplResponse, error)
	ReadKaktusCaps(context.Context, string) (ImplResponse, error)
	ListKaktusInstances(context.Context, string) (ImplResponse, error)
}


// KawaiiAPIServicer defines the api actions for the KawaiiAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type KawaiiAPIServicer interface { 
	ListKawaiis(context.Context) (ImplResponse, error)
	ReadKawaii(context.Context, string) (ImplResponse, error)
	UpdateKawaii(context.Context, string, Kawaii) (ImplResponse, error)
	DeleteKawaii(context.Context, string) (ImplResponse, error)
	ListKawaiiIpSecs(context.Context, string) (ImplResponse, error)
	CreateKawaiiIpSec(context.Context, string, KawaiiIpSec) (ImplResponse, error)
	ReadKawaiiIpSec(context.Context, string, string) (ImplResponse, error)
	UpdateKawaiiIpSec(context.Context, string, string, KawaiiIpSec) (ImplResponse, error)
	DeleteKawaiiIpSec(context.Context, string, string) (ImplResponse, error)
}


// KiwiAPIServicer defines the api actions for the KiwiAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type KiwiAPIServicer interface { 
	ListKiwis(context.Context) (ImplResponse, error)
	ReadKiwi(context.Context, string) (ImplResponse, error)
	UpdateKiwi(context.Context, string, Kiwi) (ImplResponse, error)
	DeleteKiwi(context.Context, string) (ImplResponse, error)
}


// KomputeAPIServicer defines the api actions for the KomputeAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type KomputeAPIServicer interface { 
	ListKomputes(context.Context) (ImplResponse, error)
	ReadKompute(context.Context, string) (ImplResponse, error)
	UpdateKompute(context.Context, string, Kompute) (ImplResponse, error)
	DeleteKompute(context.Context, string) (ImplResponse, error)
	ReadKomputeState(context.Context, string) (ImplResponse, error)
	RebootKompute(context.Context, string) (ImplResponse, error)
	ResetKompute(context.Context, string) (ImplResponse, error)
	SuspendKompute(context.Context, string) (ImplResponse, error)
	ResumeKompute(context.Context, string) (ImplResponse, error)
	StartKompute(context.Context, string) (ImplResponse, error)
	StopKompute(context.Context, string) (ImplResponse, error)
	ShutdownKompute(context.Context, string) (ImplResponse, error)
}


// KonveyAPIServicer defines the api actions for the KonveyAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type KonveyAPIServicer interface { 
	ListKonveys(context.Context) (ImplResponse, error)
	ReadKonvey(context.Context, string) (ImplResponse, error)
	UpdateKonvey(context.Context, string, Konvey) (ImplResponse, error)
	DeleteKonvey(context.Context, string) (ImplResponse, error)
}


// KyloAPIServicer defines the api actions for the KyloAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type KyloAPIServicer interface { 
	ListKylos(context.Context) (ImplResponse, error)
	ReadKylo(context.Context, string) (ImplResponse, error)
	UpdateKylo(context.Context, string, Kylo) (ImplResponse, error)
	DeleteKylo(context.Context, string) (ImplResponse, error)
}


// NfsAPIServicer defines the api actions for the NfsAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type NfsAPIServicer interface { 
	ListStorageNFSs(context.Context) (ImplResponse, error)
	ReadStorageNFS(context.Context, string) (ImplResponse, error)
	UpdateStorageNFS(context.Context, string, StorageNfs) (ImplResponse, error)
	DeleteStorageNFS(context.Context, string) (ImplResponse, error)
	ListStorageNFSKylos(context.Context, string) (ImplResponse, error)
}


// PoolAPIServicer defines the api actions for the PoolAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type PoolAPIServicer interface { 
	ListStoragePools(context.Context) (ImplResponse, error)
	ReadStoragePool(context.Context, string) (ImplResponse, error)
	UpdateStoragePool(context.Context, string, StoragePool) (ImplResponse, error)
	DeleteStoragePool(context.Context, string) (ImplResponse, error)
	ListStoragePoolVolumes(context.Context, string) (ImplResponse, error)
	CreateTemplate(context.Context, string, Template) (ImplResponse, error)
	SetStoragePoolDefaultTemplate(context.Context, string, string) (ImplResponse, error)
	ListStoragePoolTemplates(context.Context, string) (ImplResponse, error)
}


// ProjectAPIServicer defines the api actions for the ProjectAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type ProjectAPIServicer interface { 
	ListProjects(context.Context, int32) (ImplResponse, error)
	CreateProject(context.Context, Project, int32) (ImplResponse, error)
	ReadProject(context.Context, string) (ImplResponse, error)
	UpdateProject(context.Context, string, Project) (ImplResponse, error)
	DeleteProject(context.Context, string) (ImplResponse, error)
	ReadProjectCost(context.Context, string) (ImplResponse, error)
	ReadProjectUsage(context.Context, string) (ImplResponse, error)
	CreateProjectDnsRecord(context.Context, string, DnsRecord) (ImplResponse, error)
	ListProjectDnsRecords(context.Context, string) (ImplResponse, error)
	CreateProjectRegionVolume(context.Context, string, string, Volume, string, string) (ImplResponse, error)
	ListProjectRegionVolumes(context.Context, string, string) (ImplResponse, error)
	CreateProjectZoneInstance(context.Context, string, string, Instance) (ImplResponse, error)
	ListProjectZoneInstances(context.Context, string, string) (ImplResponse, error)
	CreateProjectZoneKompute(context.Context, string, string, Kompute, string, string, bool) (ImplResponse, error)
	ListProjectZoneKomputes(context.Context, string, string) (ImplResponse, error)
	ListProjectRegionKylos(context.Context, string, string, string) (ImplResponse, error)
	CreateProjectRegionKylo(context.Context, string, string, Kylo, string) (ImplResponse, error)
	CreateProjectRegionKawaii(context.Context, string, string, Kawaii) (ImplResponse, error)
	ListProjectRegionKawaiis(context.Context, string, string) (ImplResponse, error)
	CreateProjectZoneKonvey(context.Context, string, string, Konvey) (ImplResponse, error)
	ListProjectZoneKonveys(context.Context, string, string) (ImplResponse, error)
	CreateProjectRegionKonvey(context.Context, string, string, Konvey) (ImplResponse, error)
	ListProjectRegionKonveys(context.Context, string, string) (ImplResponse, error)
}


// RecordAPIServicer defines the api actions for the RecordAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type RecordAPIServicer interface { 
	ReadDnsRecord(context.Context, string) (ImplResponse, error)
	UpdateDnsRecord(context.Context, string, DnsRecord) (ImplResponse, error)
	DeleteDnsRecord(context.Context, string) (ImplResponse, error)
}


// RegionAPIServicer defines the api actions for the RegionAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type RegionAPIServicer interface { 
	ListRegions(context.Context) (ImplResponse, error)
	CreateRegion(context.Context, Region) (ImplResponse, error)
	ReadRegion(context.Context, string) (ImplResponse, error)
	UpdateRegion(context.Context, string, Region) (ImplResponse, error)
	DeleteRegion(context.Context, string) (ImplResponse, error)
	CreateZone(context.Context, string, Zone) (ImplResponse, error)
	ListRegionZones(context.Context, string) (ImplResponse, error)
	CreateStoragePool(context.Context, string, StoragePool) (ImplResponse, error)
	CreateKiwi(context.Context, string, Kiwi) (ImplResponse, error)
	ListRegionKiwis(context.Context, string) (ImplResponse, error)
	CreateVNet(context.Context, string, VNet) (ImplResponse, error)
	ListRegionVNets(context.Context, string) (ImplResponse, error)
	SetRegionDefaultStoragePool(context.Context, string, string) (ImplResponse, error)
	ListRegionStoragePools(context.Context, string) (ImplResponse, error)
	ListRegionStorageNFSs(context.Context, string, string) (ImplResponse, error)
	CreateStorageNFS(context.Context, string, StorageNfs, string) (ImplResponse, error)
	SetRegionDefaultStorageNFS(context.Context, string, string) (ImplResponse, error)
}


// SubnetAPIServicer defines the api actions for the SubnetAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type SubnetAPIServicer interface { 
	ListSubnets(context.Context) (ImplResponse, error)
	ReadSubnet(context.Context, string) (ImplResponse, error)
	UpdateSubnet(context.Context, string, Subnet) (ImplResponse, error)
	DeleteSubnet(context.Context, string) (ImplResponse, error)
	CreateAdapter(context.Context, string, Adapter, bool) (ImplResponse, error)
	ListSubnetAdapters(context.Context, string) (ImplResponse, error)
}


// TeamAPIServicer defines the api actions for the TeamAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type TeamAPIServicer interface { 
	ListTeams(context.Context) (ImplResponse, error)
	CreateTeam(context.Context, Team) (ImplResponse, error)
	ReadTeam(context.Context, string) (ImplResponse, error)
	UpdateTeam(context.Context, string, Team) (ImplResponse, error)
	DeleteTeam(context.Context, string) (ImplResponse, error)
}


// TemplateAPIServicer defines the api actions for the TemplateAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type TemplateAPIServicer interface { 
	ListTemplates(context.Context) (ImplResponse, error)
	ReadTemplate(context.Context, string) (ImplResponse, error)
	UpdateTemplate(context.Context, string, Template) (ImplResponse, error)
	DeleteTemplate(context.Context, string) (ImplResponse, error)
}


// TokenAPIServicer defines the api actions for the TokenAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type TokenAPIServicer interface { 
	ListApiTokens(context.Context) (ImplResponse, error)
	ReadApiToken(context.Context, string) (ImplResponse, error)
	UpdateApiToken(context.Context, string, ApiToken) (ImplResponse, error)
	DeleteApiToken(context.Context, string) (ImplResponse, error)
}


// UserAPIServicer defines the api actions for the UserAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type UserAPIServicer interface { 
	Login(context.Context, UserCredentials) (ImplResponse, error)
	Logout(context.Context) (ImplResponse, error)
	ResetPassword(context.Context, UserEmail) (ImplResponse, error)
	ListUsers(context.Context) (ImplResponse, error)
	CreateUser(context.Context, User) (ImplResponse, error)
	ReadUser(context.Context, string) (ImplResponse, error)
	UpdateUser(context.Context, string, User) (ImplResponse, error)
	DeleteUser(context.Context, string) (ImplResponse, error)
	SetUserPassword(context.Context, string, Password) (ImplResponse, error)
	ResetUserPassword(context.Context, string) (ImplResponse, error)
	SetUserApiToken(context.Context, string, bool, string) (ImplResponse, error)
}


// VnetAPIServicer defines the api actions for the VnetAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type VnetAPIServicer interface { 
	ListVNets(context.Context) (ImplResponse, error)
	ReadVNet(context.Context, string) (ImplResponse, error)
	UpdateVNet(context.Context, string, VNet) (ImplResponse, error)
	DeleteVNet(context.Context, string) (ImplResponse, error)
	CreateSubnet(context.Context, string, Subnet) (ImplResponse, error)
	SetVNetDefaultSubnet(context.Context, string, string) (ImplResponse, error)
	ListVNetSubnets(context.Context, string) (ImplResponse, error)
}


// VolumeAPIServicer defines the api actions for the VolumeAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type VolumeAPIServicer interface { 
	ListVolumes(context.Context) (ImplResponse, error)
	ReadVolume(context.Context, string) (ImplResponse, error)
	UpdateVolume(context.Context, string, Volume) (ImplResponse, error)
	DeleteVolume(context.Context, string) (ImplResponse, error)
}


// ZoneAPIServicer defines the api actions for the ZoneAPI service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type ZoneAPIServicer interface { 
	ListZones(context.Context) (ImplResponse, error)
	ReadZone(context.Context, string) (ImplResponse, error)
	UpdateZone(context.Context, string, Zone) (ImplResponse, error)
	DeleteZone(context.Context, string) (ImplResponse, error)
	CreateKaktus(context.Context, string, Kaktus) (ImplResponse, error)
	ListZoneKaktuses(context.Context, string) (ImplResponse, error)
}
