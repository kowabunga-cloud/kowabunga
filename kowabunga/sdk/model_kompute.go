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




// Kompute - A Kompute is a wrapper object for bare virtual machines. It consists of an instance, one to several attached volumes and 2 network adapters (a private one, a public one). This is the prefered way for creating virtual machines. IP addresses will be automatically assigned.
type Kompute struct {

	// The Kompute ID (auto-generated).
	Id string `json:"id,omitempty"`

	// The Kompute name.
	Name string `json:"name"`

	// The Kompute description.
	Description string `json:"description,omitempty"`

	// The Kompute memory size (in bytes).
	Memory int64 `json:"memory"`

	// The Kompute number of vCPUs.
	Vcpus int64 `json:"vcpus"`

	// The Kompute OS disk size (in bytes).
	Disk int64 `json:"disk"`

	// The Kompute extra data disk size (in bytes). If unspecified, no extra data disk will be assigned.
	DataDisk int64 `json:"data_disk,omitempty"`

	// The Kompute assigned private IPv4 address (read-only).
	Ip string `json:"ip,omitempty"`
}

// AssertKomputeRequired checks if the required fields are not zero-ed
func AssertKomputeRequired(obj Kompute) error {
	elements := map[string]interface{}{
		"name": obj.Name,
		"memory": obj.Memory,
		"vcpus": obj.Vcpus,
		"disk": obj.Disk,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	return nil
}

// AssertKomputeConstraints checks if the values respects the defined constraints
func AssertKomputeConstraints(obj Kompute) error {
	return nil
}
