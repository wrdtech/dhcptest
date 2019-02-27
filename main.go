package main

import (
	"bufio"
	"dhcptest/connection"
	"dhcptest/utility"
	"fmt"
	"log"
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
		ListenThreadPoolSize: 20,
		Timeout: time.Second * 10,
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
				"\t\t You can optionally specify a part or an entire MAC address\n" +
				"\t\t to use for the client hardware address field (chaddr), e.g.\n" +
				"\t\t \"d 02:00:00\" will use the specified first 3 octets and\n" +
				"\t\t randomly generate the rest.\n")
			fmt.Printf("\t r / request\n" +
				"\t\t Broadcast a DHCP discover.Then broadcast a DHCP request packet when you gen an offer packet.\n" +
				"\t\t You can also specify parameters as d command does.\n")
			fmt.Printf("\t h / help\n" +
				"\t\t Print this message.\n")
			fmt.Printf("\t q / quit\n" +
				"\t\t Quits the program\n")
		case "d", "discover":
			num := 1
			if len(params) == 2 {
				num, err = strconv.Atoi(params[1])
				if err != nil {
					log.Println(err)
					continue
				}
			}
			for i := 0; i< num; i++ {
				err := dc.SendDiscover(false)
				if err != nil {
					log.Println(err)
				}
			}
		case "r", "request":
			err := dc.SendDiscover(true)
			if err != nil {
				log.Println(err)
			}
		case "l":
			/*
			if params == "start" {
				for _, lt := range listenThreads {
					c := make(chan interface{}, 10)
					err := lt.Start(&net.UDPAddr{IP:net.ParseIP("10.123.11.124"), Port:68}, c)
					if err != nil {
						fmt.Println(err)
					}
				}
			}

			if params == "stop" {
				err := listenThreads[num].Stop()
				if err != nil {
					fmt.Println(err)
				}
			}

			if params == "pause" {
				err := listenThreads[num].Pause()
				if err != nil {
					fmt.Println(err)
				}
			}

			if params == "resume" {
				err := listenThreads[num].Resume()
				if err != nil {
					fmt.Println(err)
				}
			}
			*/
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
