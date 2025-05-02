/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

type KowabungaVRRPType uint8
type KowabungaVRRPAuthType uint8

const (
	KowabungaVRRPAdvertisement KowabungaVRRPType = 0x01 // router advertisement
)

// String conversions for VRRP message types
func (v KowabungaVRRPType) String() string {
	switch v {
	case KowabungaVRRPAdvertisement:
		return "KowabungaVRRP Advertisement"
	default:
		return ""
	}
}

const (
	KowabungaVRRPVersion        uint8                 = 15
	KowabungaVRRPAuthNoAuth     KowabungaVRRPAuthType = 0x00 // No Authentication
	KowabungaVRRPAuthReserved1  KowabungaVRRPAuthType = 0x01 // Reserved field 1
	KowabungaVRRPAuthReserved2  KowabungaVRRPAuthType = 0x02 // Reserved field 2
	KowabungaVRRPTTL            uint8                 = 255
	KowabungaVRRPIPv4BaseLength uint16                = 8
	VRRPReservedProtocolId      uint16                = 112
)

func (v KowabungaVRRPAuthType) String() string {

	switch v {
	case KowabungaVRRPAuthNoAuth:
		return "No Authentication"
	case KowabungaVRRPAuthReserved1:
		return "Reserved"
	case KowabungaVRRPAuthReserved2:
		return "Reserved"
	default:
		return ""
	}
}

// Pseudo header required for checksum computation. see RFC2460
type PseudoHeader struct {
	SourceAddress      net.IP
	DestinationAddress net.IP
	len                uint16
	zero               uint8
	nxtHeaderProtocol  uint8
}

func (header PseudoHeader) Serialize() []byte {
	var bytes = make([]byte, 36)
	copy(bytes, header.SourceAddress)
	copy(bytes[16:], header.DestinationAddress)
	copy(bytes[32:], []byte{header.zero, header.nxtHeaderProtocol, byte(header.len >> 8), byte(header.len)})
	return bytes
}

// KowabungaVRRP represents an VRRP v15 message.
// Derived from VRRP3 impl. IPV4 only
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                    IPv4 Fields								                 |
// ...                                                             ...
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |Version| Type  | Virtual Rtr ID|   Priority    |Count IPvX Addr|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |(rsvd) |     Max Adver Int     |          Checksum             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// +                                                               +
// |                       IPvX Address(es)                        |
// +                                                               +
// +                                                               +
// +                                                               +
// +                                                               +
// |                                                               |
// +                                                               +
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type KowabungaVRRP struct {
	pseudoHeader PseudoHeader      // Pseudo header data to build the checksum
	Version      uint8             // The version field specifies the VRRP protocol version of this packet (v255 KowabungaCustom)
	Type         KowabungaVRRPType // The type field specifies the type of this VRRP packet.  The only type defined is ADVERTISEMENT
	VirtualRtrID uint8             // identifies the virtual router this packet is reporting status for
	Priority     uint8             // specifies the sending VRRP router's priority for the virtual router (100 = default)
	CountIPAddr  uint8             // The number of IP addresses contained in this VRRP advertisement.
	AdverInt     uint16            // The Advertisement interval indicates the time interval (in centiseconds) between ADVERTISEMENTS.  The default must be 100 centiseconds (1 second).
	Checksum     uint16            // used to detect data corruption in the VRRP message.
	IPAddresses  []net.IP          // one or more IP addresses associated with the virtual router. Specified in the CountIPAddr field.
}

func (v *KowabungaVRRP) configurePseudoHeader(source, dest *net.IP) {
	v.pseudoHeader.SourceAddress = *source
	v.pseudoHeader.DestinationAddress = *dest
	v.pseudoHeader.len = v.length()
	v.pseudoHeader.nxtHeaderProtocol = uint8(VRRPReservedProtocolId)
}

func (v *KowabungaVRRP) DecodeFromBytes(data []byte) error {

	v.Version = data[0] >> 4
	v.Type = KowabungaVRRPType(data[0] << 4 >> 4)
	v.VirtualRtrID = data[1]
	v.Priority = data[2]
	v.CountIPAddr = data[3]
	v.AdverInt = binary.BigEndian.Uint16(data[4:6])
	v.Checksum = binary.BigEndian.Uint16(data[6:8])

	if v.Type != 1 {
		// rfc3768: A packet with unknown type MUST be discarded.
		return fmt.Errorf("unrecognized KowabungaVRRP type field.: %d", v.Type)
	}
	if v.CountIPAddr < 1 {
		return errors.New("KowabungaVRRP number of IP addresses is not valid")
	}
	// populate the IPAddress field. The number of addresses is specified in the v.CountIPAddr field
	// offset references the starting byte containing the list of ip addresses
	offset := KowabungaVRRPIPv4BaseLength
	for i := uint8(0); i < v.CountIPAddr; i++ {
		v.IPAddresses = append(v.IPAddresses, data[offset:offset+4])
		offset += 4
	}
	return nil
}

func (v *KowabungaVRRP) length() uint16 {
	return KowabungaVRRPIPv4BaseLength + uint16(v.CountIPAddr)*4
}

// computeChecksum=true to send packet
// computeChecksum=false when receive
func (v *KowabungaVRRP) Serialize(source, dest *net.IP, computeChecksum bool) ([]byte, error) {
	if int(v.CountIPAddr) != len(v.IPAddresses) {
		return nil, fmt.Errorf("packet CountIPAddr must match IPAddresses count")
	}
	if source == nil || dest == nil {
		return nil, fmt.Errorf("source and destination IP can't be nil")
	}
	//Pre configure pseudo header for checksum usage
	v.configurePseudoHeader(source, dest)
	// IPV4
	bytes := make([]byte, v.length())
	bytes[0] = v.Version<<4 + uint8(v.Type)
	bytes[1] = v.VirtualRtrID
	bytes[2] = v.Priority
	bytes[3] = v.CountIPAddr
	binary.BigEndian.PutUint16(bytes[4:6], v.AdverInt)
	ipsStartingBytesIndex := KowabungaVRRPIPv4BaseLength
	for _, ip := range v.IPAddresses {
		copy(bytes[ipsStartingBytesIndex:ipsStartingBytesIndex+4], ip.To4())
		ipsStartingBytesIndex += 4
	}

	if computeChecksum {
		binary.BigEndian.PutUint16(bytes[6:8], v.computeChecksum(bytes))
	} else {
		binary.BigEndian.PutUint16(bytes[6:8], v.Checksum)
	}
	return bytes, nil
}

func (v *KowabungaVRRP) computeChecksum(vrrpBytes []byte) uint16 {

	// Make sure checksum bytes are cleared
	vrrpBytes[6] = 0
	vrrpBytes[7] = 0

	bytes := v.pseudoHeader.Serialize()
	// Checksum is computed based on pseudoheader appended with
	// vrrp packet bytes
	bytes = append(bytes, vrrpBytes...)

	// Compute checksum
	var csum uint32
	for i := 0; i < len(bytes); i += 2 {
		csum += uint32(bytes[i]) << 8
		csum += uint32(bytes[i+1])
	}
	for csum > 65535 {
		// Add carry to the sum
		csum = (csum >> 16) + uint32(uint16(csum))
	}
	// Flip all the bits
	return ^uint16(csum)
}

func (v *KowabungaVRRP) ValidateChecksum(source, dest *net.IP) (bool, uint16, error) {
	// Checksum is computed based on pseudoheader appended with
	// vrrp packet bytes
	vrrpBytes, err := v.Serialize(source, dest, false)
	if err != nil {
		return false, 0, err
	}
	bytes := v.pseudoHeader.Serialize()
	bytes = append(bytes, vrrpBytes...)

	// Compute checksum
	var csum uint32
	for i := 0; i < len(bytes); i += 2 {
		csum += uint32(bytes[i]) << 8
		csum += uint32(bytes[i+1])
	}
	for csum > 65535 {
		// Add carry to the sum
		csum = (csum >> 16) + uint32(uint16(csum))
	}
	if uint16(csum) == 0xFFFF {
		return true, uint16(csum), nil
	}
	return false, uint16(csum), nil
}
