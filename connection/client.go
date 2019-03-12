package connection

import (
	"dhcptest/layers"
	"dhcptest/utility"
	"github.com/google/gopacket"
	"log"
	"net"
	"sync"
	"time"
)

type Callback func(*Lease)

//DhcpClient
type DhcpClient struct {
	BindIP net.IP
	//ClientMac net.HardwareAddr
	Iface *net.Interface
	BufferSize int
	Raddr net.UDPAddr
	IfRequest bool
	connection net.PacketConn
	laddr net.UDPAddr
	logger *utility.Log
	sendQueue chan *layers.DHCPv4
	errors chan error
	packets map[uint32]*PacketResponse
	packetsLock *sync.Mutex
	stop chan int
	wg *sync.WaitGroup
}

func (dc *DhcpClient) Open() error {
	dc.packetsLock = new(sync.Mutex)
	dc.wg = new(sync.WaitGroup)
	dc.logger = &utility.Log{Logger: utility.DHCPLogger()}
	dc.laddr = net.UDPAddr{IP: dc.BindIP, Port: 68}
	var err error
	dc.connection, err = UDPListener()(&dc.laddr)
	if err != nil {
		return err
	}
	return nil

}

func (dc *DhcpClient) Close() error {
	err := dc.connection.Close()
	if err != nil {
		return err
	}
	return nil
}

func (dc *DhcpClient) Start(size int, ifRequest bool) {
	dc.BufferSize = size
	dc.IfRequest = ifRequest
	dc.sendQueue = make(chan *layers.DHCPv4, dc.BufferSize)
	dc.errors = make(chan error, dc.BufferSize)
	dc.packets = make(map[uint32]*PacketResponse)
	dc.stop  = make(chan int)
	dc.wg.Add(3)
	go dc.errorLoop()
	go dc.listenLoop()
	go dc.sendLoop()
}

func (dc *DhcpClient) Stop() {
	log.Printf("[%s] shutting down dhcp client", dc.Iface.Name)
	dc.done(3)
	dc.wg.Wait()
	close(dc.sendQueue)
	close(dc.errors)
	log.Printf("[%s] shutting down dhcp client over", dc.Iface.Name)

}

func (dc *DhcpClient) done(num int) {
	for i := 0; i < num; i++ {
		dc.stop <- 1
	}
}

func (dc *DhcpClient) send(packet *layers.DHCPv4) error {

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths: true,
	}

	err := gopacket.SerializeLayers(buf, opts, packet)

	if err != nil {
		return err
	}

	dc.connection.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
	_, err = dc.connection.WriteTo(buf.Bytes(), &dc.Raddr)

	return err
}


func (dc *DhcpClient) sendLoop() {
	log.Println("send loop start")
	defer func(){
		dc.wg.Done()
	}()

	for {
		select {
		case <- dc.stop:
			log.Println("send loop stop")
			return
		case packet := <- dc.sendQueue:
			//utility.DHCPCounter[packet.ClientHWAddr.String()].AddRequest(1)
			dc.packetsLock.Lock()
			pr, ok := dc.packets[packet.Xid]
			dc.packetsLock.Unlock()
			if ok {
				if packet.MessageType() == layers.DHCPMsgTypeDiscover {
					pr.Call(NewEvent(discoverDequeue, packet))
				} else if packet.MessageType() == layers.DHCPMsgTypeRequest {
					pr.Call(NewEvent(requestDequeue, packet))
				}
				//log.Printf("get packet %s\n", packet)
				err := dc.send(packet)
				if err != nil {
					dc.errors <- err
				}
			}
		}
	}
}

func (dc *DhcpClient) errorLoop() {
	log.Println("error loop start")
	defer func() {
		dc.wg.Done()
	}()

	for {
		select {
		case <- dc.stop:
			log.Println("error loop stop")
			return
		case err := <-dc.errors:
			log.Println(err)
		}
	}
}

func (dc *DhcpClient) addError(err error) {
	dc.errors <- err
}

func (dc *DhcpClient) listenLoop() {
	log.Println("listen loop start")
	defer func() {
		dc.wg.Done()
	}()

	for {
		select {
		case <- dc.stop:
			log.Println("listen loop stop")
			return
		default:
			recvBuf := make([]byte, MAXUDPReceivedPacketSize)
			dc.connection.SetReadDeadline(time.Now().Add(DefaultReadTimeout))
			_, _, err := dc.connection.ReadFrom(recvBuf)

			if err != nil {
				dc.addError(err)
				continue
			}

			packet := ParsePacket(recvBuf)

			if packet == nil {
				continue
			}
			dc.packetsLock.Lock()
			if pr, ok := dc.packets[packet.Xid];ok && packet.Operation == layers.DHCPOpReply {
				//utility.DHCPCounter[packet.ClientHWAddr.String()].AddResponse(1)
				if packet.MessageType() == layers.DHCPMsgTypeOffer {
					pr.Call(NewEvent(receivedOffer, packet))
					if dc.IfRequest {
						dc.sendQueue <- NewRequestFromOffer(packet)
					}
				} else if packet.MessageType() == layers.DHCPMsgTypeAck {
					pr.Call(NewEvent(receivedAck, packet))
				} else if packet.MessageType() == layers.DHCPMsgTypeNak {
					pr.Call(NewEvent(receivedNak, packet))
				}

			}
			dc.packetsLock.Unlock()
		}
	}
}

func (dc *DhcpClient) GetAddr() *net.UDPAddr {
	return &dc.laddr
}


func (dc *DhcpClient) Send(packet *layers.DHCPv4, modifiers ...Modifier) *PacketResponse {
	for _, modifier := range modifiers {
		modifier(packet)
	}

	pr := NewPacketResponse(dc.BufferSize)
	dc.packetsLock.Lock()
	dc.packets[packet.Xid] = pr
	dc.packetsLock.Unlock()
	dc.sendQueue <- packet
	//log.Printf("sendqueue enter %s\n", packet)
	return pr
}

