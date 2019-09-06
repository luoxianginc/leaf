package gate

import (
	"net"
	"reflect"
	"time"

	"github.com/luoxianginc/leaf/chanrpc"
	"github.com/luoxianginc/leaf/log"
	"github.com/luoxianginc/leaf/network"
)

type Gate struct {
	MaxConnNum      int
	PendingWriteNum int
	MaxMsgLen       uint32
	Processor       network.Processor
	AgentChanRPC    *chanrpc.Server

	// websocket
	WSAddr      string
	WSPort      string
	WSSPort     string
	HTTPTimeout time.Duration
	CertFile    string
	KeyFile     string

	// tcp
	TCPAddr      string
	LenMsgLen    int
	LittleEndian bool
}

func (gate *Gate) Run(closeSig chan bool) {
	var wsServers []*network.WSServer
	if gate.WSAddr != "" && gate.WSPort != "" {
		wsServers = append(wsServers, gate.newWSServer(gate.WSPort, false))
	}
	if gate.WSAddr != "" && gate.WSSPort != "" {
		wsServers = append(wsServers, gate.newWSServer(gate.WSSPort, true))
	}

	var tcpServer *network.TCPServer
	if gate.TCPAddr != "" {
		tcpServer = new(network.TCPServer)
		tcpServer.Addr = gate.TCPAddr
		tcpServer.MaxConnNum = gate.MaxConnNum
		tcpServer.PendingWriteNum = gate.PendingWriteNum
		tcpServer.LenMsgLen = gate.LenMsgLen
		tcpServer.MaxMsgLen = gate.MaxMsgLen
		tcpServer.LittleEndian = gate.LittleEndian
		tcpServer.NewAgent = func(conn *network.TCPConn) network.Agent {
			a := &agent{conn: conn, gate: gate}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			return a
		}
	}

	if wsServers != nil {
		for _, s := range wsServers {
			s.Start()
		}
	}
	if tcpServer != nil {
		tcpServer.Start()
	}

	<-closeSig
	if wsServers != nil {
		for _, s := range wsServers {
			s.Close()
		}
	}
	if tcpServer != nil {
		tcpServer.Close()
	}
}

func (gate *Gate) newWSServer(port string, isWSS bool) *network.WSServer {
	addr := gate.WSAddr + ":" + port
	wsServer := &network.WSServer{
		Addr:            addr,
		MaxConnNum:      gate.MaxConnNum,
		PendingWriteNum: gate.PendingWriteNum,
		MaxMsgLen:       gate.MaxMsgLen,
		HTTPTimeout:     gate.HTTPTimeout,
		NewAgent: func(conn *network.WSConn) network.Agent {
			a := &agent{conn: conn, gate: gate}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			return a
		},
	}

	if isWSS {
		if gate.CertFile == "" || gate.KeyFile == "" {
			panic("lack of SSL cert or key file")
		}

		wsServer.CertFile = gate.CertFile
		wsServer.KeyFile = gate.KeyFile
	}

	return wsServer
}

func (gate *Gate) OnDestroy() {}

type agent struct {
	conn     network.Conn
	gate     *Gate
	userData interface{}
}

func (a *agent) Run() {
	for {
		data, err := a.conn.ReadMsg()
		if err != nil {
			log.Debug("read message: %v", err)
			break
		}

		if a.gate.Processor != nil {
			msg, err := a.gate.Processor.Unmarshal(data)
			if err != nil {
				log.Debug("unmarshal message error: %v", err)
				break
			}
			err = a.gate.Processor.Route(msg, a)
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		}
	}
}

func (a *agent) OnClose() {
	if a.gate.AgentChanRPC != nil {
		err := a.gate.AgentChanRPC.Call0("CloseAgent", a)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}
}

func (a *agent) WriteMsg(msg interface{}) {
	if a.gate.Processor != nil {
		data, err := a.gate.Processor.Marshal(msg)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *agent) Close() {
	a.conn.Close()
}

func (a *agent) Destroy() {
	a.conn.Destroy()
}

func (a *agent) UserData() interface{} {
	return a.userData
}

func (a *agent) SetUserData(data interface{}) {
	a.userData = data
}
