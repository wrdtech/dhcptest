package main

import (
	"dhcptest/connection"
	"dhcptest/utility"
	"fmt"
	"log"
	"net"
	"os"
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
	option       string
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

	//bindMac
	clientID, err := net.ParseMAC(bindMac)
	if err != nil {
		fmt.Println("wrong mac format")
		return
	}

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

	//option
	parser := &utility.Parser{}
	parser.Init()
	dhcpOptions,err := parser.Parse(option)
	if err != nil {
		fmt.Println(err)
	}

	//dhcpOptions = append(dhcpOptions, layers.NewDHCPOption(layers.DHCPOptRequestIP, requestIPByte), layers.NewDHCPOption(layers.DHCPOptClientID, clientID))
	//dhcpOptions = append(dhcpOptions, layers.NewDHCPOption(layers.DHCPOptClientID, clientID))

	//fmt.Printf("dhcpOptions: %+v\n", dhcpOptions)

	//classID := []byte("MSFT 5.0")

	hostname, _ := os.Hostname()

	connection.OnlyDiscover  = onlyDiscover

	dc := connection.DhcpClient{
		BindIP: net.ParseIP(bindIP),
		Hostname: hostname,
		ClientMac: clientID,
		Iface: iface,
		OnBound: func(lease *utility.Lease) {
			log.Printf("Bound: %+v", lease)
		},
		DHCPOptions: dhcpOptions,
	}

	dc.Start()
	defer dc.Stop()
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
			if bindMac == "" {
				bindMac = utility.RandomMac()
			}
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
			option = *command.Value.(*string)
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
