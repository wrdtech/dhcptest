package utility

import (
	"bytes"
	"dhcptest/layers"
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	"math/rand"
	"net"
	"strconv"
	"time"
)

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

func IsEqualBetweenByteSliceAndArray(slice []byte, array []byte) bool {
	if slice == nil {
		return false
	}
	if len(slice) != len(array) {
		return false
	}
	for i, v := range slice {
		if v != array[i] {
			return false
		}
	}
	return true
}

func GetInterfaceByIP(ip string, validip map[string]net.Interface) (*net.Interface, error) {
	iface, ok := validip[ip]
	if !ok {
		return nil, fmt.Errorf("invalid ip:%s", ip)
	}
	return &iface, nil
}

func ParseOption(options string) (layers.DHCPOptions ,error){
	var code,value string
	var dhcpOptions layers.DHCPOptions
	buf := bytes.Buffer{}
	for i:=0 ; i < len(options); i++ {
		switch options[i] {
		case '=':
			code = buf.String()
			buf.Reset()
		case ';':
			if len(code) == 0 {
				return dhcpOptions, fmt.Errorf("%s invalid option code", code)
			}

			optionCode, err := strconv.Atoi(code)
			if err != nil {
				return dhcpOptions, err
			}

			value = buf.String()
			dhcpOption, err := layers.ParseString(layers.DHCPOpt(optionCode), value)
			dhcpOptions = append(dhcpOptions, dhcpOption)
		default:
			buf.WriteRune(rune(options[i]))
			fmt.Println(buf.String())
		}
	}
	return dhcpOptions, nil
}

func RandomMac() string {
	m := []byte{2, 0, 0}

	rand.Seed(time.Now().UnixNano())

	for i := 3; i < 6; i++ {
		macByte := rand.Intn(256)
		m = append(m, byte(macByte))

        rand.Seed(int64(macByte))
	}

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", m[0], m[1], m[2], m[3], m[4], m[5])

}

func parseIPs(data []byte) []net.IP {
	result := make([]net.IP, len(data)/4)
	for i:=0; i+3 < len(data); i +=4 {
		result[i/4] = net.IP(data[i : i+4])
	}
	return result
}

func ParsePacket(data []byte) *layers.DHCPv4 {
	packet := gopacket.NewPacket(data, layers.LayerTypeDHCPv4, gopacket.Default)

	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)

	if dhcpLayer == nil {
		return nil
	}

	return dhcpLayer.(*layers.DHCPv4)
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
			lease.Router = parseIPs(option.Data)
			break
		case layers.DHCPOptDNS:
			lease.DNS = parseIPs(option.Data)
			break
		case layers.DHCPOptTimeServer:
			lease.TimeServer = parseIPs(option.Data)
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

func getFileString(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	for _, d := range data {
		if d == 0 {
			continue
		}
		return string(data)
	}
	return ""
}
