/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	PrometheusNamespace             = "kowabunga"
	PrometheusScrapeIntervalSeconds = 1800
)

type KowabungaExporter struct {
	scheduler *time.Ticker
	mutex     sync.RWMutex

	up           prometheus.Gauge
	totalScrapes prometheus.Counter

	Metrics map[string]*prometheus.GaugeVec
}

func (e *KowabungaExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()

	for _, m := range e.Metrics {
		m.Describe(ch)
	}
}

func (e *KowabungaExporter) Collect(ch chan<- prometheus.Metric) {
	// Protect metrics from concurrent collects.
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	ch <- e.up
	ch <- e.totalScrapes

	for _, m := range e.Metrics {
		m.Collect(ch)
	}
}

func (e *KowabungaExporter) Schedule() {
	for t := range e.scheduler.C {
		e.Scrape(t)
	}
}

func newGaugeVecMetric(metricName string, docString string, labels ...string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   PrometheusNamespace,
			Name:        metricName,
			Help:        docString,
			ConstLabels: nil,
		},
		append([]string{}, labels...),
	)
}

func newMetric(metricName string, docString string, labels ...string) *prometheus.GaugeVec {
	return newGaugeVecMetric(metricName, docString, labels...)
}

const (
	MetricRegions          = "regions"
	MetricZones            = "zones"
	MetricKaktusCPU        = "kaktus_allocated_vcpus"
	MetricKaktusMem        = "kaktus_allocated_memory_bytes"
	MetricKaktusInstances  = "kaktus_instances"
	MetricKaktusCost       = "kaktus_cost"
	MetricPoolsAllocated   = "storage_pools_allocated_bytes"
	MetricPoolsFree        = "storage_pools_free_bytes"
	MetricPoolsCapacity    = "storage_pools_capacity_bytes"
	MetricPoolsCost        = "storage_pools_cost"
	MetricVNets            = "virtual_networks"
	MetricPrivateSubnets   = "private_subnets"
	MetricPublicSubnets    = "public_subnets_ips"
	MetricProjects         = "projects"
	MetricProjectInstances = "project_instances"
	MetricProjectCPU       = "project_allocated_vcpus"
	MetricProjectMem       = "project_allocated_memory_bytes"
	MetricProjectVolumes   = "project_volumes"
	MetricProjectStorage   = "project_allocated_storage_bytes"
	MetricProjectCost      = "project_estimated_cost"
	MetricInstanceSettings = "instance_settings"
	MetricInstanceCPU      = "instance_allocated_vcpus"
	MetricInstanceMem      = "instance_allocated_memory_bytes"
	MetricInstanceCost     = "instance_estimated_cost"
)

func (e *KowabungaExporter) SetupMetrics() {
	e.up = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Name:      "up",
		Help:      "Was the last scrape of Kowabunga metrics successful.",
	})

	e.totalScrapes = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: PrometheusNamespace,
		Name:      "total_scrapes",
		Help:      "Current total Kowabunga metrics scrapes.",
	})

	e.Metrics = map[string]*prometheus.GaugeVec{
		MetricRegions:         newMetric(MetricRegions, "Kowabunga regions", "name"),
		MetricZones:           newMetric(MetricZones, "Kowabunga zones", "name"),
		MetricKaktusCPU:       newMetric(MetricKaktusCPU, "Kowabunga kaktus allocated vCPUs count", "zone", "name"),
		MetricKaktusMem:       newMetric(MetricKaktusMem, "Kowabunga kaktus allocated memory size (bytes)", "zone", "name"),
		MetricKaktusInstances: newMetric(MetricKaktusInstances, "Kowabunga kaktus allocated instances count", "zone", "name"),
		MetricKaktusCost:      newMetric(MetricKaktusCost, "Kowabunga kaktus cost", "zone", "name", "currency"),
		MetricPoolsAllocated:  newMetric(MetricPoolsAllocated, "Kowabunga storage pools allocated size (bytes)", "region"),
		MetricPoolsFree:       newMetric(MetricPoolsFree, "Kowabunga storage pools free size (bytes)", "region"),
		MetricPoolsCapacity:   newMetric(MetricPoolsCapacity, "Kowabunga storage pools capacity (bytes)", "region"),
		MetricPoolsCost:       newMetric(MetricPoolsCost, "Kowabunga storage pools cost", "region", "currency"),
		MetricVNets:           newMetric(MetricVNets, "Kowabunga virtual networks", "region", "type"),
		MetricPrivateSubnets:  newMetric(MetricPrivateSubnets, "Kowabunga private subnets", "region", "netmask", "type"),
		MetricPublicSubnets:   newMetric(MetricPublicSubnets, "Kowabunga public subnets IP addresses", "region", "netmask", "type"),

		MetricProjects:         newMetric(MetricProjects, "Kowabunga projects", "name"),
		MetricProjectInstances: newMetric(MetricProjectInstances, "Kowabunga project allocated instances count", "name"),
		MetricProjectCPU:       newMetric(MetricProjectCPU, "Kowabunga project allocated vCPUs count", "name"),
		MetricProjectMem:       newMetric(MetricProjectMem, "Kowabunga project allocated memory size (bytes)", "name"),
		MetricProjectVolumes:   newMetric(MetricProjectVolumes, "Kowabunga project allocated volumes count", "name"),
		MetricProjectStorage:   newMetric(MetricProjectStorage, "Kowabunga project allocated storage size (bytes)", "name"),
		MetricProjectCost:      newMetric(MetricProjectCost, "Kowabunga project estimated cost", "name", "currency"),
		MetricInstanceSettings: newMetric(MetricInstanceSettings, "Kowabunga instance settings", "name", "project", "os", "ip"),
		MetricInstanceCPU:      newMetric(MetricInstanceCPU, "Kowabunga instance allocated vCPUs count", "name", "project"),
		MetricInstanceMem:      newMetric(MetricInstanceMem, "Kowabunga instance allocated memory size (bytes)", "name", "project"),
		MetricInstanceCost:     newMetric(MetricInstanceCost, "Kowabunga instance estimated cost", "name", "project", "currency"),
	}
}

type KowabungaZoneKaktus struct {
	VCPUs     float64
	Memory    float64
	Instances float64
	Cost      float64
	Currency  string
}

type KowabungaRegionPools struct {
	Capacity  float64
	Allocated float64
	Free      float64
	Cost      float64
	Currency  string
}

type KowabungaRegionPrivateSubnets struct {
	Allocated float64
	Free      float64
}

type KowabungaRegionPublicSubnets struct {
	Reserved  float64
	Allocated float64
	Free      float64
}

type KowabungaRegionMetrics struct {
	Pools          map[string]*KowabungaRegionPools
	PrivateVNets   float64
	PublicVNets    float64
	PrivateSubnets map[int]*KowabungaRegionPrivateSubnets
	PublicSubnets  map[int]*KowabungaRegionPublicSubnets
}

type KowabungaZoneMetrics struct {
	Kaktuses map[string]*KowabungaZoneKaktus
}

type KowabungaProjectMetrics struct {
	Instances float64
	VCPUs     float64
	Memory    float64
	Volumes   float64
	Storage   float64
	Cost      float64
	Currency  string
}

type KowabungaInstanceMetrics struct {
	Project  string
	OS       string
	VCPUs    float64
	Memory   float64
	Cost     float64
	Currency string
	IP       string
}

type KowabungaMetrics struct {
	Regions   map[string]KowabungaRegionMetrics
	Zones     map[string]KowabungaZoneMetrics
	Projects  map[string]KowabungaProjectMetrics
	Instances map[string]KowabungaInstanceMetrics
}

func (m *KowabungaMetrics) Log() {
	klog.Infof("Kowabunga System Digest")
	klog.Infof("Global Statistics:")
	klog.Infof(" - Regions: %d", len(m.Regions))
	for key, r := range m.Regions {
		klog.Infof("  + Name: %s", key)
		klog.Infof("    Public Virtual Networks: %.0f", r.PublicVNets)
		klog.Infof("    Private Virtual Networks: %.0f", r.PrivateVNets)
		for mask, s := range r.PrivateSubnets {
			total := s.Free + s.Allocated
			ratio := s.Allocated * 100 / total
			klog.Infof("    Private /%d Subnets: %.0f/%.0f allocated (%.2f%%)", mask, s.Allocated, total, ratio)
		}
		for mask, s := range r.PublicSubnets {
			used := s.Reserved + s.Allocated
			total := s.Free + used
			ratio := used * 100 / total
			klog.Infof("    Public /%d Subnets: %.0f/%.0f reserved/allocated IPs (%.2f%%)", mask, used, total, ratio)
		}
		for t, p := range r.Pools {
			total := p.Free + p.Allocated
			ratio := p.Allocated * 100 / total
			klog.Infof("    Storage Pools '%s': %s/%s allocated (%.2f%%), cost: %.0f %s", t, byteCountIEC(uint64(p.Allocated)), byteCountIEC(uint64(total)), ratio, p.Cost, p.Currency)
		}
	}
	klog.Infof(" - Zones: %d", len(m.Zones))
	for key, z := range m.Zones {
		klog.Infof("  + Name: %s", key)
		for name, h := range z.Kaktuses {
			klog.Infof("    Kaktus '%s': %.0f vCPUs, %s memory, %.0f instances, cost: %.0f %s", name, h.VCPUs, byteCountIEC(uint64(h.Memory)), h.Instances, h.Cost, h.Currency)
		}
	}
	klog.Infof(" - Projects: %d", len(m.Projects))
	for key, p := range m.Projects {
		klog.Infof("  + Name: %s", key)
		klog.Infof("    Instances: %.0f", p.Instances)
		klog.Infof("    Allocated vCPUs: %.0f", p.VCPUs)
		klog.Infof("    Allocated Memory: %s", byteCountIEC(uint64(p.Memory)))
		klog.Infof("    Volumes: %.0f", p.Volumes)
		klog.Infof("    Allocatable SSD Storage: %s", byteCountIEC(uint64(p.Storage)))
		klog.Infof("    Estimated Monthly Cost: %.2f %s", p.Cost, p.Currency)
	}
}

func NewKowabungaMetrics() *KowabungaMetrics {
	klog.Infof("Collecting Kowabunga statistics and metrics ...")

	m := KowabungaMetrics{
		Regions:   map[string]KowabungaRegionMetrics{},
		Zones:     map[string]KowabungaZoneMetrics{},
		Projects:  map[string]KowabungaProjectMetrics{},
		Instances: map[string]KowabungaInstanceMetrics{},
	}

	regions := FindRegions()
	for _, r := range regions {
		rm := KowabungaRegionMetrics{
			Pools:          map[string]*KowabungaRegionPools{},
			PrivateSubnets: map[int]*KowabungaRegionPrivateSubnets{},
			PublicSubnets:  map[int]*KowabungaRegionPublicSubnets{},
		}

		for _, id := range r.VNets() {
			v, err := r.VNet(id)
			if err != nil {
				continue
			}
			if v.Private {
				rm.PrivateVNets += 1
				for _, sid := range v.Subnets() {
					s, err := v.Subnet(sid)
					if err != nil {
						continue
					}

					_, ipnet, err := net.ParseCIDR(s.CIDR)
					if err != nil {
						continue
					}
					mask, _ := ipnet.Mask.Size()

					_, exists := rm.PrivateSubnets[mask]
					if !exists {
						rm.PrivateSubnets[mask] = &KowabungaRegionPrivateSubnets{}
					}

					if s.ProjectID != "" {
						rm.PrivateSubnets[mask].Allocated += 1
					} else {
						rm.PrivateSubnets[mask].Free += 1
					}
				}
			} else {
				rm.PublicVNets += 1
				for _, sid := range v.Subnets() {
					s, err := v.Subnet(sid)
					if err != nil {
						continue
					}

					_, ipnet, err := net.ParseCIDR(s.CIDR)
					if err != nil {
						continue
					}
					mask, _ := ipnet.Mask.Size()

					_, exists := rm.PublicSubnets[mask]
					if !exists {
						rm.PublicSubnets[mask] = &KowabungaRegionPublicSubnets{}
					}

					var reserved uint64 = 0
					for _, r := range s.Reserved {
						first := net.ParseIP(r.First)
						last := net.ParseIP(r.Last)
						loop_over := true
						ip := first
						for loop_over {
							if ip.Equal(last) {
								loop_over = false
							}
							reserved += 1
							ip = cidr.Inc(ip)
						}
					}
					rm.PublicSubnets[mask].Reserved += float64(reserved)

					var allocated uint64 = 0
					for _, aid := range s.Adapters() {
						a, err := s.Adapter(aid)
						if err != nil {
							continue
						}
						allocated += uint64(len(a.Addresses))
					}
					rm.PublicSubnets[mask].Allocated += float64(allocated)

					cidr_size := cidr.AddressCount(ipnet)
					free := cidr_size - reserved - allocated
					rm.PublicSubnets[mask].Free += float64(free)
				}
			}
		}

		for _, id := range r.StoragePools() {
			p, err := FindStoragePoolByID(id)
			if err != nil {
				continue
			}

			_, exists := rm.Pools[p.Name]
			if !exists {
				rm.Pools[p.Name] = &KowabungaRegionPools{}
			}

			rm.Pools[p.Name].Capacity += float64(p.Capacity)
			rm.Pools[p.Name].Allocated += float64(p.Allocation)
			rm.Pools[p.Name].Free += float64(p.Available)
			rm.Pools[p.Name].Cost += float64(p.Cost.Price)
			rm.Pools[p.Name].Currency = p.Cost.Currency
		}

		m.Regions[r.Name] = rm
	}

	zones := FindZones()
	for _, z := range zones {
		zm := KowabungaZoneMetrics{
			Kaktuses: map[string]*KowabungaZoneKaktus{},
		}

		for _, id := range z.Kaktuses() {
			h, err := z.Kaktus(id)
			if err != nil {
				continue
			}

			zm.Kaktuses[h.Name] = &KowabungaZoneKaktus{
				VCPUs:     float64(h.Usage.VCPUs),
				Memory:    float64(h.Usage.MemorySize),
				Instances: float64(h.Usage.InstancesCount),
				Cost:      float64(h.Costs.CPU.Price + h.Costs.Memory.Price),
				Currency:  h.Costs.CPU.Currency,
			}
		}

		m.Zones[z.Name] = zm
	}

	projects := FindProjects()
	for _, p := range projects {
		_ = p.GetCost()
		pm := KowabungaProjectMetrics{
			VCPUs:     float64(p.Usage.VCPUs),
			Memory:    float64(p.Usage.MemorySize),
			Storage:   float64(p.Usage.StorageSize),
			Instances: float64(len(p.Instances())),
			Volumes:   float64(len(p.Volumes())),
			Cost:      float64(p.Cost.Price),
			Currency:  p.Cost.Currency,
		}

		m.Projects[p.Name] = pm
	}

	instances := FindInstances()
	for _, i := range instances {
		p, err := i.Project()
		if err != nil {
			continue
		}
		im := KowabungaInstanceMetrics{
			Project:  p.Name,
			OS:       i.OS,
			VCPUs:    float64(i.CPU),
			Memory:   float64(i.Memory),
			Cost:     float64(i.Cost.Price),
			Currency: i.Cost.Currency,
			IP:       i.LocalIP,
		}

		m.Instances[i.Name] = im
	}

	return &m
}

func (e *KowabungaExporter) SetMetrics(m *KowabungaMetrics) {
	for key, r := range m.Regions {
		e.Metrics[MetricRegions].WithLabelValues(key).Set(1)

		e.Metrics[MetricVNets].WithLabelValues(key, "private").Set(r.PrivateVNets)
		e.Metrics[MetricVNets].WithLabelValues(key, "public").Set(r.PublicVNets)

		for m, s := range r.PrivateSubnets {
			mask := fmt.Sprintf("%d", m)
			e.Metrics[MetricPrivateSubnets].WithLabelValues(key, mask, "allocated").Set(s.Allocated)
			e.Metrics[MetricPrivateSubnets].WithLabelValues(key, mask, "free").Set(s.Free)
		}

		for m, s := range r.PublicSubnets {
			mask := fmt.Sprintf("%d", m)
			e.Metrics[MetricPublicSubnets].WithLabelValues(key, mask, "reserved").Set(s.Reserved)
			e.Metrics[MetricPublicSubnets].WithLabelValues(key, mask, "allocated").Set(s.Allocated)
			e.Metrics[MetricPublicSubnets].WithLabelValues(key, mask, "free").Set(s.Free)
		}

		for _, p := range r.Pools {
			e.Metrics[MetricPoolsAllocated].WithLabelValues(key).Set(p.Allocated)
			e.Metrics[MetricPoolsFree].WithLabelValues(key).Set(p.Free)
			e.Metrics[MetricPoolsCapacity].WithLabelValues(key).Set(p.Capacity)
			e.Metrics[MetricPoolsCost].WithLabelValues(key, p.Currency).Set(p.Cost)
		}
	}

	for key, z := range m.Zones {
		e.Metrics[MetricZones].WithLabelValues(key).Set(1)
		for name, h := range z.Kaktuses {
			e.Metrics[MetricKaktusCPU].WithLabelValues(key, name).Set(h.VCPUs)
			e.Metrics[MetricKaktusMem].WithLabelValues(key, name).Set(h.Memory)
			e.Metrics[MetricKaktusInstances].WithLabelValues(key, name).Set(h.Instances)
			e.Metrics[MetricKaktusCost].WithLabelValues(key, name, h.Currency).Set(h.Cost)
		}
	}

	for key, p := range m.Projects {
		e.Metrics[MetricProjects].WithLabelValues(key).Set(1)
		e.Metrics[MetricProjectInstances].WithLabelValues(key).Set(p.Instances)
		e.Metrics[MetricProjectCPU].WithLabelValues(key).Set(p.VCPUs)
		e.Metrics[MetricProjectMem].WithLabelValues(key).Set(p.Memory)
		e.Metrics[MetricProjectVolumes].WithLabelValues(key).Set(p.Volumes)
		e.Metrics[MetricProjectStorage].WithLabelValues(key).Set(p.Storage)
		e.Metrics[MetricProjectCost].WithLabelValues(key, p.Currency).Set(p.Cost)
	}

	for key, i := range m.Instances {
		e.Metrics[MetricInstanceSettings].WithLabelValues(key, i.Project, i.OS, i.IP).Set(1)
		e.Metrics[MetricInstanceCPU].WithLabelValues(key, i.Project).Set(i.VCPUs)
		e.Metrics[MetricInstanceMem].WithLabelValues(key, i.Project).Set(i.Memory)
		e.Metrics[MetricInstanceCost].WithLabelValues(key, i.Project, i.Currency).Set(i.Cost)
	}
}

func (e *KowabungaExporter) Scrape(t time.Time) {
	// get all system stats and metrics
	metrics := NewKowabungaMetrics()
	//metrics.Log()

	// Protect metrics from concurrent collects.
	e.mutex.Lock()
	e.SetMetrics(metrics)
	e.mutex.Unlock()

	// One more scrape
	e.totalScrapes.Inc()
	e.up.Set(1)
}

func NewExporter() *KowabungaExporter {
	e := KowabungaExporter{
		scheduler: time.NewTicker(time.Second * time.Duration(PrometheusScrapeIntervalSeconds)),
	}
	e.SetupMetrics()
	klog.Infof("Registered Prometheus exporter ...")

	go e.Schedule()
	go e.Scrape(time.Now()) // initial scrape

	return &e
}

func (e *KowabungaExporter) HttpHandler() http.Handler {
	// Use our own registry and not the default one,
	// because we don't want all the go stats
	registry := prometheus.NewRegistry()
	registry.MustRegister(e)

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
