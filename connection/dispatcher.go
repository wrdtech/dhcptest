package connection

var (
	discoverDequeue   = "discover start send"
	requestDequeue    = "request send"
	receivedOffer     = "received offer"
	receivedAck       = "received ack"
	receivedNak       = "received nak"
	offerTimeout      = "offer time out"
	ackNakTimeout     = "ack Timeout"
)

type PacketEventHandler func(e PacketEvent)

type IPacketEventDispatcher interface {
	AddEventListener(eventType string, handler PacketEventHandler)
	RemoveEventListener(eventType string)
	HasEventListener(eventType string) bool
	DispatchEvent(event PacketEvent)
}

type PacketEventListener struct {
	Type    string
	Handler PacketEventHandler
}

type PacketEvent struct {
	Target IPacketEventDispatcher
	Type   string
	object interface{}
}

func NewEvent(eventType string, object interface{}) PacketEvent {
	return PacketEvent{Type: eventType, object: object}
}

type PacketEventDispatcher struct {
	listeners []*PacketEventListener
}

func (pd *PacketEventDispatcher) AddEventListener(eventType string, handler PacketEventHandler) {
	for _, l := range pd.listeners {
		if l.Type == eventType {
			l.Handler = handler
			return
		}
	}

	listener := &PacketEventListener{Type: eventType, Handler: handler}
	pd.listeners = append(pd.listeners, listener)
}

func (pd *PacketEventDispatcher) RemoveEventListener(eventType string) {
	for i, l := range pd.listeners {
		if l.Type == eventType {
			pd.listeners = append(pd.listeners[:i], pd.listeners[i+1:]...)
		}
	}
}

func (pd *PacketEventDispatcher) HasEventListener(eventType string) bool {
	for _, l := range pd.listeners {
		if l.Type == eventType {
			return true
		}
	}
	return false
}

func (pd *PacketEventDispatcher) DispatchEvent(event PacketEvent) {
	for _, l := range pd.listeners {
		if l.Type == event.Type {
			l.Handler(event)
		}
	}
}

