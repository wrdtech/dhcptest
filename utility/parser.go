package utility

import (
	"bytes"
	"dhcptest/layers"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type optionFormat string

const (
	 IPFormat          optionFormat = "ip"
	 HexFormat         optionFormat = "hex"
	 StringFormat      optionFormat = "string"
	 BoolForamt        optionFormat = "bool"
	 DHCPOptionFormat  optionFormat = "option"
	 DHCPMessageFormat optionFormat = "message"
)

type parse func(string) ([]byte, error)

type Parser struct{
	parser map[string]parse
}

func (parser *Parser) Init() {
	parser.parser[string(IPFormat)] = parseIP
//	parser.parser[string(HexFormat)] = parseHex
//	parser.parser[string(StringFormat)] = parseString
}

func (parser *Parser) Parse(str string) (layers.DHCPOptions, error) {
	var code, value string
	var format optionFormat
	var dhcpOptions layers.DHCPOptions
	buf := bytes.Buffer{}
	for i:=0 ; i < len(str); i++ {
		switch str[i] {
		case '=':
			if len(format) == 0 {
				code = buf.String()
				buf.Reset()
			}
		case ';':
			//code
			if len(code) == 0 {
				return dhcpOptions, fmt.Errorf("missing option code")
			}
			optionCode, err := strconv.Atoi(code)
			if err != nil {
				return dhcpOptions,  fmt.Errorf("code parser error: %s", err)
			}
			//format
			if len(format) == 0 {
				format = StringFormat
			}
			//value
			value = buf.String()

			//get parser
			valueParser,ok := parser.parser[string(format)]
			if !ok {
				return dhcpOptions, fmt.Errorf("%s unsupport value format", format)
			}

			//parse
			data, err := valueParser(value)
			if err != nil {
				return dhcpOptions, err
			}

			dhcpOption := layers.NewDHCPOption(layers.DHCPOpt(optionCode),data)
			dhcpOptions = append(dhcpOptions, dhcpOption)
			buf.Reset()
		case '[':
		    code = buf.String()
			buf.Reset()
		case ']':
			format = optionFormat(buf.String())
			buf.Reset()
		default:
			buf.WriteRune(rune(str[i]))
			fmt.Println(buf.String())
		}
	}
	return dhcpOptions, nil
}

/*
func parseHex(value string) ([]byte, error) {


}

func parseString(value string) ([]byte, error){

}
*/

func parseBool(value string) ([]byte, error) {
	var data []byte
	boolean, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	if boolean {
		data = append(data, byte(1))
	} else {
		data = append(data, byte(0))
	}
	return data, nil
}

func parseOption(value string) ([]byte, error) {
	var data []byte
	params := strings.Split(value, ",")
	for _, str := range params {

		if str == "subnet mask" {
			data = append(data, byte(layers.DHCPOptSubnetMask))
		} else if str == "router" {
			data = append(data, byte(layers.DHCPOptRouter))
		} else if str == "time server" {
			data = append(data, byte(layers.DHCPOptTimeServer))
		} else if str == "domain name server" {
			data = append(data, byte(layers.DHCPOptDNS))
		} else if str == "doamin name" {
			data = append(data, byte(layers.DHCPOptDomainName))
		} else if str == "interface mtu" {
			data = append(data, byte(layers.DHCPOptInterfaceMTU))
		} else if str == "network time protocol servers" {
			data = append(data, byte(layers.DHCPOptNTPServers))
		} else {

		}
	}
}

func parseMessage(value string) ([]byte, error) {
	var data []byte
	if strings.EqualFold(value, "discover") {
		data = append(data, byte(layers.DHCPMsgTypeDiscover))
	} else if strings.EqualFold(value, "request") {
		data = append(data, byte(layers.DHCPMsgTypeRequest))
	} else if strings.EqualFold(value, "offer") {
		data = append(data, byte(layers.DHCPMsgTypeOffer))
	} else if strings.EqualFold(value, "ack") {
		data = append(data, byte(layers.DHCPMsgTypeAck))
	} else if strings.EqualFold(value, "nak") {
		data = append(data, byte(layers.DHCPMsgTypeNak))
	} else if strings.EqualFold(value, "inform") {
		data = append(data, byte(layers.DHCPMsgTypeInform))
	} else if strings.EqualFold(value, "release") {
		data = append(data, byte(layers.DHCPMsgTypeRelease))
	} else if strings.EqualFold(value, "decline") {
		data = append(data, byte(layers.DHCPMsgTypeDecline))
	} else {
		data = append(data, byte(layers.DHCPMsgTypeUnspecified))
	}
	return data, nil
}

func parseIP(value string) ([]byte, error) {
	var data []byte
	ipAddrs := strings.Split(",", value)
	for _, ipAddr := range ipAddrs {
		ip := net.ParseIP(ipAddr)
		if ip == nil {
			return data, fmt.Errorf("%s is not a vaid ip address", ipAddr)
		}
		ip = ip.To4()
		if ip == nil {
			return data, fmt.Errorf("%s is not a valid ipv4 address", ipAddr)
		}
		data = append(data, []byte(ip)...)
	}
	return data, nil
}
