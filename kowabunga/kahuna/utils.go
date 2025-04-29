/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"

	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/inhies/go-bytesize"
)

const (
	// https://tools.ietf.org/html/rfc952
	HostnameRegexStringRFC952 = `^[a-zA-Z][a-zA-Z0-9\-\.]+[a-zA-Z0-9]$`
	// accepts hostname starting with a digit https://tools.ietf.org/html/rfc1123
	HostnameRegexStringRFC1123 = `^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*?$`

	PortErrorExpression   = "Invalid port list expression"
	PortErrorInvalid      = "One of the specified ports is"
	PortErrorRangeSize    = "Port range has more than 2 elements"
	PortErrorRangeInvalid = "One of the specified port range has last port > first port"

	MinPort = 1
	MaxPort = 65535
)

func SetDefaultStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func SetFieldStr(field *string, value string) {
	if value != "" {
		*field = value
	}
}

func XmlMarshal(b interface{}) (string, error) {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("  ", "    ")
	err := enc.Encode(b)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func XmlUnmarshal(input string, v any) error {
	return xml.Unmarshal([]byte(input), v)
}

func HumanByteSize(n uint64) string {
	b := bytesize.New(float64(n))
	return b.String()
}

func byteCountIEC(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func VerifyEmail(email string) error {
	verifier := emailverifier.NewVerifier()
	ret, err := verifier.Verify(email)
	if err != nil {
		return err
	}
	if !ret.Syntax.Valid {
		return fmt.Errorf("email address %s syntax is invalid", email)
	}

	return nil
}

func VerifyDomain(name string) bool {
	r := regexp.MustCompile(`^(?i)[a-z0-9-]+(\.[a-z0-9-]+)+\.?$`)
	return r.MatchString(name)
}

func VerifyHostname(name string) bool {
	r := regexp.MustCompile(HostnameRegexStringRFC1123)
	return r.MatchString(name)
}

func HasChildRef(children *[]string, childId string) bool {
	return slices.Contains(*children, childId)
}

func FindChildByID[T any](children *[]string, childId, collection, msg string) (*T, error) {
	var res T
	child, err := FindResourceByID[T](collection, childId)
	if err != nil {
		return &res, err
	}

	if !HasChildRef(children, childId) {
		return &res, fmt.Errorf("%s", msg)
	}

	return child, nil
}

func AddChildRef(children *[]string, childId string) {
	if !HasChildRef(children, childId) {
		*children = append(*children, childId)
	}
}

func RemoveChildRef(children *[]string, childId string) {
	for idx, id := range *children {
		if id == childId {
			*children = append((*children)[:idx], (*children)[idx+1:]...)
			break
		}
	}
}

func HasChildRefs(children ...[]string) bool {
	for _, c := range children {
		if len(c) > 0 {
			return true
		}
	}
	return false
}

func bytesToGB(val int64) int64 {
	return val / 1024 / 1024 / 1024
}

func FindNextBitIp(ipnet net.IPNet, bit int) (net.IP, error) {
	netcidr_first_ip := ipnet.IP
	for i := 0; i < bit; i++ {
		netcidr_first_ip = cidr.Inc(netcidr_first_ip)
	}
	if ipnet.Contains(netcidr_first_ip) {
		return netcidr_first_ip, nil
	}
	return netcidr_first_ip, fmt.Errorf("IP incrementation is outside of subnet %s", ipnet.String())
}

func isValidPort(expr, p string) (int, error) {
	port, err := strconv.Atoi(p)
	if err != nil {
		return -1, err
	}
	if port < MinPort {
		return port, fmt.Errorf("%s: '%s'. %s < %d", PortErrorExpression, expr, PortErrorInvalid, MinPort)
	}
	if port > MaxPort {
		return port, fmt.Errorf("%s: '%s'. %s > %d", PortErrorExpression, expr, PortErrorInvalid, MaxPort)
	}
	return port, nil
}

func IsValidPortListExpression(expr string) error {
	items := strings.Split(expr, ",")
	for _, i := range items {
		rg := strings.Split(i, "-")
		if len(rg) > 2 {
			return fmt.Errorf("%s: '%s'. %s", PortErrorExpression, expr, PortErrorRangeSize)
		}
		first, err := isValidPort(expr, rg[0])
		if err != nil {
			return err
		}
		if len(rg) == 2 {
			last, err := isValidPort(expr, rg[1])
			if err != nil {
				return err
			}
			if last < first {
				return fmt.Errorf("%s: '%s'. %s", PortErrorExpression, expr, PortErrorRangeInvalid)
			}
		}
	}

	return nil
}
