package connection

import (
	"dhcptest/layers"
	"encoding/binary"
	"net"
)

type Modifier func(*layers.DHCPv4)


// WithTransactionID sets the Transaction ID for a layers.DHCPv4 packet.
func WithTransactionID(xid uint32) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.Xid = xid
	}
}

// WithClientIP sets the Client IP for a layers.DHCPv4 packet.
func WithClientIP(ip net.IP) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.ClientIP = ip
	}

}

// WithYourIP sets the Your IP for a layers.DHCPv4 packet.
func WithYourIP(ip net.IP) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.YourClientIP = ip
	}
}

// WithServerIP sets the Server IP for a layers.DHCPv4 packet.
func WithServerIP(ip net.IP) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.NextServerIP = ip
	}
}

// WithReply fills in opcode, hwtype, xid, clienthwaddr, flags, and gateway ip
// addr from the given packet.
func WithReply(request *layers.DHCPv4) Modifier {
	return func(packet *layers.DHCPv4) {
		if request.Operation == layers.DHCPOpRequest {
			packet.Operation = layers.DHCPOpReply
		} else {
			packet.Operation = layers.DHCPOpRequest
		}
		packet.HardwareType = request.HardwareType
		packet.Xid = request.Xid
		packet.ClientHWAddr = request.ClientHWAddr
		packet.Flags = request.Flags
		packet.RelayAgentIP = request.RelayAgentIP
	}
}

// WithHWType sets the Hardware Type for a layers.DHCPv4 packet.
func WithHWType(hwt layers.LinkType) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.HardwareType = hwt
	}
}

// WithBroadcast sets the packet to be broadcast or unicast
func WithBroadcast(broadcast bool) Modifier {
	return func(packet *layers.DHCPv4) {
		if broadcast {
			packet.SetBroadcast()
		} else {
			packet.SetUnicast()
		}
	}
}

// WithHwAddr sets the hardware address for a packet
func WithHwAddr(hwaddr net.HardwareAddr) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.ClientHWAddr = hwaddr
		packet.HardwareLen = uint8(len(packet.ClientHWAddr))
	}
}

// WithOption appends a layers.DHCPv4 option provided by the user
func WithOption(opt layers.DHCPOpt, data []byte) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.AddOption(opt, data)
	}
}

func WithHostName(hostname string) Modifier {
	return WithOption(layers.DHCPOptHostname, []byte(hostname))
}

/*
haven't resolve it yet

// WithUserClass adds a user class option to the packet.
// The rfc parameter allows you to specify if the userclass should be
// rfc compliant or not. More details in issue #113
func WithUserClass(uc []byte, rfc bool) Modifier {
	// TODO let the user specify multiple user classes
	return func(packet *layers.DHCPv4) {
		if rfc {
			packet.AddOption(OptRFC3004UserClass([][]byte{uc}))
		} else {
			packet.UpdateOption(OptUserClass(uc))
		}
	}
}
*/

// WithNetboot adds bootfile URL and bootfile param options to a layers.DHCPv4 packet.
func WithNetboot(d *layers.DHCPv4) {
	WithRequestedOptions(layers.DHCPOptTFTPServerName, layers.DHCPOptBootfileName)(d)
}

// WithMessageType adds the layers.DHCPv4 message type m to a packet.
func WithMessageType(m layers.DHCPMsgType) Modifier {
	return WithOption(layers.DHCPOptMessageType, []byte{byte(m)})
}

// WithRequestedOptions adds requested options to the packet.
func WithRequestedOptions(optionCodes ...layers.DHCPOpt) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.AddParamRequest(optionCodes...)
	}
}

// WithRelay adds parameters required for layers.DHCPv4 to be relayed by the relay
// server with given ip
func WithRelay(ip net.IP) Modifier {
	return func(packet *layers.DHCPv4) {
		packet.SetUnicast()
		packet.RelayAgentIP = ip
		packet.HardwareOpts++
	}
}

// WithNetmask adds or updates an OptSubnetMask
func WithNetmask(mask net.IPMask) Modifier {
	return WithOption(layers.DHCPOptSubnetMask, []byte(mask))
}

// WithLeaseTime adds or updates an OptIPAddressLeaseTime
func WithLeaseTime(leaseTime uint32) Modifier {
	var data []byte
	binary.BigEndian.PutUint32(data, leaseTime)
	return WithOption(layers.DHCPOptLeaseTime, data)
}

/*
haven't resolve it yet

// WithDomainSearchList adds or updates an OptionDomainSearch
func WithDomainSearchList(searchList ...string) Modifier {
	return WithOption(OptDomainSearch(&rfc1035label.Labels{
		Labels: searchList,
	}))
}
*/

func WithGeneric(code layers.DHCPOpt, value []byte) Modifier {
	return WithOption(code, value)
}
