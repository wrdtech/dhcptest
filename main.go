package main

import (
	"bufio"
	"dhcptest/connection"
	"dhcptest/layers"
	"dhcptest/utility"
	"fmt"
	"github.com/pinterest/bender"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
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

	dc := connection.DhcpClient{
		BindIP:    net.ParseIP(utility.BindIP),
		//ClientMac: clientMac,
		Iface:     iface,
		Raddr:     net.UDPAddr{IP:net.IPv4bcast, Port: 67},
	}
	dc.Open()
	defer dc.Close()

	intervalC := make(chan int)
	loggerC := make(chan int)
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
				"\t\t to use for the stress testing, e.g.\n" +
				"\t\t \"d 5 100\" will pretend 5 terminals and request\n" +
				"\t\t 100 times per minute.\n")
			fmt.Printf("\t r / request\n" +
				"\t\t Broadcast a DHCP discover.Then broadcast a DHCP request packet when you gen an offer packet.\n" +
				"\t\t You can also specify parameters as d command does.\n")
			fmt.Printf("\t h / help\n" +
				"\t\t Print this message.\n")
			fmt.Printf("\t q / quit\n" +
				"\t\t Quits the program\n")
		case "d", "discover", "r", "request":
			utility.DHCPCounter = make(map[string]*utility.Counter)
			//init deviceNum
			deviceNum := 1
			if len(params) >= 2 {
				var err error
				deviceNum, err = strconv.Atoi(params[1])
				if err != nil {
					log.Println(err)
					continue
				}
			}
			fmt.Printf("the %d device mac is: ", deviceNum)
			var macList []net.HardwareAddr
			for i:=0; i< deviceNum; i++ {
				mac, err := net.ParseMAC(utility.RandomMac())
				time.Sleep(time.Millisecond) //only ensure time seed different
				if err != nil {
					log.Println(err)
					continue
				}
				macList = append(macList, mac)
				counter := new(utility.Counter)
				counter.Init()
				utility.DHCPCounter[mac.String()] = counter
				fmt.Printf("%s ", mac)
			}
			fmt.Println()


			//init rate
			if len(params) == 3 {
				rate, err := strconv.Atoi(params[2])
				if err != nil {
					log.Println(err)
					continue
				}
				//send func
				go func() {
					dc.Start(rate * 10, false)
					defer dc.Stop()
					requests := make(chan interface{}, rate * 10)
					ticker := time.NewTicker(time.Microsecond * time.Duration(1000 * 1000 / rate))
					fmt.Printf("rate: %d qps, interval: %s\n", rate,time.Microsecond * time.Duration(1000 * 1000 / rate))
					go func() {
						defer close(requests)
						index := 0
						for {
							select {
							case <-ticker.C:
								mac := macList[index]
								packet := connection.NewPacket(utility.DhcpOptions...)
								connection.WithHWType(layers.LinkTypeEthernet)(packet)
								connection.WithHwAddr(mac)(packet)
								connection.WithMessageType(layers.DHCPMsgTypeDiscover)(packet)
								//log.Printf("generate request %s\n", packet)
								requests <- packet
								if index = index +1 ; index == deviceNum {
									index = 0
							    }
							case <-intervalC:
								log.Println("ticker stop")
								return
							}
						}
					}()
					intervals := bender.UniformIntervalGenerator(float64(rate))
					exec := connection.CreateExecutor(&dc)
					recorder := make(chan interface{}, rate * 10)
					bender.LoadTestThroughput(intervals, requests, exec, recorder)
					//l := log.New(os.Stdout, "", log.LstdFlags)
					//h := hist.NewHistogram(60000, int(time.Microsecond))
					//bender.Record(recorder, bender.NewLoggingRecorder(l), bender.NewHistogramRecorder(h))
					bender.Record(recorder)
					//fmt.Println(h)
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
							/*
							request, response := 0, 0
							for _, mac := range macList {
								if counter, ok := utility.DHCPCounter[mac.String()]; ok {
									//log.Printf("mac: %s,request: %d, response: %d\n", mac, counter.GetRequest(), counter.GetResponse())
									time.Sleep(time.Millisecond * time.Duration(100))
									request = request + counter.GetRequest()
									response = response + counter.GetResponse()
									log.Printf("mac: %s,request: %d, response: %d\n", mac, counter.GetRequest(), counter.GetResponse())
								}
							}
							now_time := time.Now()
							during := int(now_time.Sub(current_time).Seconds())
							log.Printf("request: %d, response: %d, during: %d, qSpeed: %d, pSpeed: %d", request, response, during, request / during,
								response/during)
							*/
						case <-loggerC:
							log.Println("logger stop")
							return
						}
					}
				}()
			} else {
				/*
				dc.InitListenThread(1)
				send()
				go func() {
					ticker := time.NewTimer(timeout)
					select {
					case <-ticker.C:
						for _, mac := range macList {
							if counter, ok := utility.DHCPCounter[mac.String()]; ok {
								log.Printf("mac:%s, request: %d, response: %d, percentage: %s",
									mac,
									counter.GetRequest(),
									counter.GetResponse(),
									counter.GetPercentage())
							}
						}
				    case <-loggerC:
						log.Println("logger stop")
						return
					}
				}()
				*/
			}
		case "s":
			intervalC <- 1
			loggerC <- 1
		default:
			fmt.Println("Enter a supported command, Type \"help\" for details")
		}
	}
}

