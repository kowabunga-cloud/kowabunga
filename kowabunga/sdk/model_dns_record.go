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




// DnsRecord - A DNS record.
type DnsRecord struct {

	// The DNS record ID (auto-generated).
	Id string `json:"id,omitempty"`

	// The DNS record name.
	Name string `json:"name"`

	// The DNS record description.
	Description string `json:"description,omitempty"`

	// The DNS record associated domain (inherited from associated project).
	Domain string `json:"domain,omitempty"`

	// A list of IPv4 addresses to be associated to the record.
	Addresses []string `json:"addresses"`
}

// AssertDnsRecordRequired checks if the required fields are not zero-ed
func AssertDnsRecordRequired(obj DnsRecord) error {
	elements := map[string]interface{}{
		"name": obj.Name,
		"addresses": obj.Addresses,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	return nil
}

// AssertDnsRecordConstraints checks if the values respects the defined constraints
func AssertDnsRecordConstraints(obj DnsRecord) error {
	return nil
}
