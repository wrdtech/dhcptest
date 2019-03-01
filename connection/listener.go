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
	SetXid(interface{})
	Start(*net.UDPAddr, ...interface{}) error
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
	xids map[uint32]chan *layers.DHCPv4
	workSignal <-chan struct{}
	workSignalBackup <-chan struct{}
	shutdown chan int

}

func (lt *ListenThread) init(){
	lt.rwlock = new(sync.RWMutex)
	lt.shutdown = make(chan int)
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

func (lt *ListenThread) SetXid(xids interface{}) {
	lt.xids = xids.(map[uint32]chan *layers.DHCPv4)
}

func (lt *ListenThread) Start(l *net.UDPAddr, msgTypes ...interface{}) error {
	con, err := lt.Listener(l)
	if err != nil {
		return err
	}

	var dhcpMsgTypes []layers.DHCPMsgType
	for _, msgType := range msgTypes {
		dhcpMsgTypes = append(dhcpMsgTypes, msgType.(layers.DHCPMsgType))
	}

	//不能在listen函数里面设置status, 原因如下:如果发送频率高， send间隔时间短，那么第二次send去取空闲线程时该thread的状态值仍为sleeping,导致取到的实际上
	//已使用过的thread,调用两次start, 有两个listen协程，两次退出，关闭channel两次导致panic close of closed channel,并且会无法收到第一次的offer包
	//(第二次setXid时把第一次的channel给覆盖了)
	lt.rwlock.Lock()
	lt.status = Running
	lt.rwlock.Unlock()

	go lt.listen(con, dhcpMsgTypes...)
	return nil

}

func (lt *ListenThread) Stop() error {
	if lt.Status() == Sleeping {
		return fmt.Errorf("listen thread %d has already stopped\n", lt.Id)
	}

	lt.shutdown <- 1

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

func (lt *ListenThread) listen(con net.PacketConn, msgTypes ...layers.DHCPMsgType) {
	/*
	lt.rwlock.Lock()
	lt.status = Running
	lt.rwlock.Unlock()
	*/
	//fmt.Printf("listen thread %d start\n", lt.Id)
	timer := time.NewTimer(lt.Timeout)
	defer func(){
		lt.rwlock.Lock()
		lt.status = Sleeping
		lt.rwlock.Unlock()
		//fmt.Printf("listen goroutine %d stop, status: %s, time: %s\n", lt.Id, lt.Status(), time.Now())
	}()
	defer con.Close()
	defer func() {
		for _, c := range lt.xids {
			//fmt.Printf("listen thread %d, xid: %d, c:%+v, xidsLength: %d\n", lt.Id, xid, c, len(lt.xids))
			close(c)
		}
	}()

	for {

		select {
		case <-timer.C:
			return
		case <-lt.shutdown:
			return
		default:
		}

		select {
		case <-lt.workSignal:
			recvBuf := make([]byte, 342)
			con.SetReadDeadline(time.Now().Add(time.Second * 1))
			_, _, err := con.ReadFrom(recvBuf)

			if err != nil {
				continue
				//log.Printf("listen goroutine %d , read err: %s", lt.Id, err)
			}

			packet := utility.ParsePacket(recvBuf)

			if packet == nil {
				continue
			}

			if c, ok := lt.xids[packet.Xid]; ok && packet.Operation == layers.DHCPOpReply {
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
