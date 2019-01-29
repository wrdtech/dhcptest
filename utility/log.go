package utility

import (
	"dhcptest/layers"
	"fmt"
)

type Logger func(interface{})

type Log struct {
	Logger Logger
}

func(l *Log) PrintLog(a interface{}) {
	l.Logger(a)
}

func DHCPLogger() Logger {
	return func(packet interface{}) {
		dhcpPacket := packet.(*layers.DHCPv4)
		//fmt.Printf("%+v\n", dhcpPacket)
		fmt.Printf("  op=%s  chaddr=%s  hops=%d  xid=%x  secs=%d  flags=%s\n", dhcpPacket.Operation, dhcpPacket.ClientHWAddr, dhcpPacket.HardwareOpts, dhcpPacket.Xid, dhcpPacket.Secs, layers.BootpFlag(dhcpPacket.Flags))
		fmt.Printf("  ciaddr=%s  yiaddr=%s  siaddr=%s  giaddr=%s  sname=%s file=%s\n", dhcpPacket.ClientIP, dhcpPacket.YourClientIP, dhcpPacket.NextServerIP, dhcpPacket.RelayAgentIP,
			getFileString(dhcpPacket.ServerName),getFileString(dhcpPacket.File))
		dhcpOptions := dhcpPacket.Options
		fmt.Printf("  %d options:\n", len(dhcpOptions))
		for _, option := range dhcpOptions {
			fmt.Printf("     %s\n", option)
		}
	}
}
