package connection

import (
	"dhcptest/layers"
	"dhcptest/utility"
	"encoding/binary"
	"github.com/google/gopacket"
	"net"
	"time"
)

const (
	MAXUDPReceivedPacketSize = 8192
)

var (
	DefaultReadTimeout = 2 * time.Second
	DefaultWriteTimeout = 2 * time.Second
    DefaultParamsRequestList = []layers.DHCPOpt{
		layers.DHCPOptSubnetMask,
		layers.DHCPOptRouter,
		layers.DHCPOptTimeServer,
		layers.DHCPOptDNS,
		layers.DHCPOptDomainName,
		layers.DHCPOptInterfaceMTU,
		layers.DHCPOptNTPServers,
	}
)

func NewPacket(dhcpOptions ...layers.DHCPOption) *layers.DHCPv4 {
	packet := layers.DHCPv4{
		Operation: layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		Flags: uint16(layers.BroadcastFlag),
		ClientIP: net.ParseIP("0.0.0.0"),
		YourClientIP: net.ParseIP("0.0.0.0"),
		NextServerIP: net.ParseIP("0.0.0.0"),
		RelayAgentIP: net.ParseIP("0.0.0.0"),
	}

	for _, dhcpOption := range dhcpOptions {
		packet.AddOption(dhcpOption.Type, dhcpOption.Data)
	}

	return &packet
}

func NewRequestFromOffer(packet *layers.DHCPv4) *layers.DHCPv4 {
	_, lease := NewLease(packet)
	fixedAddress :=  []byte(lease.FixedAddress)
	serverID := []byte(lease.ServerID)
	requestPacket := NewPacket(utility.DhcpOptions...)
	WithReply(packet)(requestPacket)
	packet.AddOption(layers.DHCPOptRequestIP, fixedAddress)
	packet.AddOption(layers.DHCPOptServerID, serverID)
	return requestPacket
}

func ParsePacket(data []byte) *layers.DHCPv4 {
	packet := gopacket.NewPacket(data, layers.LayerTypeDHCPv4, gopacket.Default)

	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)

	if dhcpLayer == nil {
		return nil
	}

	return dhcpLayer.(*layers.DHCPv4)
}

type PacketResponse struct {
	dispatcher *PacketEventDispatcher
	dLastTimer *time.Timer
	rLastTimer *time.Timer
	packets    map[layers.DHCPMsgType][]*layers.DHCPv4

}


func (pr *PacketResponse) Call(event PacketEvent) {
		pr.dispatcher.DispatchEvent(event)
}

func (pr *PacketResponse) AddPacket(packet *layers.DHCPv4) {
	pr.packets[packet.MessageType()] = append(pr.packets[packet.MessageType()], packet)
}

func NewPacketResponse() *PacketResponse {
	pr := &PacketResponse{}
	pr.packets = make(map[layers.DHCPMsgType][]*layers.DHCPv4)
	pr.dispatcher = new(PacketEventDispatcher)
	pr.dispatcher.AddEventListener(discoverDequeue, func(e PacketEvent) {
		packet := e.object.(*layers.DHCPv4)
		pr.AddPacket(packet)
		pr.dLastTimer = time.NewTimer(utility.Timeout)
	})
	pr.dispatcher.AddEventListener(receivedOffer, func(e PacketEvent) {
		select {
		case <- pr.dLastTimer.C:
			pr.Call(NewEvent(offerTimeout, nil))
		default:
			packet := e.object.(*layers.DHCPv4)
			pr.AddPacket(packet)
		}
	})
	pr.dispatcher.AddEventListener(offerTimeout, func(e PacketEvent) {
		pr.dispatcher.RemoveEventListener(receivedOffer)
		pr.dLastTimer.Stop()
	})
	pr.dispatcher.AddEventListener(requestDequeue, func(e PacketEvent) {
		packet := e.object.(*layers.DHCPv4)
		pr.AddPacket(packet)
		pr.rLastTimer = time.NewTimer(utility.Timeout)
	})
	pr.dispatcher.AddEventListener(receivedAck, func (e PacketEvent) {
		select {
		case <- pr.rLastTimer.C:
			pr.Call(NewEvent(ackNakTimeout, nil))
		default:
			packet := e.object.(*layers.DHCPv4)
			pr.AddPacket(packet)
		}
	})
	pr.dispatcher.AddEventListener(receivedNak, func (e PacketEvent) {
		select {
		case <- pr.rLastTimer.C:
			pr.Call(NewEvent(ackNakTimeout, nil))
		default:
			packet := e.object.(*layers.DHCPv4)
			pr.AddPacket(packet)
		}
	})
	pr.dispatcher.AddEventListener(ackNakTimeout, func(e PacketEvent) {
		pr.dispatcher.RemoveEventListener(receivedAck)
		pr.dispatcher.RemoveEventListener(receivedNak)
		pr.rLastTimer.Stop()
	})
	return pr
}

type Lease struct {
	ServerID net.IP
	FixedAddress net.IP
	Netmask net.IPMask
	NextServer net.IP
	Broadcast net.IP
	Router []net.IP
	DNS []net.IP
	TimeServer []net.IP
	DomainName string
	MTU uint16

	Bound time.Time
	Renew time.Time
	Rebind time.Time
	Expire time.Time
}

func NewLease(packet *layers.DHCPv4) (msgType layers.DHCPMsgType, lease Lease) {
	lease.Bound = time.Now()
	lease.FixedAddress = packet.YourClientIP

	for _,option := range packet.Options {
		switch option.Type {
		case layers.DHCPOptMessageType:
			if option.Length == 1 {
				msgType  = layers.DHCPMsgType(option.Data[0])
			}
			break
		case layers.DHCPOptSubnetMask:
			lease.Netmask = net.IPMask(option.Data)
			break
		case layers.DHCPOptBroadcastAddr:
			lease.Broadcast = net.IP(option.Data)
			break
		case layers.DHCPOptServerID:
			lease.ServerID = net.IP(option.Data)
			break
		case layers.DHCPOptRouter:
			lease.Router = utility.ParseIPs(option.Data)
			break
		case layers.DHCPOptDNS:
			lease.DNS = utility.ParseIPs(option.Data)
			break
		case layers.DHCPOptTimeServer:
			lease.TimeServer = utility.ParseIPs(option.Data)
			break
		case layers.DHCPOptDomainName:
			lease.DomainName = string(option.Data)
			break
		case layers.DHCPOptInterfaceMTU:
			if option.Length == 2 {
				lease.MTU = binary.BigEndian.Uint16(option.Data)
			}
			break
		case layers.DHCPOptLeaseTime:
			if option.Length == 4 {
				lease.Expire = lease.Bound.Add(time.Second * time.Duration(binary.BigEndian.Uint32(option.Data)))
			}
			break
		case layers.DHCPOptT1:
			if option.Length == 4 {
				lease.Renew = lease.Bound.Add(time.Second * time.Duration(binary.BigEndian.Uint32(option.Data)))
			}
			break
		case layers.DHCPOptT2:
			if option.Length == 4 {
				lease.Rebind = lease.Bound.Add(time.Second * time.Duration(binary.BigEndian.Uint32(option.Data)))
			}
			break
		default:
			break
		}
	}
	return
}
