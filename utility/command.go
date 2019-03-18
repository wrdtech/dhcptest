package utility

import (
	"dhcptest/layers"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

var (
	Help         bool
	BindIface    string
	BindMac      string
	Secs         time.Duration
	Quiet        bool
	Query        bool
	Wait         bool
	Option       RequestParams
	Request      string
	PrintOnly    string
	Timeout      time.Duration
	Try          int
	RequestIP    string
	DhcpOptions  layers.DHCPOptions
)

type CommandFlag struct {
	Name  string
	usage string
}

type Command struct {
	*CommandFlag
	Value interface{}
}

type RequestParams []string

func (r *RequestParams) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func (r *RequestParams) String() string {
	return fmt.Sprint([]string(*r))
}

func (r *RequestParams) Get() interface{} {
	return []string(*r)
}

var (
	ValidIface = make(map[string]net.Interface)
	optionRequest = RequestParams{}

	CommandHelp           = CommandFlag{Name: "help",         usage: "  --help          get a list of command-line options"}
	CommandOptionHelp     = CommandFlag{Name: "optionhelp",   usage: "  --optionhelp    get a list of dhcp option"}
	CommandIfaceList      = CommandFlag{Name: "iface-list",   usage: "  --iface-list    get a list of avaliable ip(only v4)"}
	CommandBindIface      = CommandFlag{Name: "bind",         usage: "  --bind Iface    Listen on the interface with the specified IP.\r\n\t\t  The default is to listen on all interfaces (0.0.0.0)."}
	CommandMac            = CommandFlag{Name: "mac",          usage: "  --mac MAC       Specify a MAC address to use for the client hardware\r\n\t\t  address field (chaddr), in the format NN:NN:NN:NN:NN:NN"}
	CommandSecs           = CommandFlag{Name: "secs",         usage: "  --secs          Specify the \"Secs\" request field (number of seconds elapsed\r\n\t\t  since a client began an attempt to acquire or renew a lease)"}
	CommandQuiet          = CommandFlag{Name: "quiet",        usage: "  --quiet         Suppress program output except for received data\r\n\t\t  and error messages"}
	CommandQuery          = CommandFlag{Name: "query",        usage: "  --query         Instead of starting an interactive prompt, immediately send\r\n\t\t  a discover packet, wait for a result, print it and exit."}
	CommandWait           = CommandFlag{Name: "wait",         usage: "  --wait          Wait until timeout elapsed before exiting from --query, all\r\n\t\t  offers returned will be reported."}
	CommandOption         = CommandFlag{Name: "option",       usage: "  --option OPTION Add an option to the request packet. The option must be\r\n\t\t  specified using the syntax CODE=VALUE or CODE[FORMAT]=VALUE,\r\n\t\t  where CODE is the numeric option number, FORMAT is how the\r\n\t\t  value is to be interpreted and decoded, and VALUE is the\r\n\t\t  option Value. FORMAT may be omitted for known option CODEs\r\n\t\t  E.g. to specify a Vendor Class Identifier:\r\n\t\t  --option \"60=Initech Groupware\"\r\n\t\t  You can specify hexadecimal or IPv4-formatted options using\r\n\t\t  --option \"N[hex]=...\" or --option \"N[IP]=...\"\r\n\t\t  Supported FORMAT types:\r\n\t\t  str, ip, ip, hex, i32, time, dhcpMessageType, dhcpOptionType, netbiosNodeType, relayAgent"}
	CommandRequest        = CommandFlag{Name: "request",      usage: "  --request N     Uses DHCP option 55 (\"Parameter Request List\") to\r\n\t\t  explicitly request the specified option from the server.\r\n\t\t  Can be repeated several times to request multiple options."}
	CommandPrint          = CommandFlag{Name: "print-only",   usage: "  --print-only N  Print only the specified DHCP option.\r\n\t\t  You can specify a desired format using the syntax N[FORMAT]\r\n\t\t  See above for a list of FORMATs. For example:\r\n\t\t  --print-only \"N[hex]\" or --print-only \"N[IP]\""}
	CommandTimeOut        = CommandFlag{Name: "timeout",      usage: "  --timeout N     Wait N seconds for a reply, after which retry or exit.\r\n\t\t  Default is 60 seconds. Can be a fractional number.\r\n\t\t  A Value of 0 causes dhcptest to wait indefinitely."}
	CommandTry            = CommandFlag{Name: "tries",        usage: "  --tries N       Send N DHCP discover packets after each timeout interval.\r\n\t\t  Specify N=0 to retry indefinitely."}
	CommandRequestIP      = CommandFlag{Name: "requestip",    usage: "  --requestip IP  Specify the IP Address you want to get for the client mac"}


	CommandList = []Command{
	Command{CommandFlag: &CommandHelp, Value: flag.Bool(CommandHelp.Name, false, CommandHelp.usage)},
	Command{CommandFlag: &CommandOptionHelp, Value: flag.Bool(CommandOptionHelp.Name, false, CommandOptionHelp.usage)},
	Command{CommandFlag: &CommandIfaceList, Value: flag.Bool(CommandIfaceList.Name, false, CommandIfaceList.usage)},
	Command{CommandFlag: &CommandBindIface, Value: flag.String(CommandBindIface.Name, "以太网", CommandBindIface.usage)},
	Command{CommandFlag: &CommandMac, Value: flag.String(CommandMac.Name, "", CommandMac.usage)},
	Command{CommandFlag: &CommandSecs, Value: flag.Duration(CommandSecs.Name, 10*time.Second, CommandSecs.usage)},
	Command{CommandFlag: &CommandQuiet, Value: flag.Bool(CommandQuiet.Name, false, CommandQuiet.usage)},
	Command{CommandFlag: &CommandQuery, Value: flag.Bool(CommandQuery.Name, false, CommandQuery.usage)},
	Command{CommandFlag: &CommandWait, Value: flag.Bool(CommandWait.Name, false, CommandWait.usage)},
	Command{CommandFlag: &CommandOption, Value: &optionRequest},
	Command{CommandFlag: &CommandRequest, Value: flag.String(CommandRequest.Name, "", CommandRequest.usage)},
	Command{CommandFlag: &CommandPrint, Value: flag.String(CommandPrint.Name, "", CommandPrint.usage)},
	Command{CommandFlag: &CommandTimeOut, Value: flag.Duration(CommandTimeOut.Name, 10*time.Second, CommandTimeOut.usage)},
	Command{CommandFlag: &CommandTry, Value: flag.Int(CommandTry.Name, 1, CommandTry.usage)},
	Command{CommandFlag: &CommandRequestIP, Value: flag.String(CommandRequestIP.Name, "", CommandRequestIP.usage)},
    }
)

func (c Command) Print() {
	switch c.CommandFlag {
	case &CommandHelp:
		for _, command := range CommandList {
			fmt.Println(command.CommandFlag.usage)
		}

	case &CommandOptionHelp:
		fmt.Println("dhcpoption list")
		fmt.Println("  code\tdescription")
		for i := 0; i <= 255; i++ {
			var opt = layers.DHCPOpt(byte(i))
			if opt.String() == "Unknown" {
				continue
			}
			fmt.Printf("  %d\t%s\n", byte(opt), opt)
		}

	case &CommandIfaceList:
		fmt.Println("only support ipv4 for now")
		fmt.Println("  ip\t\t网卡")
		for  name, iface := range ValidIface {
			fmt.Printf("  %s\t%s\n", name, iface.HardwareAddr)
		}

	default:
		fmt.Println(c.CommandFlag.usage)
	}
}

func init() {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		ValidIface[iface.Name] = iface
		/*
		addrs, _ := (&iface).Addrs()
		for _, addr := range addrs {
			ipAddr, _, _ := net.ParseCIDR(addr.String())
			ipAddr = ipAddr.To4()
			if ipAddr == nil {
				continue
			}
			if ipAddr.IsGlobalUnicast() {
				ValidIP[ipAddr.String()] = iface
			}
		}
		*/
	}

	flag.Var(&optionRequest, CommandOption.Name, CommandOption.usage)

	flag.Parse()

	getOpts()
}

func getOpts() {
	for _, command := range CommandList {
		switch command.CommandFlag {
		case &CommandHelp, &CommandOptionHelp, &CommandIfaceList:
			Help = *(command.Value.(*bool))
			if Help {
				command.Print()
				os.Exit(0)
			}
			break
		case &CommandBindIface:
			BindIface = *command.Value.(*string)
			break
		case &CommandMac:
			BindMac = *command.Value.(*string)
			break
		case &CommandSecs:
			Secs = *command.Value.(*time.Duration)
			break
		case &CommandQuiet:
			Quiet = *command.Value.(*bool)
			break
		case &CommandQuery:
			Query = *command.Value.(*bool)
			break
		case &CommandWait:
			Wait = *command.Value.(*bool)
			break
		case &CommandOption:
			Option = *command.Value.(*RequestParams)
			break
		case &CommandRequest:
			Request = *command.Value.(*string)
			break
		case &CommandPrint:
			PrintOnly = *command.Value.(*string)
			break
		case &CommandTimeOut:
			Timeout = *command.Value.(*time.Duration)
			break
		case &CommandTry:
			Try = *command.Value.(*int)
			break
		case &CommandRequestIP:
			RequestIP = *command.Value.(*string)
			break
		default:
			break
		}
	}
}
