package connection

import (
	"dhcptest/layers"
	"fmt"
	"github.com/libp2p/go-reuseport"
	"github.com/pinterest/bender"
	"math/rand"
	"net"
)

type Dialer func(*net.UDPAddr, *net.UDPAddr) (net.Conn, error)

type Listener func(net.Addr) (net.PacketConn, error)

type TransPort struct {
	Dialer Dialer
	Listener Listener
}

func (t *TransPort) Dial(l *net.UDPAddr, r *net.UDPAddr) (net.Conn, error) {
	return t.Dialer(l,r)
}

func (t *TransPort) Listen(l net.Addr) (net.PacketConn, error) {
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
	return func(addr net.Addr) (net.PacketConn, error) {
		packetConn , err := reuseport.ListenPacket("udp",addr.String())
		if err != nil {
			return nil, err
		}
		return packetConn, nil
	}
}


// CreateExecutor creates a new DHCPv4 RequestExecutor.
func CreateExecutor(client *DhcpClient) bender.RequestExecutor {
	send := newSendFunc(client)
	return func(_ int64, request interface{}) (interface{}, error) {
		packet, ok := request.(*layers.DHCPv4)
		if !ok {
			return nil, fmt.Errorf("invalid request type %T, want: *layers.DHCPv4", request)
		}
		packetResonse := send(packet)
		return  packetResonse, nil
	}
}

// sendFunc represents a function used to send dhcpv4 datagrams
type sendFunc = func(*layers.DHCPv4) *PacketResponse

// newSendFunc creates a function which will send messages using the given client
func newSendFunc(client *DhcpClient) sendFunc {

	// send a message and check that the response is of a given type
	return func(d *layers.DHCPv4) *PacketResponse{
		//log.Printf("sendqueue %s\n", d)
		return client.Send(d, WithTransactionID(rand.Uint32()))
	}
}
