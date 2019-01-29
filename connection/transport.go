package connection

import (
	"fmt"
	"net"
)

type Dialer func(*net.UDPAddr, *net.UDPAddr) (*net.UDPConn, error)

type Listener func(*net.UDPAddr) (*net.UDPConn, error)

type TransPort struct {
	Dialer Dialer
	Listener Listener
}

func (t *TransPort) Dial(l *net.UDPAddr, r *net.UDPAddr) (*net.UDPConn, error) {
	return t.Dialer(l,r)
}

func (t *TransPort) Listen(l *net.UDPAddr) (*net.UDPConn, error) {
	return t.Listener(l)
}

func UDPDialer() Dialer {
	return func(l *net.UDPAddr, r *net.UDPAddr) (*net.UDPConn, error) {
		conn, err := net.DialUDP("udp", l, r)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		return conn, nil
	}
}

func UDPListener() Listener {
	return func(l *net.UDPAddr) (*net.UDPConn, error) {
		conn, err := net.ListenUDP("udp", l)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
}

