package connection

import (
	"dhcptest/layers"
	"dhcptest/utility"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type UDPListenThread interface {
	Status() ListenStatus
	SetXid(uint32)
	Start(*net.UDPAddr, chan interface{}, ...interface{}) error
	Stop() error
	Pause() error
	Resume() error

}

type ListenStatus byte

var (
	Running  ListenStatus = 1
	Sleeping ListenStatus = 0
	Pausing  ListenStatus = 2
)

func (ls ListenStatus) String() string {
	switch ls {
	case Running:
		return "running"
	case Sleeping:
		return "sleeping"
	case Pausing:
		return "pausing"
	default:
		return "unknown status"
	}
}

type ListenThread struct {
	Id  uint32
	Timeout time.Duration
	Listener Listener
	status ListenStatus
	rwlock *sync.RWMutex
	xid uint32
	workSignal <-chan struct{}
	workSignalBackup <-chan struct{}

}

func (lt *ListenThread) init(){
	lt.rwlock = new(sync.RWMutex)
	ch := make(chan struct{})
	defer close(ch)

	lt.workSignal = ch
	lt.workSignalBackup = ch
}

func (lt *ListenThread) Status() ListenStatus {
	lt.rwlock.RLock()
	status := lt.status
	lt.rwlock.RUnlock()
	return status
}

func (lt *ListenThread) SetXid(xid uint32) {
	lt.xid = xid
}

func (lt *ListenThread) Start(l *net.UDPAddr, c chan interface{}, msgTypes ...interface{}) error {
	con, err := lt.Listener(l)
	if err != nil {
		return err
	}

	var dhcpMsgTypes []layers.DHCPMsgType
	for _, msgType := range msgTypes {
		dhcpMsgTypes = append(dhcpMsgTypes, msgType.(layers.DHCPMsgType))
	}

	go lt.listen(con, c, dhcpMsgTypes...)
	lt.rwlock.Lock()
	lt.status = Running
	lt.rwlock.Unlock()
	return nil

}

func (lt *ListenThread) Stop() error {
	if lt.Status() == Sleeping {
		return fmt.Errorf("listen thread %d has already stopped\n", lt.Id)
	}

	lt.rwlock.Lock()
    lt.status = Sleeping
	lt.rwlock.Unlock()

    //log.Printf("listen thread %d stopped\n", lt.Id)
    return nil
}

func (lt *ListenThread) Pause() error {
	if lt.Status() == Running {
		lt.workSignal = nil

		lt.rwlock.Lock()
		lt.status = Pausing
		lt.rwlock.Unlock()

		log.Printf("listen thread %d paused\n", lt.Id)
	} else {
		return fmt.Errorf("Can't pause listen thread %d, status: %s\n", lt.Id, lt.status)
	}
	return nil
}

func (lt *ListenThread) Resume() error {
	if lt.Status() == Pausing {
		lt.workSignal = lt.workSignalBackup

		lt.rwlock.Lock()
		lt.status = Running
		lt.rwlock.Unlock()

		log.Printf("listen thread %d resumed\n", lt.Id)
	} else {
		return fmt.Errorf("Can't resume listen thread %d, status: %s\n", lt.Id, lt.status)
	}
	return nil
}

func (lt *ListenThread) listen(con net.PacketConn, c chan interface{}, msgTypes ...layers.DHCPMsgType) {
	timer := time.NewTimer(lt.Timeout)
	fmt.Printf("listen thread %d start\n", lt.Id)
	defer con.Close()
	defer close(c)

	for {

		select {
		case <-timer.C:
			fmt.Printf("listen goroutine %d exit\n", lt.Id)
			return
		default:
		}

		select {
		case <-lt.workSignal:
			if lt.Status() == Sleeping {
				fmt.Printf("listen goroutine %d stop\n", lt.Id)
				return
			}
			recvBuf := make([]byte, 342)
			con.SetReadDeadline(time.Now().Add(time.Second * 2))
			_, _, err := con.ReadFrom(recvBuf)

			if err != nil {
				continue
				//log.Printf("listen goroutine %d , read err: %s", lt.Id, err)
			}

			packet := utility.ParsePacket(recvBuf)

			if packet == nil {
				continue
			}

			if packet.Xid == lt.xid && packet.Operation == layers.DHCPOpReply{
				t, _ := utility.NewLease(packet)
				for _, msgType := range msgTypes {
					if t == msgType {
						c <- packet
					}
				}
			}
		default:
			continue
		}

	}
}
