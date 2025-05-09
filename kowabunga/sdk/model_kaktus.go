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




// Kaktus - A Kaktus (Kowabunga Affordable KVM and Tight Underneath Storage) is an hyper-converged infrastructure (HCI) bare-metal node offering computing and distributed storage capabilites.
type Kaktus struct {

	// The Kaktus computing node ID (auto-generated).
	Id string `json:"id,omitempty"`

	// The Kaktus computing node name.
	Name string `json:"name"`

	// The Kaktus computing node description.
	Description string `json:"description,omitempty"`

	CpuCost Cost `json:"cpu_cost,omitempty"`

	MemoryCost Cost `json:"memory_cost,omitempty"`

	// The Kaktus node CPU resource over-commit ratio. Overcommitting CPU resources for VMs means allocating more virtual CPUs (vCPUs) to the virtual machines (VMs) than the physical cores available on the node. This can help optimize the utilization of the node CPU and increase the density of VMs per node.
	OvercommitCpuRatio int64 `json:"overcommit_cpu_ratio,omitempty"`

	// The Kaktus node memory resource over-commit ratio. Memory overcommitment is a concept in computing that covers the assignment of more memory to virtual computing devices (or processes) than the physical machine they are hosted, or running on, actually has.
	OvercommitMemoryRatio int64 `json:"overcommit_memory_ratio,omitempty"`

	// a list of existing remote agents managing the Kaktus node.
	Agents []string `json:"agents"`
}

// AssertKaktusRequired checks if the required fields are not zero-ed
func AssertKaktusRequired(obj Kaktus) error {
	elements := map[string]interface{}{
		"name": obj.Name,
		"agents": obj.Agents,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	if err := AssertCostRequired(obj.CpuCost); err != nil {
		return err
	}
	if err := AssertCostRequired(obj.MemoryCost); err != nil {
		return err
	}
	return nil
}

// AssertKaktusConstraints checks if the values respects the defined constraints
func AssertKaktusConstraints(obj Kaktus) error {
	if err := AssertCostConstraints(obj.CpuCost); err != nil {
		return err
	}
	if err := AssertCostConstraints(obj.MemoryCost); err != nil {
		return err
	}
	return nil
}
