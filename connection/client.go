package connection

import (
	"dhcptest/layers"
	"dhcptest/utility"
	"fmt"
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
	ifRequest bool
	ifLog     bool
	connection net.PacketConn
	laddr net.UDPAddr
	logger *utility.Log
	sendQueue chan *layers.DHCPv4
	messages chan interface{}
	packets map[uint32]*PacketResponse
	packetsLock *sync.Mutex
	stop chan int
	requestSend chan int
	requestGet chan int
	responseSend chan int
	responseGet chan int
	workers  []func()
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

func (dc *DhcpClient) Start(size int, ifRequest bool, ifLog bool) {
	dc.BufferSize = size
	dc.ifRequest = ifRequest
	dc.ifLog = ifLog
	dc.sendQueue = make(chan *layers.DHCPv4, dc.BufferSize)
	dc.messages = make(chan interface{}, dc.BufferSize)
	dc.packets = make(map[uint32]*PacketResponse)
	dc.requestSend = make(chan int, dc.BufferSize)
	dc.requestGet = make(chan int)
	dc.responseSend =  make(chan int, dc.BufferSize)
	dc.responseGet = make(chan int)
	dc.stop  = make(chan int)
	dc.workers = make([]func(), 0)
	dc.workers = append(dc.workers, dc.messageLoop)
	dc.workers = append(dc.workers, dc.listenLoop)
	dc.workers = append(dc.workers, dc.sendLoop)
	dc.wg.Add(3)
	if !dc.ifLog {
		dc.workers = append(dc.workers, dc.counter)
		dc.wg.Add(1)
	}
	dc.startWorkers()
}

func (dc *DhcpClient) startWorkers() {
	for _, worker := range dc.workers {
		go worker()
	}
}

func (dc *DhcpClient) counter() {
	log.Println("counter goroutine start")
	request, response := 0,0
	ticket := time.NewTicker(time.Second * time.Duration(5))
	defer func(){
		dc.wg.Done()
	}()
	for {
		select {
		case <-dc.stop:
			log.Println("counter goroutine stop")
			return
		default:
		}
		select {
		case amount := <-dc.requestSend:
			request = request + amount
		default:
		}
		select {
		case amount := <-dc.responseSend:
			response = response + amount
		default:
		}
		select {
		case <- ticket.C:
			dc.requestGet <- request
			dc.responseGet <- response
		default:
		}
	}
}

func (dc *DhcpClient) GetRequestAndResponse() (request int, response int) {
	select {
	case request = <- dc.requestGet:
	}
	select {
	case response = <- dc.responseGet:
	}
	return request, response
}

func (dc *DhcpClient) Stop() {
	log.Printf("[%s] shutting down dhcp client", dc.Iface.Name)
	dc.stopWorkers()
	dc.wg.Wait()
	close(dc.sendQueue)
	close(dc.messages)
	close(dc.requestSend)
	close(dc.responseSend)
	close(dc.requestGet)
	close(dc.responseGet)
	log.Printf("[%s] shutting down dhcp client over", dc.Iface.Name)

}

func (dc *DhcpClient) stopWorkers() {
	num := len(dc.workers)
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

	//request := 0
	//ticker := time.NewTicker(time.Second)

	for {
		select {
		case <- dc.stop:
			log.Println("send loop stop")
			return
		case packet := <- dc.sendQueue:

			/*速度调试 用
			request = request + 1
			select  {
			case <- ticker.C:
				log.Printf("request count from sendqueue: %d", request)
			default:

			}
			*/
			//顺序结构，对packets中某一xid的不会同时进行读写操作,貌似可以不用互斥锁.但是xid会有重复，造成对相同键值的同时读写,仍会出错

			dc.packetsLock.Lock()
			pr, ok := dc.packets[packet.Xid]
			dc.packetsLock.Unlock()
			if ok {
				if packet.MessageType() == layers.DHCPMsgTypeDiscover {
					pr.Call(NewEvent(discoverDequeue, packet))
				} else if packet.MessageType() == layers.DHCPMsgTypeRequest {
					pr.Call(NewEvent(requestDequeue, packet))
				}
				if dc.ifLog {
					dc.addMessage(packet)
				}
				err := dc.send(packet)
				dc.requestSend <- 1
				if err != nil {
					dc.addMessage(err)
				}
			} else {
				dc.addMessage(fmt.Errorf("xid %d not found in packets\n", packet.Xid))
			}
		}
	}
}

func (dc *DhcpClient) messageLoop() {
	log.Println("message loop start")
	defer func() {
		dc.wg.Done()
	}()

	for {
		select {
		case <- dc.stop:
			log.Println("message loop stop")
			return
		case message := <-dc.messages:
			dc.logger.PrintLog(message)
		}
	}
}

func (dc *DhcpClient) addMessage(message interface{}) {
	dc.messages <- message
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
				dc.addMessage(err)
				continue
			}

			packet := ParsePacket(recvBuf)

			if packet == nil {
				continue
			}
			dc.packetsLock.Lock()
			if pr, ok := dc.packets[packet.Xid];ok && packet.Operation == layers.DHCPOpReply {
				dc.responseSend <- 1
				if dc.ifLog {
					dc.addMessage(packet)
				}
				if packet.MessageType() == layers.DHCPMsgTypeOffer {
					pr.Call(NewEvent(receivedOffer, packet))
					if dc.ifRequest {
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

	pr := NewPacketResponse()
	dc.packetsLock.Lock()
	dc.packets[packet.Xid] = pr
	dc.packetsLock.Unlock()
	dc.sendQueue <- packet
	return pr
}

