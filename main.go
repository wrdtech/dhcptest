package main

import (
	"bufio"
	"dhcptest/connection"
	"dhcptest/layers"
	"dhcptest/utility"
	"fmt"
	"github.com/pinterest/bender"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	intervalC chan int
	loggerC chan int
)

func init() {
	intervalC = make(chan int)
	loggerC = make(chan int)
	fmt.Println("dhcptest v0.1 -Created by WRD, based on gopacket")
	fmt.Println("Run with --help for a list of command-line options")
}

func main() {

	//bind ip
	iface, err :=utility.GetInterfaceByIP(utility.BindIP, utility.ValidIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	/*
	//bind mac
	clientMac, err := net.ParseMAC(utility.BindMac)
	if err != nil {
		fmt.Println(err)
		return
	}
	*/

	//option
	parser := &utility.Parser{}
	parser.Init()
	utility.DhcpOptions, err =  parser.Parse(utility.Option)
	if err != nil {
		fmt.Println(err)
		return
	}

	//dhcpOptions = append(dhcpOptions, layers.NewDHCPOption(layers.DHCPOptRequestIP, requestIPByte), layers.NewDHCPOption(layers.DHCPOptClientID, clientID))
	//dhcpOptions = append(dhcpOptions, layers.NewDHCPOption(layers.DHCPOptClientID, clientID))

	fmt.Printf("dhcpOptions: %+v\n", utility.DhcpOptions)
	/*
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		return
	}
	*/

	dc := &connection.DhcpClient{
		BindIP:    net.ParseIP(utility.BindIP),
		//ClientMac: clientMac,
		Iface:     iface,
		Raddr:     net.UDPAddr{IP:net.IPv4bcast, Port: 67},
	}
	dc.Open()
	defer dc.Close()

	inputReader := bufio.NewReader(os.Stdin)
	fmt.Println("Type \"d\" to broadcast a DHCP discover packet, or \"help\" for details")
	for {
		input, err := inputReader.ReadString('\n')
		if err != nil {
			log.Println(err)
		}
		input = strings.TrimSpace(input)
		params := strings.Split(input, " ")
		//fmt.Printf("str: %+v", params)
		if len(params) == 0 || len(params[0]) == 0 {
			continue
		}
		command := params[0]
		switch command {
		case "q", "quit":
			return
		case "h", "help":
			fmt.Println("Commands:")
			fmt.Printf("\t d / discover\n" +
				"\t\t Broadcasts a DHCP discover packet.\n" +
				"\t\t You can optionally specify the device num and request rate\n" +
				"\t\t to use for the throughput testing, e.g.\n" +
				"\t\t \"d 5 100\" will pretend 5 terminals and request\n" +
				"\t\t 100 times per second.\n" +
				"\t\t You can also only specify the device num to use for one-time request\n" +
				"\t\t dhcp packet message will be printed.The default value is 1 when the device num is omitted\n")
			fmt.Printf("\t r / request\n" +
				"\t\t Broadcast a DHCP discover.Then broadcast a DHCP request packet when you gen an offer packet.\n" +
				"\t\t You can also specify parameters as d command does.\n")
			fmt.Printf("\t s / stop used for stop the dhcp client\n")
			fmt.Printf("\t h / help\n" +
				"\t\t Print this message.\n")
			fmt.Printf("\t q / quit\n" +
				"\t\t Quits the program\n")
		case "d", "discover":
			err = sendDHCP(params, dc, false)
			if err != nil {
				log.Println(err)
			}
		case "r", "request":
			err = sendDHCP(params, dc, true)
			if err != nil {
				log.Println(err)
			}
		case "s", "stop":
			intervalC <- 1
			loggerC <- 1
		default:
			fmt.Println("Enter a supported command, Type \"help\" for details")
		}
	}
}

func sendDHCP(params []string, dc *connection.DhcpClient, ifRequest bool) error {
	//init deviceNum
	deviceNum := 1
	if len(params) >= 2 {
		var err error
		deviceNum, err = strconv.Atoi(params[1])
		if err != nil {
			return err
		}
	}
	fmt.Printf("the %d device mac is: ", deviceNum)
	var macList []net.HardwareAddr
	for i:=0; i< deviceNum; i++ {
		mac, err := net.ParseMAC(utility.RandomMac())
		time.Sleep(time.Millisecond) //only ensure time seed different
		if err != nil {
			return err
		}
		macList = append(macList, mac)
		fmt.Printf("%s ", mac)
	}
	fmt.Println()


	//init rate
	if len(params) == 3 {
		rate, err := strconv.Atoi(params[2])
		if err != nil {
			return err
		}
		//send func
		go func() {
			/* cpu  or memory test
			fc, err := os.OpenFile("./cpu.prof", os.O_RDWR | os.O_CREATE, 0644)

			if err != nil {
				log.Fatal(err)
			}

			defer fc.Close()
			pprof.StartCPUProfile(fc)
			defer pprof.StopCPUProfile()

			fm, err := os.OpenFile("./memory.prof", os.O_RDWR | os.O_CREATE, 0644)

			if err != nil {
				log.Fatal(err)
			}

			defer fm.Close()
			pprof.WriteHeapProfile(fm)
			*/

			dc.Start(rate * 3, ifRequest, false)
			defer dc.Stop()
			requests := make(chan interface{}, rate * 3)
			ticker := time.NewTicker(time.Second)
			//countTicker测试用
			//countTicker := time.NewTicker(time.Second)
			//windows下的ticker精度是ms级(本次测试2ms) 所以在1s的ticker内循环rate次
			go func() {
				defer close(requests)
				index := 0
				//request := 0
				for {
					select {
					case <-ticker.C:
						for i:=0; i < rate; i++ {
							mac := macList[index]
							packet := connection.NewPacket(utility.DhcpOptions...)
							connection.WithHWType(layers.LinkTypeEthernet)(packet)
							connection.WithHwAddr(mac)(packet)
							connection.WithMessageType(layers.DHCPMsgTypeDiscover)(packet)
							requests <- packet
							//request = request + 1
							if index = index +1 ; index == deviceNum {
								index = 0
							}
						}
					case <-intervalC:
						log.Println("ticker stop")
						return
					}
					/*
					select {
					case <-countTicker.C:
						log.Printf("produce request count : %d\n", request)
					default:
					}
					*/
				}
			}()
			intervals := bender.ExponentialIntervalGenerator(float64(rate))
			exec := connection.CreateExecutor(dc)
			bender.LoadTestThroughput(intervals, requests, exec)
		}()
		//低延迟读取 不要使用共享数据来通信；使用通信来共享数据
		go func() {
			ticker := time.NewTicker(time.Second * 5)
			current_time := time.Now()
			for {
				select {
				case <-ticker.C:
					request, response := dc.GetRequestAndResponse()
					now_time := time.Now()
					during := int(now_time.Sub(current_time).Seconds())
					log.Printf("request: %d, response: %d, during: %d, qSpeed: %d, pSpeed: %d", request, response, during, request / during, response/during)
				case <-loggerC:
					log.Println("logger stop")
					return
				}
			}
		}()
	} else {
		go func() {
			dc.Start(deviceNum, ifRequest, true)
			defer dc.Stop()
			for i:=0; i < deviceNum; i++ {
				mac := macList[i]
				packet := connection.NewPacket(utility.DhcpOptions...)
				connection.WithHWType(layers.LinkTypeEthernet)(packet)
				connection.WithHwAddr(mac)(packet)
				connection.WithMessageType(layers.DHCPMsgTypeDiscover)(packet)
				dc.Send(packet, connection.WithTransactionID(rand.Uint32()))
			}
			select {
			case <- intervalC:
				<-loggerC
				return
			}
		}()
	}
	return nil
}

