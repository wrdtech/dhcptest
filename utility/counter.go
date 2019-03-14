package utility

import (
	"fmt"
	"sync"
)

var (
	DHCPCounter map[string]*Counter
)

type Counter struct {
	request int
	response int
	percentage float32
	qLock *sync.Mutex
	rLock *sync.Mutex
}

func (c *Counter) Init() {
	c.qLock = new(sync.Mutex)
	c.rLock = new(sync.Mutex)
}

func (c *Counter) GetRequest() int {
	//c.qLock.Lock()
	request := c.request
	//c.qLock.Unlock()
	return request
}

func (c *Counter) GetResponse() int {
	//c.rLock.Lock()
	response := c.response
	//c.rLock.Unlock()
	return response
}

func (c *Counter) AddRequest(n int) {
	request := c.GetRequest()
	//c.qLock.Lock()
	c.request = request + n
	//c.qLock.Unlock()
}

func (c *Counter) AddResponse(n int) {
	response := c.GetResponse()
	//c.rLock.Lock()
	c.response = response + n
	//c.rLock.Unlock()
}

func (c *Counter) GetPercentage() string {
	request := c.GetRequest()
	response := c.GetResponse()
	c.percentage = float32(response * 1.0) / float32(request * 1.0)
	return fmt.Sprintf("%.2f%%", c.percentage * 100)

}
