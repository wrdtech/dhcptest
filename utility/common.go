package utility

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

func GetInterfaceByName(ifaceName string, validIface map[string]net.Interface) (*net.Interface, error) {
	iface, ok := validIface[ifaceName]
	if !ok {
		return nil, fmt.Errorf("invalid iface:%s", iface)
	}
	return &iface, nil
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

func ParseIPs(data []byte) []net.IP {
	result := make([]net.IP, len(data)/4)
	for i:=0; i+3 < len(data); i +=4 {
		result[i/4] = net.IP(data[i : i+4])
	}
	return result
}


func GetFileString(data []byte) string {
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

func GetUnicastIPofInterface(ifi *net.Interface) ([]net.IP, error) {
	res := make([]net.IP, 0)
	addrs , err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ipAddr, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil, err
		}
		ipAddr = ipAddr.To4()
		if ipAddr == nil {
			continue
		}
		if ipAddr.IsGlobalUnicast() {
			res = append(res, ipAddr)
		}
	}
	return res, nil

}
