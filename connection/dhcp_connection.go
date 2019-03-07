package connection

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"math/rand"
	"dhcptest/utility"
	"dhcptest/layers"
	"github.com/google/gopacket"
)

type Callback func(*utility.Lease)

//DhcpClient
type DhcpClient struct {
	BindIP net.IP
	ClientMac net.HardwareAddr
	Hostname string
	Iface *net.Interface
	Lease *utility.Lease
	OnBound Callback
	OnExpire Callback
	DHCPOptions []layers.DHCPOption
	Timeout time.Duration
	laddr net.UDPAddr
	logger *utility.Log
	listenThreadPool []UDPListenThread
	rebind bool
	shutdown bool
	running bool
	notify chan struct{}
	sign chan int
	wg sync.WaitGroup
}

var DefaultParamsRequestList = []layers.DHCPOpt{
	layers.DHCPOptSubnetMask,
	layers.DHCPOptRouter,
	layers.DHCPOptTimeServer,
	layers.DHCPOptDNS,
	layers.DHCPOptDomainName,
	layers.DHCPOptInterfaceMTU,
	layers.DHCPOptNTPServers,
}

func (dc *DhcpClient) AddOption(optType layers.DHCPOpt, data []byte) {
	dc.DHCPOptions  = append(dc.DHCPOptions, layers.NewDHCPOption(optType, data))
}

func (dc *DhcpClient) AddParamRequest(dhcpOpt layers.DHCPOpt) {
	for i := range dc.DHCPOptions {
		if dc.DHCPOptions[i].Type == layers.DHCPOptParamsRequest {
			dc.DHCPOptions[i].AddByte(byte(dhcpOpt))
			return
		}
	}
	dc.AddOption(layers.DHCPOptParamsRequest, []byte{byte(dhcpOpt)})
}

func (dc *DhcpClient) Start() {
	/*
	for _, param := range DefaultParamsRequestList {
		dc.AddParamRequest(param)
	}
	*/
	/*
	dc.AddOption(layers.DHCPOptHostname, []byte(dc.Hostname))

	if dc.notify != nil {
		log.Panicf("client for %s already started", dc.Iface.Name)
	}

	dc.notify = make(chan struct{})
	dc.laddr = net.UDPAddr{IP: dc.BindIP, Port: 68}
	dc.logger = &utility.Log{Logger: utility.DHCPLogger()}
	dc.wg.Add(1)
	fmt.Printf("client: %+v\n", dc)
	go dc.run()
	*/
	//dc.sign = make(chan int)
	dc.logger = &utility.Log{Logger: utility.DHCPLogger()}
	dc.laddr = net.UDPAddr{IP: dc.BindIP, Port: 68}
	dc.AddOption(layers.DHCPOptHostname, []byte(dc.Hostname))

}

func (dc *DhcpClient) Stop() {
	//goroutine开启需要消耗时间，所以在shutdown之前调用同样耗时的Print,能保证run调用时shutdown还为false,这样runonce能够顺利执行
	//log.Printf("[%s] shutting down dhcp client", dc.Iface.Name)
	//未开始之前先不调用stop
	/*
	for !dc.running {}

	close(dc.notify)
	dc.shutdown = true

	//wait all goroutines stop
	dc.wg.Wait()

	log.Printf("[%s] shutting down dhcp client", dc.Iface.Name)
	dc.sign <- 1
	dc.wg.Wait()
	*/
	dc.wg.Wait()
	log.Printf("[%s] shutting down dhcp client", dc.Iface.Name)

}

func (dc *DhcpClient) GetIdleListenThread() UDPListenThread {
	for _, lt := range dc.listenThreadPool {
		if lt.Status() == Sleeping {
			return lt
		}
	}
	return nil
}

func (dc *DhcpClient) GetAddr() *net.UDPAddr {
	return &dc.laddr
}

func (dc *DhcpClient) InitListenThread(size int) {
	dc.listenThreadPool = nil
	for i := 0; i < size; i++ {
		lt := &ListenThread{
			Id: rand.Uint32(),
			Timeout: dc.Timeout,
			Listener: UDPListener(),
		}
		lt.init()
		dc.listenThreadPool = append(dc.listenThreadPool, lt)
	}
}

func (dc *DhcpClient) StopListenThread() error {
	for _, lt := range dc.listenThreadPool{
		if lt.Status() != Sleeping {
			err := lt.Stop()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (dc *DhcpClient) SendDiscover(mac net.HardwareAddr ,  xid uint32, c chan *layers.DHCPv4, request bool) error {
	/*
	// start listen thread
	lt := dc.getIdleListenThread()
	if lt == nil {
		return fmt.Errorf("no idle listen thread now\n")
	}
	lt.SetXid(xid)
	err := lt.Start(&dc.laddr, c, layers.DHCPMsgTypeOffer)
	if err != nil {
		return err
	}
	*/

	//generate packet
	var clientMac net.HardwareAddr
	if dc.ClientMac == nil {
		clientMac = mac
	} else {
		clientMac = dc.ClientMac
	}
	packet := dc.newPacket(layers.DHCPMsgTypeDiscover, xid, clientMac, dc.DHCPOptions)
	//log.Printf("%s,sending %s:\n", dc.Iface.Name, layers.DHCPMsgTypeDiscover)
	//dc.logger.PrintLog(packet)

	//start send
	go func () {
		dc.wg.Add(1)
		defer dc.wg.Done()
		err := dc.sendMulticast(packet)
		if err != nil {
			log.Println(err)
			return
		}
		utility.DHCPCounter[clientMac.String()].AddRequest(1)

		for resPacket := range c {
			_, lease := utility.NewLease(resPacket)
			utility.DHCPCounter[clientMac.String()].AddResponse(1)
			//msgType, lease := utility.NewLease(resPacket)
			//log.Printf("[%s] received %s\n", dc.Iface.Name, msgType)
			//dc.logger.PrintLog(resPacket)
			if request {
				err = dc.sendRequest(&lease, xid, clientMac)
				if err != nil {
					log.Println(err)
				}
			}
		}

		//log.Println("discover over")

	}()

	return nil
}

func (dc *DhcpClient) sendRequest(lease *utility.Lease, xid uint32, clientMac net.HardwareAddr) error {
	// start listen thread
	lt := dc.GetIdleListenThread()
	if lt == nil {
		return fmt.Errorf("no idle listen thread now\n")
	}
	c := make(chan interface{}, 10)
	lt.SetXid(xid)
	err := lt.Start(&dc.laddr, c, layers.DHCPMsgTypeAck, layers.DHCPMsgTypeNak)
	if err != nil {
		return err
	}

	//generate packet
	fixedAddress :=  []byte(lease.FixedAddress)
	serverID := []byte(lease.ServerID)

	packet := dc.newPacket(layers.DHCPMsgTypeRequest, xid, clientMac, append(dc.DHCPOptions,
		layers.NewDHCPOption(layers.DHCPOptRequestIP, fixedAddress),
		layers.NewDHCPOption(layers.DHCPOptServerID, serverID)))
	log.Printf("%s,sending %s:\n", dc.Iface.Name, layers.DHCPMsgTypeRequest)
	dc.logger.PrintLog(packet)

	//send request
	go func() {
		defer lt.Stop()
		err = dc.sendMulticast(packet)
		if err != nil {
			log.Println(err)
			return
		}

		for resPacket := range c {
			msgType, _ := utility.NewLease(resPacket.(*layers.DHCPv4))
			log.Printf("[%s] received %s\n", dc.Iface.Name, msgType)
			dc.logger.PrintLog(resPacket)
		}

		log.Println("request over")
	}()

	return nil

}

func (dc *DhcpClient) sendMulticast(dhcp *layers.DHCPv4) error {

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths: true,
	}

	err := gopacket.SerializeLayers(buf, opts, dhcp)

	if err != nil {
		return err
	}

    con, err := UDPDialer()(&dc.laddr, &net.UDPAddr{IP:net.IPv4bcast, Port:67})
	defer con.Close()

	con.SetWriteDeadline(time.Now().Add(dc.Timeout))

	_, err = con.Write(buf.Bytes())

	return err
}

func (dc *DhcpClient) newPacket(msgType layers.DHCPMsgType, xid uint32, clientMac net.HardwareAddr, options[]layers.DHCPOption) *layers.DHCPv4 {
	packet := layers.DHCPv4{
		Operation: layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		ClientHWAddr: clientMac,
		HardwareLen: uint8(len([]byte(dc.ClientMac))),
		Flags: uint16(layers.BroadcastFlag),
		ClientIP: net.ParseIP("0.0.0.0"),
		YourClientIP: net.ParseIP("0.0.0.0"),
		NextServerIP: net.ParseIP("0.0.0.0"),
		RelayAgentIP: net.ParseIP("0.0.0.0"),
		Xid: xid,
	}

	packet.Options = append(packet.Options, layers.NewDHCPOption(layers.DHCPOptMessageType, []byte{byte(msgType)}))
	for _, option := range options {
		packet.Options = append(packet.Options, layers.NewDHCPOption(option.Type, option.Data))
	}

	return &packet
}
