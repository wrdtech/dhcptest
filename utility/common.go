package utility

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

func GetInterfaceByIP(ip string, validip map[string]net.Interface) (*net.Interface, error) {
	if ip == "0.0.0.0" {
		return &net.Interface{Name:""}, nil
	}
	iface, ok := validip[ip]
	if !ok {
		return nil, fmt.Errorf("invalid ip:%s", ip)
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
