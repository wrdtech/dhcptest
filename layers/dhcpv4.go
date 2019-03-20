// Copyright 2016 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package layers

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/google/gopacket"
	"net"
)

// DHCPOp rerprents a bootp operation
type DHCPOp byte

// bootp operations
const (
	DHCPOpRequest DHCPOp = 1
	DHCPOpReply   DHCPOp = 2
)

// String returns a string version of a DHCPOp.
func (o DHCPOp) String() string {
	switch o {
	case DHCPOpRequest:
		return "Request"
	case DHCPOpReply:
		return "Reply"
	default:
		return "Unknown"
	}
}

// DHCPMsgType represents a DHCP operation
type DHCPMsgType byte

// Constants that represent DHCP operations
const (
	DHCPMsgTypeUnspecified DHCPMsgType = iota
	DHCPMsgTypeDiscover
	DHCPMsgTypeOffer
	DHCPMsgTypeRequest
	DHCPMsgTypeDecline
	DHCPMsgTypeAck
	DHCPMsgTypeNak
	DHCPMsgTypeRelease
	DHCPMsgTypeInform
)

// String returns a string version of a DHCPMsgType.
func (o DHCPMsgType) String() string {
	switch o {
	case DHCPMsgTypeUnspecified:
		return "Unspecified"
	case DHCPMsgTypeDiscover:
		return "Discover"
	case DHCPMsgTypeOffer:
		return "Offer"
	case DHCPMsgTypeRequest:
		return "Request"
	case DHCPMsgTypeDecline:
		return "Decline"
	case DHCPMsgTypeAck:
		return "Ack"
	case DHCPMsgTypeNak:
		return "Nak"
	case DHCPMsgTypeRelease:
		return "Release"
	case DHCPMsgTypeInform:
		return "Inform"
	default:
		return "Unknown"
	}
}

type BootpFlag uint16

const (
	BroadcastFlag BootpFlag = 0x8000
	UnicastFlag BootpFlag = 0x0000
)

func (b BootpFlag) String() string {
	switch b {
	case BroadcastFlag:
		return "Brocast"
	case UnicastFlag:
		return "Unicast"
	default:
		return "Unkown Bootp Flag"
	}
}



//DHCPMagic is the RFC 2131 "magic cooke" for DHCP.
var DHCPMagic uint32 = 0x63825363

// DHCPv4 contains data for a single DHCP packet.
type DHCPv4 struct {
	BaseLayer
	Operation    DHCPOp
	HardwareType LinkType
	HardwareLen  uint8
	HardwareOpts uint8
	Xid          uint32
	Secs         uint16
	Flags        uint16
	ClientIP     net.IP
	YourClientIP net.IP
	NextServerIP net.IP
	RelayAgentIP net.IP
	ClientHWAddr net.HardwareAddr
	ServerName   []byte
	File         []byte
	Options      DHCPOptions
}

// DHCPOptions is used to get nicely printed option lists which would normally
// be cut off after 5 options.
type DHCPOptions []DHCPOption

// String returns a string version of the options list.
func (o DHCPOptions) String() string {
	buf := &bytes.Buffer{}
	buf.WriteByte('[')
	for i, opt := range o {
		buf.WriteString(opt.String())
		if i+1 != len(o) {
			buf.WriteString(", ")
		}
	}
	buf.WriteByte(']')
	return buf.String()
}

// LayerType returns gopacket.LayerTypeDHCPv4
func (d *DHCPv4) LayerType() gopacket.LayerType { return LayerTypeDHCPv4 }

// DecodeFromBytes decodes the given bytes into this layer.
func (d *DHCPv4) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	d.Options = d.Options[:0]
	d.Operation = DHCPOp(data[0])
	d.HardwareType = LinkType(data[1])
	d.HardwareLen = data[2]
	d.HardwareOpts = data[3]
	d.Xid = binary.BigEndian.Uint32(data[4:8])
	d.Secs = binary.BigEndian.Uint16(data[8:10])
	d.Flags = binary.BigEndian.Uint16(data[10:12])
	d.ClientIP = net.IP(data[12:16])
	d.YourClientIP = net.IP(data[16:20])
	d.NextServerIP = net.IP(data[20:24])
	d.RelayAgentIP = net.IP(data[24:28])
	d.ClientHWAddr = net.HardwareAddr(data[28 : 28+d.HardwareLen])
	d.ServerName = data[44:108]
	d.File = data[108:236]
	if binary.BigEndian.Uint32(data[236:240]) != DHCPMagic {
		return InvalidMagicCookie
	}

	if len(data) <= 240 {
		// DHCP Packet could have no option (??)
		return nil
	}

	options := data[240:]

	stop := len(options)
	start := 0
	for start < stop {
		o := DHCPOption{}
		if err := o.decode(options[start:]); err != nil {
			return err
		}
		if o.Type == DHCPOptEnd {
			break
		}
		d.Options = append(d.Options, o)
		// Check if the option is a single byte pad
		if o.Type == DHCPOptPad {
			start++
		} else {
			start += int(o.Length) + 2
		}
	}
	return nil
}

// Len returns the length of a DHCPv4 packet.
func (d *DHCPv4) Len() uint16 {
	n := uint16(240)
	for _, o := range d.Options {
		if o.Type == DHCPOptPad {
			n++
		} else {
			n += uint16(o.Length) + 2
		}

	}
	n++ // for opt end
	return n
}

// SerializeTo writes the serialized form of this layer into the
// SerializationBuffer, implementing gopacket.SerializableLayer.
// See the docs for gopacket.SerializableLayer for more info.
func (d *DHCPv4) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	plen := int(d.Len())

	data, err := b.PrependBytes(plen)
	if err != nil {
		return err
	}

	data[0] = byte(d.Operation)
	data[1] = byte(d.HardwareType)
	if opts.FixLengths {
		d.HardwareLen = uint8(len(d.ClientHWAddr))
	}
	data[2] = d.HardwareLen
	data[3] = d.HardwareOpts
	binary.BigEndian.PutUint32(data[4:8], d.Xid)
	binary.BigEndian.PutUint16(data[8:10], d.Secs)
	binary.BigEndian.PutUint16(data[10:12], d.Flags)
	copy(data[12:16], d.ClientIP.To4())
	copy(data[16:20], d.YourClientIP.To4())
	copy(data[20:24], d.NextServerIP.To4())
	copy(data[24:28], d.RelayAgentIP.To4())
	copy(data[28:44], d.ClientHWAddr)
	copy(data[44:108], d.ServerName)
	copy(data[108:236], d.File)
	binary.BigEndian.PutUint32(data[236:240], DHCPMagic)


	if len(d.Options) > 0 {
		offset := 240
		for _, o := range d.Options {
			if err := o.encode(data[offset:]); err != nil {
				return err
			}
			// A pad option is only a single byte
			if o.Type == DHCPOptPad {
				offset++
			} else {
				offset += 2 + len(o.Data)
			}
		}
		optend := NewDHCPOption(DHCPOptEnd, nil)
		if err := optend.encode(data[offset:]); err != nil {
			return err
		}
	}
	return nil
}

// CanDecode returns the set of layer types that this DecodingLayer can decode.
func (d *DHCPv4) CanDecode() gopacket.LayerClass {
	return LayerTypeDHCPv4
}

// NextLayerType returns the layer type contained by this DecodingLayer.
func (d *DHCPv4) NextLayerType() gopacket.LayerType {
	return gopacket.LayerTypePayload
}

// edited by wsx

func (d *DHCPv4) MessageType() DHCPMsgType {
	for _,option := range d.Options {
		switch option.Type {
		case DHCPOptMessageType:
			if option.Length == 1 {
				return DHCPMsgType(option.Data[0])
			}
		}
	}
	return DHCPMsgTypeUnspecified
}

func (d *DHCPv4) AddOption(optType DHCPOpt, data []byte) {
	d.Options = append(d.Options, NewDHCPOption(optType, data))
}

func (d *DHCPv4) AddParamRequest(dhcpOpts ...DHCPOpt) {
	for _, dhcpOpt := range dhcpOpts {
		for i := range d.Options {
			if d.Options[i].Type == DHCPOptParamsRequest {
				d.Options[i].AddByte(byte(dhcpOpt))
				return
			}
		}
		d.AddOption(DHCPOptParamsRequest, []byte{byte(dhcpOpt)})

	}
}

func (d *DHCPv4) SetBroadcast() {
	d.Flags = uint16(BroadcastFlag)
}

func (d *DHCPv4) SetUnicast() {
	d.Flags = uint16(UnicastFlag)
}
//edited by wsx end

func decodeDHCPv4(data []byte, p gopacket.PacketBuilder) error {
	dhcp := &DHCPv4{}
	err := dhcp.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(dhcp)
	return p.NextDecoder(gopacket.LayerTypePayload)
}

// DHCPOpt represents a DHCP option or parameter from RFC-2132
type DHCPOpt byte

// Constants for the DHCPOpt options.
const (
	DHCPOptPad                   DHCPOpt = 0
	DHCPOptSubnetMask            DHCPOpt = 1   // 4, net.IP
	DHCPOptTimeOffset            DHCPOpt = 2   // 4, int32 (signed seconds from UTC)
	DHCPOptRouter                DHCPOpt = 3   // n*4, [n]net.IP
	DHCPOptTimeServer            DHCPOpt = 4   // n*4, [n]net.IP
	DHCPOptNameServer            DHCPOpt = 5   // n*4, [n]net.IP
	DHCPOptDNS                   DHCPOpt = 6   // n*4, [n]net.IP
	DHCPOptLogServer             DHCPOpt = 7   // n*4, [n]net.IP
	DHCPOptCookieServer          DHCPOpt = 8   // n*4, [n]net.IP
	DHCPOptLPRServer             DHCPOpt = 9   // n*4, [n]net.IP
	DHCPOptImpressServer         DHCPOpt = 10  // n*4, [n]net.IP
	DHCPOptResLocServer          DHCPOpt = 11  // n*4, [n]net.IP
	DHCPOptHostname              DHCPOpt = 12  // n, string
	DHCPOptBootfileSize          DHCPOpt = 13  // 2, uint16
	DHCPOptMeritDumpFile         DHCPOpt = 14  // >1, string
	DHCPOptDomainName            DHCPOpt = 15  // n, string
	DHCPOptSwapServer            DHCPOpt = 16  // n*4, [n]net.IP
	DHCPOptRootPath              DHCPOpt = 17  // n, string
	DHCPOptExtensionsPath        DHCPOpt = 18  // n, string
	DHCPOptIPForwarding          DHCPOpt = 19  // 1, bool
	DHCPOptSourceRouting         DHCPOpt = 20  // 1, bool
	DHCPOptPolicyFilter          DHCPOpt = 21  // 8*n, [n]{net.IP/net.IP}
	DHCPOptDatagramMTU           DHCPOpt = 22  // 2, uint16
	DHCPOptDefaultTTL            DHCPOpt = 23  // 1, byte
	DHCPOptPathMTUAgingTimeout   DHCPOpt = 24  // 4, uint32
	DHCPOptPathPlateuTableOption DHCPOpt = 25  // 2*n, []uint16
	DHCPOptInterfaceMTU          DHCPOpt = 26  // 2, uint16
	DHCPOptAllSubsLocal          DHCPOpt = 27  // 1, bool
	DHCPOptBroadcastAddr         DHCPOpt = 28  // 4, net.IP
	DHCPOptMaskDiscovery         DHCPOpt = 29  // 1, bool
	DHCPOptMaskSupplier          DHCPOpt = 30  // 1, bool
	DHCPOptRouterDiscovery       DHCPOpt = 31  // 1, bool
	DHCPOptSolicitAddr           DHCPOpt = 32  // 4, net.IP
	DHCPOptStaticRoute           DHCPOpt = 33  // n*8, [n]{net.IP/net.IP} -- note the 2nd is router not mask
	DHCPOptARPTrailers           DHCPOpt = 34  // 1, bool
	DHCPOptARPTimeout            DHCPOpt = 35  // 4, uint32
	DHCPOptEthernetEncap         DHCPOpt = 36  // 1, bool
	DHCPOptTCPTTL                DHCPOpt = 37  // 1, byte
	DHCPOptTCPKeepAliveInt       DHCPOpt = 38  // 4, uint32
	DHCPOptTCPKeepAliveGarbage   DHCPOpt = 39  // 1, bool
	DHCPOptNISDomain             DHCPOpt = 40  // n, string
	DHCPOptNISServers            DHCPOpt = 41  // 4*n,  [n]net.IP
	DHCPOptNTPServers            DHCPOpt = 42  // 4*n, [n]net.IP
	DHCPOptVendorOption          DHCPOpt = 43  // n, [n]byte // may be encapsulated.
	DHCPOptNetBIOSTCPNS          DHCPOpt = 44  // 4*n, [n]net.IP
	DHCPOptNetBIOSTCPDDS         DHCPOpt = 45  // 4*n, [n]net.IP
	DHCPOptNETBIOSTCPNodeType    DHCPOpt = 46  // 1, magic byte
	DHCPOptNetBIOSTCPScope       DHCPOpt = 47  // n, string
	DHCPOptXFontServer           DHCPOpt = 48  // n, string
	DHCPOptXDisplayManager       DHCPOpt = 49  // n, string
	DHCPOptRequestIP             DHCPOpt = 50  // 4, net.IP
	DHCPOptLeaseTime             DHCPOpt = 51  // 4, uint32
	DHCPOptExtOptions            DHCPOpt = 52  // 1, 1/2/3
	DHCPOptMessageType           DHCPOpt = 53  // 1, 1-7
	DHCPOptServerID              DHCPOpt = 54  // 4, net.IP
	DHCPOptParamsRequest         DHCPOpt = 55  // n, []byte
	DHCPOptMessage               DHCPOpt = 56  // n, 3
	DHCPOptMaxMessageSize        DHCPOpt = 57  // 2, uint16
	DHCPOptT1                    DHCPOpt = 58  // 4, uint32
	DHCPOptT2                    DHCPOpt = 59  // 4, uint32
	DHCPOptClassID               DHCPOpt = 60  // n, []byte
	DHCPOptClientID              DHCPOpt = 61  // n >=  2, []byte
	DHCPOptIPDomainName          DHCPOpt = 62  // n, str
	DHCPOptInformation           DHCPOpt = 63  // n,
	DHCPOptServiceDomain         DHCPOpt = 64  // n, str
	DHCPOptServiceServers        DHCPOpt = 65  // 4*n, [n]netIP
	DHCPOptTFTPServerName        DHCPOpt = 66  // n, str
	DHCPOptBootfileName          DHCPOpt = 67  // n, str
	DHCPOptMobileIPHomeAgent     DHCPOpt = 68  // 4*n, [n]net.IP
	DHCPOptSMTPServerOption      DHCPOpt = 69  // 4*n, [n]net.IP
	DHCPOptPOP3ServerOption      DHCPOpt = 70  // 4*n, [n]net.IP
	DHCPOptNNTPServerOption      DHCPOpt = 71  // 4*n, [n]net.IP
	DHCPOptWWWServerOption       DHCPOpt = 72  // 4*n, [n]net.IP
	DHCPOptFingerServerOption    DHCPOpt = 73  // 4*n, [n]net.IP
	DHCPOptIRCServerOption       DHCPOpt = 74  // 4*n, [n]net.IP
	DHCPOptStreeTalkServerOption DHCPOpt = 75  // 4*n, [n]netIP
	DHCPOptSTDAServerOption      DHCPOpt = 76  // 4*n, [n]netIP
	DHCPOptUserClassInformation  DHCPOpt = 77  // n, str
	DHCPOptSLPDirectoryAgent     DHCPOpt = 78  // unknown
	DHCPOptSLPServiceScope       DHCPOpt = 79  // unknown
	DHCPOptRapidCommit           DHCPOpt = 80  // 0
	DHCPOptFQDN                  DHCPOpt = 81  // n, str
	DHCPOptRelayAgent            DHCPOpt = 82  // n, str
	DHCPOptInternetStorage       DHCPOpt = 83  // unknown
	DHCPOptNDSServers            DHCPOpt = 85  // unknown
	DHCPOptNDSTreeName           DHCPOpt = 86  // unknown
	DHCPOptNDSContext            DHCPOpt = 87  // unknown
	DHCPOptBCMCSDomainName       DHCPOpt = 88  // n, str
	DHCPOptBCMCSIPv4Adress       DHCPOpt = 89  // 4*n, [n]net.IP
	DHCPOptAuthentication        DHCPOpt = 90  // unknown
	DHCPOptLastTransactionTime   DHCPOpt = 91  // 4
	DHCPOptAssociatedIP          DHCPOpt = 92  // 4*n, [n]net.IP
	DHCPOptCSAT                  DHCPOpt = 93  // unknown
	DHCPOptCNII                  DHCPOpt = 94  //unknown
	DHCPOptLDAP                  DHCPOpt = 95  // unknown
	DHCPOptCMI                   DHCPOpt = 97  //unknown
	DHCPOptOGUA                  DHCPOpt = 98  //unknown
	DHCPOptGEOCONF_CIVIC         DHCPOpt = 99  //unknown
	DHCPOptIEEE1003_1TZString    DHCPOpt = 100 // n, str
	DHCPOptRefToTZDatabase       DHCPOpt = 101 //unknown
	DHCPOptParentServerAddress   DHCPOpt = 112 //unknown
	DHCPOptParentServerTag       DHCPOpt = 113 //unknown
	DHCPOptURL                   DHCPOpt = 114 //unknown
	DHCPOptAutoConfigure         DHCPOpt = 116 //1 byte
	DHCPOptNameServiceSearch     DHCPOpt = 117 //unknown
	DHCPOptSubnetSelection       DHCPOpt = 118 //4 byte
	DHCPOptDomainSearch          DHCPOpt = 119 // n, string
	DHCPOptSIPServers            DHCPOpt = 120 // n, url
	DHCPOptClasslessStaticRoute  DHCPOpt = 121 //
	DHCPOptCCC                   DHCPOpt = 122 //
	DHCPOptGeoConf               DHCPOpt = 123 //
	DHCPOptVendorIdentifyVClass  DHCPOpt = 124 //
	DHCPOptVendorIdentifySpec    DHCPOpt = 125 //
	DHCPOptTFTPServerIP          DHCPOpt = 128 //
	DHCPOptCallServerIP          DHCPOpt = 129 //
	DHCPOptDiscriminationStr     DHCPOpt = 130 //
	DHCPOptRemoteStatisticsIP    DHCPOpt = 131 //
	DHCPOpt802_1PVLANID          DHCPOpt = 132 //
	DHCPOpt802_1QL2Priority      DHCPOpt = 133 //
	DHCPOptDiffservCodePoint     DHCPOpt = 134 //
	DHCPOptHTTPProxyForPhone     DHCPOpt = 135 //
	DHCPOptPANAAuthentication    DHCPOpt = 136 //
	DHCPOptLoSTServer            DHCPOpt = 137 //
	DHCPOptCAPWPACAdress         DHCPOpt = 138 //
	DHCPOptOPTIONV4AddressMos    DHCPOpt = 139 //
	DHCPOptOPTIONV4FQDNMos       DHCPOpt = 140 //
	DHCPOptSIPUACSD              DHCPOpt = 141 //
	DHCPOptOPTIONV4ANDSF         DHCPOpt = 142 //
	DHCPOptOPTIONV6ANDSF         DHCPOpt = 143 //
	DHCPOptTFTPServerAddress     DHCPOpt = 150 //
	DHCPOptPXELinuxMagic         DHCPOpt = 208 //
	DHCPOptPXELinuxConfigFile    DHCPOpt = 209 //
	DHCPOptPXELinuxPathPrefix    DHCPOpt = 210 //
	DHCPOptPXELinuxRebootTime    DHCPOpt = 211 //
	DHCPOptOPTION_6RD            DHCPOpt = 212 //
	DHCPOptOPTIONV4AccessDomain  DHCPOpt = 213 //
	DHCPOptVirtualSubnetSelect   DHCPOpt = 221 //
	DHCPOptEnd                   DHCPOpt = 255
)

// String returns a string version of a DHCPOpt.
func (o DHCPOpt) String() string {
	switch o {
	case DHCPOptPad:
		return "(padding)"
	case DHCPOptSubnetMask:
		return "SubnetMask"
	case DHCPOptTimeOffset:
		return "TimeOffset"
	case DHCPOptRouter:
		return "Router"
	case DHCPOptTimeServer:
		return "rfc868" // old time server protocol stringified to dissuade confusion w. NTP
	case DHCPOptNameServer:
		return "ien116" // obscure nameserver protocol stringified to dissuade confusion w. DNS
	case DHCPOptDNS:
		return "DNS"
	case DHCPOptLogServer:
		return "mitLCS" // MIT LCS server protocol yada yada w. Syslog
	case DHCPOptCookieServer:
		return "CookieServer"
	case DHCPOptLPRServer:
		return "LPRServer"
	case DHCPOptImpressServer:
		return "ImpressServer"
	case DHCPOptResLocServer:
		return "ResourceLocationServer"
	case DHCPOptHostname:
		return "Hostname"
	case DHCPOptBootfileSize:
		return "BootfileSize"
	case DHCPOptMeritDumpFile:
		return "MeritDumpFile"
	case DHCPOptDomainName:
		return "DomainName"
	case DHCPOptSwapServer:
		return "SwapServer"
	case DHCPOptRootPath:
		return "RootPath"
	case DHCPOptExtensionsPath:
		return "ExtensionsPath"
	case DHCPOptIPForwarding:
		return "IPForwarding"
	case DHCPOptSourceRouting:
		return "SourceRouting"
	case DHCPOptPolicyFilter:
		return "PolicyFilter"
	case DHCPOptDatagramMTU:
		return "DatagramMTU"
	case DHCPOptDefaultTTL:
		return "DefaultTTL"
	case DHCPOptPathMTUAgingTimeout:
		return "PathMTUAgingTimeout"
	case DHCPOptPathPlateuTableOption:
		return "PathPlateuTableOption"
	case DHCPOptInterfaceMTU:
		return "InterfaceMTU"
	case DHCPOptAllSubsLocal:
		return "AllSubsLocal"
	case DHCPOptBroadcastAddr:
		return "BroadcastAddress"
	case DHCPOptMaskDiscovery:
		return "MaskDiscovery"
	case DHCPOptMaskSupplier:
		return "MaskSupplier"
	case DHCPOptRouterDiscovery:
		return "RouterDiscovery"
	case DHCPOptSolicitAddr:
		return "SolicitAddr"
	case DHCPOptStaticRoute:
		return "StaticRoute"
	case DHCPOptARPTrailers:
		return "ARPTrailers"
	case DHCPOptARPTimeout:
		return "ARPTimeout"
	case DHCPOptEthernetEncap:
		return "EthernetEncap"
	case DHCPOptTCPTTL:
		return "TCPTTL"
	case DHCPOptTCPKeepAliveInt:
		return "TCPKeepAliveInt"
	case DHCPOptTCPKeepAliveGarbage:
		return "TCPKeepAliveGarbage"
	case DHCPOptNISDomain:
		return "NISDomain"
	case DHCPOptNISServers:
		return "NISServers"
	case DHCPOptNTPServers:
		return "NTPServers"
	case DHCPOptVendorOption:
		return "VendorOption"
	case DHCPOptNetBIOSTCPNS:
		return "NetBIOSOverTCPNS"
	case DHCPOptNetBIOSTCPDDS:
		return "NetBiosOverTCPDDS"
	case DHCPOptNETBIOSTCPNodeType:
		return "NetBIOSOverTCPNodeType"
	case DHCPOptNetBIOSTCPScope:
		return "NetBIOSOverTCPScope"
	case DHCPOptXFontServer:
		return "XFontServer"
	case DHCPOptXDisplayManager:
		return "XDisplayManager"
	case DHCPOptEnd:
		return "(end)"
	case DHCPOptSIPServers:
		return "SipServers"
	case DHCPOptRequestIP:
		return "RequestIP"
	case DHCPOptLeaseTime:
		return "LeaseTime"
	case DHCPOptExtOptions:
		return "ExtOpts"
	case DHCPOptMessageType:
		return "MessageType"
	case DHCPOptServerID:
		return "ServerID"
	case DHCPOptParamsRequest:
		return "ParamsRequest"
	case DHCPOptMessage:
		return "Message"
	case DHCPOptMaxMessageSize:
		return "MaxDHCPSize"
	case DHCPOptT1:
		return "Timer1"
	case DHCPOptT2:
		return "Timer2"
	case DHCPOptClassID:
		return "ClassID"
	case DHCPOptClientID:
		return "ClientID"
	case DHCPOptIPDomainName:
		return "NetWare/IP Domain Name"
	case DHCPOptInformation:
		return "NetWare/IP information"
	case DHCPOptServiceDomain:
		return "Network Information Service+ Domain"
	case DHCPOptServiceServers:
		return "Network Information Service+ Servers"
	case DHCPOptTFTPServerName:
		return "TFTP server name"
	case DHCPOptBootfileName:
		return "Bootfile name"
	case DHCPOptMobileIPHomeAgent:
		return "Mobile IP Home Agent"
	case DHCPOptSMTPServerOption:
		return "Simple Mail Transport Protocol Server"
	case DHCPOptPOP3ServerOption:
		return "Post Office Protocol Server"
	case DHCPOptNNTPServerOption:
		return "Network News Transport Protocol Server"
	case DHCPOptWWWServerOption:
		return "Default World Wide Web Server"
	case DHCPOptFingerServerOption:
		return "Default Finger Server"
	case DHCPOptIRCServerOption:
		return "Default Internet Relay Chat Server"
	case DHCPOptStreeTalkServerOption:
		return "StreetTalk Server"
	case DHCPOptSTDAServerOption:
		return "StreetTalk Directory Assistance Server"
	case DHCPOptUserClassInformation:
		return "User Class Information"
	case DHCPOptSLPDirectoryAgent:
		return "SLP Directory Agent"
	case DHCPOptSLPServiceScope:
		return "SLP Service Scope"
	case DHCPOptRapidCommit:
		return "Rapid Commit"
	case DHCPOptFQDN:
		return "FQDN, Fully Qualified Domain Name"
	case DHCPOptRelayAgent:
		return "Relay Agent Information"
	case DHCPOptInternetStorage:
		return "Internet Storage Name Service"
	case DHCPOptNDSServers:
		return "NDS servers"
	case DHCPOptNDSTreeName:
		return "NDS tree name"
	case DHCPOptNDSContext:
		return "NDS context"
	case DHCPOptBCMCSDomainName:
		return "BCMCS Controller Domain Name list"
	case DHCPOptBCMCSIPv4Adress:
		return "BCMCS Controller IPv4 address list"
	case DHCPOptAuthentication:
		return "Authentication"
	case DHCPOptLastTransactionTime:
		return "client-last-transaction-time"
	case DHCPOptAssociatedIP:
		return "associated-ip"
	case DHCPOptCSAT:
		return "Client System Architecture Type"
	case DHCPOptCNII:
		return "Client Network Interface Identifier"
	case DHCPOptLDAP:
		return "LDAP, Lightweight Directory Access Protocol"
	case DHCPOptCMI:
		return "Client Machine Identifier"
	case DHCPOptOGUA:
		return "Open Group's User Authentication"
	case DHCPOptGEOCONF_CIVIC:
		return "GEOCONF_CIVIC"
	case DHCPOptIEEE1003_1TZString:
		return "IEEE 1003.1 TZ String"
	case DHCPOptRefToTZDatabase:
		return "Reference to the TZ Database"
	case DHCPOptParentServerAddress:
		return "NetInfo Parent Server Address"
	case DHCPOptParentServerTag:
		return "NetInfo Parent Server Tag"
	case DHCPOptURL:
		return "URL"
	case DHCPOptAutoConfigure:
		return "Auto-Configure"
	case DHCPOptNameServiceSearch:
		return "Name Service Search"
	case DHCPOptSubnetSelection:
		return "Subnet Selection"
	case DHCPOptDomainSearch:
		return "DomainSearch"
	case DHCPOptClasslessStaticRoute:
		return "ClasslessStaticRoute"
	case DHCPOptCCC:
		return "CCC, CableLabs Client Configuration"
	case DHCPOptGeoConf:
		return "GeoConf"
	case DHCPOptVendorIdentifyVClass:
		return "Vendor-Identifying Vendor Class"
	case DHCPOptVendorIdentifySpec:
		return "Vendor-Identifying Vendor-Specific"
	case DHCPOptTFTPServerIP:
		return "TFTP Server IP address"
	case DHCPOptCallServerIP:
		return "Call Server IP address"
	case DHCPOptDiscriminationStr:
		return "Discrimination string"
	case DHCPOptRemoteStatisticsIP:
		return "Remote statistics server IP addres"
	case DHCPOpt802_1PVLANID:
		return "802.1P VLAN ID"
	case DHCPOpt802_1QL2Priority:
		return "802.1Q L2 Priority"
	case DHCPOptDiffservCodePoint:
		return "Diffserv Code Point"
	case DHCPOptHTTPProxyForPhone:
		return "HTTP Proxy for phone-specific applications"
	case DHCPOptPANAAuthentication:
		return "PANA Authentication Agent"
	case DHCPOptLoSTServer:
		return "LoST Server"
	case DHCPOptCAPWPACAdress:
		return "CAPWAP Access Controller addresse"
	case DHCPOptOPTIONV4AddressMos:
		return "OPTION-IPv4_Address-MoS"
	case DHCPOptOPTIONV4FQDNMos:
		return "OPTION-IPv4_FQDN-MoS"
	case DHCPOptSIPUACSD:
		return "SIP UA Configuration Service Domains"
	case DHCPOptOPTIONV4ANDSF:
		return "OPTION-IPv4_Address-ANDSF"
	case DHCPOptOPTIONV6ANDSF:
		return "OPTION-IPv6_Address-ANDSF"
	case DHCPOptTFTPServerAddress:
		return "TFTP server address"
	case DHCPOptPXELinuxMagic:
		return "pxelinux.magic (string) F1:00:74:7E (241.0.116.126)"
	case DHCPOptPXELinuxConfigFile:
		return "pxelinux.configfile (text)"
	case DHCPOptPXELinuxPathPrefix:
		return "pxelinux.pathprefix (text)"
	case DHCPOptPXELinuxRebootTime:
		return "pxelinux.reboottime (unsigned integer 32 bits)"
	case DHCPOptOPTION_6RD:
		return "OPTION_6RD"
	case DHCPOptOPTIONV4AccessDomain:
		return "OPTION_V4_ACCESS_DOMAIN"
	case DHCPOptVirtualSubnetSelect:
		return "Virtual Subnet Selection"
	default:
		return "Unknown"
	}
}

// DHCPOption rerpresents a DHCP option.
type DHCPOption struct {
	Type   DHCPOpt
	Length uint8
	Data   []byte
}

// String returns a string version of a DHCP Option.
func (o DHCPOption) String() string {
	switch o.Type {

	case DHCPOptHostname, DHCPOptMeritDumpFile, DHCPOptDomainName, DHCPOptRootPath,
		DHCPOptExtensionsPath, DHCPOptNISDomain, DHCPOptNetBIOSTCPScope, DHCPOptXFontServer,
		DHCPOptXDisplayManager, DHCPOptMessage, DHCPOptDomainSearch, DHCPOptClassID: // string
		return fmt.Sprintf("%d (%s): %s", byte(o.Type), o.Type, string(o.Data))

	case DHCPOptMessageType:
		if len(o.Data) != 1 {
			return fmt.Sprintf("%d (%s) :INVALID)", byte(o.Type), o.Type)
		}
		return fmt.Sprintf("%d (%s): %s", byte(o.Type), o.Type, DHCPMsgType(o.Data[0]))

	case DHCPOptSubnetMask, DHCPOptServerID, DHCPOptBroadcastAddr,
		DHCPOptSolicitAddr, DHCPOptRequestIP, DHCPOptRouter: // net.IP
		if len(o.Data) < 4 {
			return fmt.Sprintf("%d (%s): INVALID)", byte(o.Type), o.Type)
		}
		return fmt.Sprintf("%d (%s): %s", byte(o.Type), o.Type, net.IP(o.Data))

	case DHCPOptT1, DHCPOptT2, DHCPOptLeaseTime, DHCPOptPathMTUAgingTimeout,
		DHCPOptARPTimeout, DHCPOptTCPKeepAliveInt: // uint32
		if len(o.Data) != 4 {
			return fmt.Sprintf("%d (%s): INVALID)", byte(o.Type), o.Type)
		}
		return fmt.Sprintf("%d (%s): %d", byte(o.Type), o.Type,
			uint32(o.Data[0])<<24|uint32(o.Data[1])<<16|uint32(o.Data[2])<<8|uint32(o.Data[3]))

	case DHCPOptParamsRequest:
		buf := &bytes.Buffer{}
		buf.WriteString(fmt.Sprintf("%d (%s): ", byte(o.Type), o.Type))
		for i, v := range o.Data {
			buf.WriteString(DHCPOpt(v).String())
			if i+1 != len(o.Data) {
				buf.WriteByte(',')
			}
		}
		return buf.String()
	case DHCPOptClientID:
		if len(o.Data) < 2 {
			return fmt.Sprintf("%d (%s): INVALID", byte(o.Type), o.Type)
		}
		return fmt.Sprintf("%d (%s): %v", byte(o.Type), o.Type, o.Data)
	case DHCPOptDNS:
		if len(o.Data) % 4 != 0 {
			return fmt.Sprint("%d (%s): INVALID", byte(o.Type), o.Type)
		}
		buf := &bytes.Buffer{}
		buf.WriteString(fmt.Sprintf("%d (%s): ", byte(o.Type), o.Type))
		for i := 0; i<len(o.Data); i = i+4 {
			dns := o.Data[i:i+4]
			buf.WriteString(net.IP(dns).String())
			if i+4 != len(o.Data) {
				buf.WriteByte(',')
			}
		}
		return buf.String()

	default:
		return fmt.Sprintf("%d (%s): 0x%s", byte(o.Type), o.Type, hex.EncodeToString(o.Data))
	}
}

// NewDHCPOption constructs a new DHCPOption with a given type and data.
func NewDHCPOption(t DHCPOpt, data []byte) DHCPOption {
	o := DHCPOption{Type: t}
	if data != nil {
		o.Data = data
		o.Length = uint8(len(data))
	}
	return o
}

func (o *DHCPOption) encode(b []byte) error {
	switch o.Type {
	case DHCPOptPad, DHCPOptEnd:
		b[0] = byte(o.Type)
	default:
		b[0] = byte(o.Type)
		b[1] = o.Length
		copy(b[2:], o.Data)
	}
	return nil
}

func (o *DHCPOption) decode(data []byte) error {
	if len(data) < 1 {
		// Pad/End have a length of 1
		return DecOptionNotEnoughData
	}
	o.Type = DHCPOpt(data[0])
	switch o.Type {
	case DHCPOptPad, DHCPOptEnd:
		o.Data = nil
	default:
		if len(data) < 2 {
			return DecOptionNotEnoughData
		}
		o.Length = data[1]
		if int(o.Length) > len(data[2:]) {
			return DecOptionMalformed
		}
		o.Data = data[2 : 2+int(o.Length)]
	}
	return nil
}

func (o *DHCPOption) AddByte(b byte) {
	o.Data = append(o.Data, b)
	o.Length++
}

// DHCPv4Error is used for constant errors for DHCPv4. It is needed for test asserts.
type DHCPv4Error string

// DHCPv4Error implements error interface.
func (d DHCPv4Error) Error() string {
	return string(d)
}

const (
	// DecOptionNotEnoughData is returned when there is not enough data during option's decode process
	DecOptionNotEnoughData = DHCPv4Error("Not enough data to decode")
	// DecOptionMalformed is returned when the option is malformed
	DecOptionMalformed = DHCPv4Error("Option is malformed")
	// InvalidMagicCookie is returned when Magic cookie is missing into BOOTP header
	InvalidMagicCookie = DHCPv4Error("Bad DHCP header")
)

