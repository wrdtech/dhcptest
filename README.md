dhcp-test-tool
=======
dhcp-test-tool是一个用go语言编写的，可跨平台的DHCP性能测试工具。它可以指定模拟终端数量和发包速率(qps)，也可以输出DHCP包的信息。支持多种DHCP option指定模式。

## Dependencies

* [github.com/google/gopacket](https://github.com/google/gopacket) 序列化和反序列化DHCP包
* [github.com/mdlayher/raw](https://github.com/mdlayher/raw) 非windows平台上的网络连接
* [github.com/pinterest/bender](https://github.com/pinterest/bender) 负载测试的辅助包

**根据项目需求,修改了gopacket中的layer包和bender库中的核心函数。请不要升级bender库，那样会导致程序性能损失**
  
## Building

编译本程序需要Go的版本为1.12.1及以上&nbsp;&nbsp;[Go1.12.1下载地址](https://golang.org/dl/)


首先从github上克隆项目到本地的$GOPATH/src目录中去
```sh
cd $GOPATH
git clone https://github.com/wrdtech/dhcptest.git
```

然后切换到项目的根目录dhcptest中开始编译
```sh
cd dhcptest
go build
```
如要编译不同平台不同架构上的版本，先设置GOOS和GOARCH再编译。示例为在windows系统上编译可在linux系统，amd64架构的平台上运行的程序
```sh 
cd dhcpetst
set GOOS=linux //linux上设置环境变量为export GOOS=linux (可选值为darwin,dragonfly,freebsd,netbsd,openbsd)
go build
```

## Usage

### **必选参数**
先用命令行参数查看并选择绑定的网卡，然后进入交互模式发包
```sh 
./dhcptest --iface-list //查看网卡
./dhcptest --bind $iface //iface为选择的可用的网卡名称
```
进入交互模式后，d为发送discover包，r为发送request包，可指定终端数量和发送速率
```sh 
d  //发送一次discover包并接收offer包，终端数量为1
d 5 //发送一次discover包并接收offer包，终端数量为5
d 5 100 //发送discover包的速率为每秒100次，终端数量为5
r  //发送一次discover包，收到offer包之后发送request包，终端数量为1
r 5 //发送一次discover包，收到offer包之后发送request包，终端数量为5
r 5 100 //发送discover包的速率为每秒100次，收到offer包之后发送request包,终端数量为5
```

### **可选参数**
--option 可用来指定dhcp包中的option，可多次指定。具体使用方法请查看--help

--mac    可用来指定模拟终端的mac地址，可多次指定。

若模拟终端数量大于指定的mac地址数量，会随机产生剩余的mac地址。若模拟终端数量小于指定的mac地址数量，会选取最先指定的mac地址

其余参数请使用--help查看，或在交互模式下键入h或help