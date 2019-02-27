package connection

import (
	"fmt"
	"github.com/libp2p/go-reuseport"
	"net"
)

type Dialer func(*net.UDPAddr, *net.UDPAddr) (net.Conn, error)

type Listener func(*net.UDPAddr) (net.PacketConn, error)

type TransPort struct {
	Dialer Dialer
	Listener Listener
}

func (t *TransPort) Dial(l *net.UDPAddr, r *net.UDPAddr) (net.Conn, error) {
	return t.Dialer(l,r)
}

func (t *TransPort) Listen(l *net.UDPAddr) (net.PacketConn, error) {
	return t.Listener(l)
}

func UDPDialer() Dialer {
	return func(l *net.UDPAddr, r *net.UDPAddr) (net.Conn, error) {
		conn, err := reuseport.Dial("udp", l.String(), r.String())
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		return conn, nil
	}
}

func UDPListener() Listener {
	return func(l *net.UDPAddr) (net.PacketConn, error) {
		packetConn, err := reuseport.ListenPacket("udp", l.String())
		if err != nil {
			return nil, err
		}
		return packetConn, nil
	}
}

