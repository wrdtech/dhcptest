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
	qLock *sync.RWMutex
	rLock *sync.RWMutex
}

func (c *Counter) Init() {
	c.qLock = new(sync.RWMutex)
	c.rLock = new(sync.RWMutex)
}

func (c *Counter) GetRequest() int {
	c.qLock.RLock()
	request := c.request
	c.qLock.RUnlock()
	return request
}

func (c *Counter) GetResponse() int {
	c.rLock.RLock()
	response := c.response
	c.rLock.RUnlock()
	return response
}

func (c *Counter) AddRequest(n int) {
	request := c.GetRequest()
	c.qLock.Lock()
	c.request = request + n
	c.qLock.Unlock()
}

func (c *Counter) AddResponse(n int) {
	response := c.GetResponse()
	c.rLock.Lock()
	c.response = response + n
	c.rLock.Unlock()
}

func (c *Counter) GetPercentage() string {
	request := c.GetRequest()
	response := c.GetResponse()
	c.percentage = float32(response * 1.0) / float32(request * 1.0)
	return fmt.Sprintf("%.2f%%", c.percentage * 100)

}
