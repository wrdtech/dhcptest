package main

import (
	"bufio"
	"dhcptest/connection"
	"dhcptest/layers"
	"dhcptest/utility"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	help         bool
	bindIP       string
	bindMac      string
	secs         time.Duration
	quiet        bool
	query        bool
	wait         bool
	option       utility.RequestParams
	request      string
	printOnly    string
	timeout      time.Duration
	try          int
	requestIP    string
	onlyDiscover bool
)

func init() {
	fmt.Println("dhcptest v0.1 -Created by WRD, based on gopacket")
	fmt.Println("Run with --help for a list of command-line options")
}

func main() {
	getOpts()

	//bind ip
	iface, err :=utility.GetInterfaceByIP(bindIP, utility.ValidIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	//bind mac
	clientMac, _ := net.ParseMAC(bindMac)

	/*
	//requestIP
	requestIPByte := net.ParseIP(requestIP)
	if requestIPByte == nil {
		fmt.Println("wrong ip format")
		return
	}
	requestIPByte = requestIPByte.To4()
	if requestIPByte == nil {
		fmt.Println("only support ipv4 for now")
		return
	}
	*/
//	fmt.Printf("raw option: %s\n", option)

	//option
	parser := &utility.Parser{}
	parser.Init()
	dhcpOptions,err := parser.Parse(option)
	if err != nil {
		fmt.Println(err)
	}

	//dhcpOptions = append(dhcpOptions, layers.NewDHCPOption(layers.DHCPOptRequestIP, requestIPByte), layers.NewDHCPOption(layers.DHCPOptClientID, clientID))
	//dhcpOptions = append(dhcpOptions, layers.NewDHCPOption(layers.DHCPOptClientID, clientID))

	fmt.Printf("dhcpOptions: %+v\n", dhcpOptions)

	hostname, _ := os.Hostname()

	connection.OnlyDiscover  = onlyDiscover

	dc := connection.DhcpClient{
		BindIP:    net.ParseIP(bindIP),
		Hostname:  hostname,
		ClientMac: clientMac,
		Iface:     iface,
		OnBound: func(lease *utility.Lease) {
			log.Printf("Bound: %+v", lease)
		},
		DHCPOptions: dhcpOptions,
		Timeout: timeout,
	}

	dc.Start()
	defer dc.Stop()
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


			//send func
			send := func() {
				var xids []uint32
				xidChan := make(map[uint32]chan *layers.DHCPv4)
				for i:=0 ;i <deviceNum; i++ {
					xid := rand.Uint32()
					xids = append(xids, xid)
					xidChan[xid] = make(chan *layers.DHCPv4)
				}
				lt := dc.GetIdleListenThread()
				if lt == nil {
					log.Println("no idle listen thread now")
					return
				}
				lt.SetXid(xidChan)
				err := lt.Start(dc.GetAddr(),layers.DHCPMsgTypeOffer)
				if err != nil {
					log.Println(err)
					return
				}
				for i := 0; i< deviceNum; i++ {
					xid := xids[i]
					if command == "r" || command == "request" {
						err = dc.SendDiscover(macList[i], xid, xidChan[xid], true)
					} else if command == "d" || command == "discover" {
						err = dc.SendDiscover(macList[i], xid, xidChan[xid], false)
					}
					if err != nil {
						log.Println(err)
					}
				}
			}

			//init rate
			if len(params) == 3 {
				rate, err := strconv.Atoi(params[2])
				if err != nil {
					log.Println(err)
					continue
				}
				ratePerDecivce := 60 * 1000 * 1000 * 1000 / (rate / deviceNum)
				interval := time.Nanosecond * time.Duration(ratePerDecivce)
				listenThreadSize := int((timeout / interval) * 2)
				if listenThreadSize <= 0 {
					listenThreadSize = 1
				}
				log.Printf("device num: %d, rate: %d, interval: %s, listen thread size: %d", deviceNum, rate, interval, listenThreadSize)
				ticker := time.NewTicker(interval)
				dc.InitListenThread(listenThreadSize)
				go func() {
					send()
					for range ticker.C {
						send()
					}
				}()
				go func() {
					ticker := time.NewTicker(time.Second)
					for {
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
						}

					}
				}()
			} else {
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
					}
				}()
			}
		case "s":
		default:
			fmt.Println("Enter a supported command, Type \"help\" for details")
		}
	}
}

func getOpts() {
	for _, command := range utility.CommandList {
		switch command.CommandFlag {
		case &utility.CommandHelp, &utility.CommandOptionHelp, &utility.CommandIPList:
			help = *(command.Value.(*bool))
			if help {
				command.Print()
				os.Exit(0)
			}
			break
		case &utility.CommandBindIP:
			bindIP = *command.Value.(*string)
			break
		case &utility.CommandMac:
			bindMac = *command.Value.(*string)
			break
		case &utility.CommandSecs:
			secs = *command.Value.(*time.Duration)
			break
		case &utility.CommandQuiet:
			quiet = *command.Value.(*bool)
			break
		case &utility.CommandQuery:
			query = *command.Value.(*bool)
			break
		case &utility.CommandWait:
			wait = *command.Value.(*bool)
			break
		case &utility.CommandOption:
			option = *command.Value.(*utility.RequestParams)
			break
		case &utility.CommandRequest:
			request = *command.Value.(*string)
			break
		case &utility.CommandPrint:
			printOnly = *command.Value.(*string)
			break
		case &utility.CommandTimeOut:
			timeout = *command.Value.(*time.Duration)
			break
		case &utility.CommandTry:
			try = *command.Value.(*int)
			break
		case &utility.CommandRequestIP:
			requestIP = *command.Value.(*string)
			break
		case &utility.CommandOnlyDisover:
			onlyDiscover = *command.Value.(*bool)
			break
		default:
			break
		}
	}
}
