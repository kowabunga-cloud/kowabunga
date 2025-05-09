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




// KonveyBackends - A Konvey Backends settings.
type KonveyBackends struct {

	// The Konvey (Kowabunga Network Load-Balancer) endpoint list of load-balanced backend hosts.
	Hosts []string `json:"hosts"`

	// The Konvey (Kowabunga Network Load-Balancer) endpoint backend service port.
	Port int64 `json:"port"`
}

// AssertKonveyBackendsRequired checks if the required fields are not zero-ed
func AssertKonveyBackendsRequired(obj KonveyBackends) error {
	elements := map[string]interface{}{
		"hosts": obj.Hosts,
		"port": obj.Port,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	return nil
}

// AssertKonveyBackendsConstraints checks if the values respects the defined constraints
func AssertKonveyBackendsConstraints(obj KonveyBackends) error {
	return nil
}
