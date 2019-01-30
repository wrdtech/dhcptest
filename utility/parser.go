package utility

import (
	"bytes"
	"dhcptest/layers"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type optionFormat string

const (
	 IPFormat          optionFormat = "ip"
	 HexFormat         optionFormat = "hex"
	 StringFormat      optionFormat = "string"
	 BoolForamt        optionFormat = "bool"
	 DHCPOptionFormat  optionFormat = "option"
	 DHCPMessageFormat optionFormat = "message"
	 MacFormat         optionFormat = "mac"
	 DHCPTimeFormat    optionFormat = "time"
)

type parse func(string) ([]byte, error)

type Parser struct{
	parser map[string]parse
}

func (parser *Parser) Init() {
	parser.parser = make(map[string]parse)
	parser.parser[string(IPFormat)] = parseIP
	parser.parser[string(HexFormat)] = parseHex
	parser.parser[string(StringFormat)] = parseString
	parser.parser[string(BoolForamt)] = parseBool
	parser.parser[string(DHCPOptionFormat)] = parseOption
	parser.parser[string(DHCPMessageFormat)] = parseMessage
	parser.parser[string(MacFormat)] = parseMac
	parser.parser[string(DHCPTimeFormat)] = parseTime
}

func (parser *Parser) Parse(str string) (layers.DHCPOptions, error) {
	//define variable
	var code, value string
	var format optionFormat
	var dhcpOptions layers.DHCPOptions
	buf := bytes.Buffer{}

	//define parse function
	parseOption := func() error {
		if len(code) == 0 {
			return fmt.Errorf("missing option code")
		}
		optionCode, err := strconv.Atoi(code)
		if err != nil {
			return fmt.Errorf("code parser error: %s", err)
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
			return fmt.Errorf("%s unsupport value format", format)
		}

		//parse
		data, err := valueParser(value)
		if err != nil {
			return err
		}
		//fmt.Printf("%+v\n", data)

		dhcpOption := layers.NewDHCPOption(layers.DHCPOpt(optionCode),data)
		dhcpOptions = append(dhcpOptions, dhcpOption)
		return nil
	}

	fmt.Println(str)

	hasLeftBracket := false
	hasRightBracket := false

	//start parse
	for i:=0 ; i < len(str); i++ {
		switch str[i] {
		case '=':
			if hasLeftBracket != hasRightBracket {
				return dhcpOptions, fmt.Errorf("左括号和右括号应成对出现")
			}
			if !hasLeftBracket {
				code = buf.String()
				buf.Reset()
			}
			//fmt.Println(code)
			//fmt.Println(format)
		case ';':
			err := parseOption()
			if err != nil {
				return dhcpOptions, err
			}

			hasLeftBracket = false
			hasRightBracket = false
			code = ""
			value = ""
			format = ""
			buf.Reset()
		case '[':
			hasLeftBracket = true
		    code = buf.String()
			buf.Reset()
		case ']':
			hasRightBracket = true
			format = optionFormat(buf.String())
			buf.Reset()
		default:
			buf.WriteRune(rune(str[i]))
			if i == len(str) - 1 {
				err := parseOption()
				if err != nil {
					return dhcpOptions, err
				}
			}
		}
	}
	return dhcpOptions, nil
}

func parseHex(value string) ([]byte, error) {
	data, err := hex.DecodeString(value)
	if err != nil {
		return data, err
	}
	return data, nil
}

func parseString(value string) ([]byte, error){
	data := []byte(value)
	return data, nil
}

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
		optionCode, ok := optionList[str]
		if !ok {
			return data, fmt.Errorf("%s unsupport option", str)
		}
		data = append(data, byte(optionCode))
	}
	return data, nil
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
	} else if strings.EqualFold(value, "unspecified"){
		data = append(data, byte(layers.DHCPMsgTypeUnspecified))
	} else {
		return data, fmt.Errorf("%s unsupport message type", value)
	}
	return data, nil
}

func parseIP(value string) ([]byte, error) {
	var data []byte
	ipAddrs := strings.Split(value, ",")
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

func parseMac(value string) ([]byte, error) {
	var data []byte
	macAddrs := strings.Split(value, ",")
	for _, macAddr := range macAddrs {
		mac, err := net.ParseMAC(macAddr)
		if err != nil {
			return data, fmt.Errorf("%s is not a valid mac address", macAddr)
		}
		data = append(data, []byte(mac)...)
	}
	return data, nil
}

func parseTime(value string) ([]byte, error) {
	data := make([]byte, 4)
	sec, err := time.ParseDuration(value)
	if err != nil {
		return data, err

	}
	binary.BigEndian.PutUint32(data, uint32(sec/time.Second))
	return data, nil
}
