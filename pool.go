package pool

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

type Connect func(s string) (Connection, error)

type qitem struct {
	wakemeUp chan bool
}

type pool struct {
	connect               Connect
	maxConnectionPoolSize int
	servers               map[string]*server
	serverMutex           sync.Mutex
	queue                 list.List
	queueMutex            sync.Mutex
}

func New(connect Connect, poolSize int) *pool {
	p := &pool{
		connect:               connect,
		maxConnectionPoolSize: poolSize,
		servers:               make(map[string]*server),
		serverMutex:           sync.Mutex{},
		queueMutex:            sync.Mutex{},
	}
	fmt.Println("pool created")
	return p
}

func (p *pool) borrow(serverName string, idlenessThreshold time.Duration) (Connection, error) {
	for {
		//check if pool is closed
		//try borrowing connection
		var conn Connection
		var err error
		conn, err = p.tryborrow(serverName, idlenessThreshold)
		if conn != nil {
			return conn, err
		}
		//if unsuccessfull create wait request and let other thread know
		p.queueMutex.Lock()
		//try to find idle connection before waiting
		conn, err = p.tryIdle(serverName)
		if err != nil {
			p.queueMutex.Unlock()
			return nil, err
		}
		if conn != nil {
			p.queueMutex.Unlock()
			return conn, nil
		}
		fmt.Println("wait request sent to the queue")
		q := &qitem{
			wakemeUp: make(chan bool, 1),
		}
		p.queue.PushBack(q)
		p.queueMutex.Unlock()
		//waits for signal which indicates that we got connection or timeout
		select {
		case <-q.wakemeUp:
			continue
		}
	}
}

func (p *pool) tryborrow(serverName string, idlenessThreshold time.Duration) (Connection, error) {
	// check for server connections
	for {
		p.serverMutex.Lock()
		defer p.serverMutex.Unlock()
		serv := p.servers[serverName]
		for {
			if serv != nil {
				connection := serv.getIdle()
				if connection == nil {
					//check if server has reached maxConnections then wait
					if serv.size() >= p.maxConnectionPoolSize {
						return nil, nil
					}
					break
				}
				healthy, err := serv.healthCheck(connection, idlenessThreshold)
				if healthy {
					return connection, nil
				}
				if err != nil {
					fmt.Printf("Health check failed for %s: %s\n", serverName, err)
					return nil, err
				}

			} else {
				//1st time connection to server
				serv = newServer()
				p.servers[serverName] = serv
				break
			}
		}
		//no idle connection, try to connect server
		c, err := p.connect(serverName)
		serv.registerBusy(c)
		return c, err
	}
}

func (p *pool) tryIdle(serverName string) (Connection, error) {
	p.serverMutex.Lock()
	defer p.serverMutex.Unlock()
	for {
		srv := p.servers[serverName]
		if srv != nil {
			conn := srv.getIdle()
			if conn == nil {
				break
			}
			if conn != nil {
				return conn, nil
			}
		}
	}
	return nil, nil
}

func newServer() *server {
	return &server{
		idle: list.List{},
		busy: list.List{},
	}
}

func (p *pool) Return(c Connection) {
	isAlive := c.IsAlive()
	serverName := c.ServerName()

	if isAlive {
		p.serverMutex.Lock()
		//return connection to idle list
		server := p.servers[serverName]
		if server != nil {
			server.returnBusy(c)
		}
		p.serverMutex.Unlock()
	}
	//pick up wait request and signal back that connection is available
	p.queueMutex.Lock()
	for e := p.queue.Front(); e != nil; e = e.Next() {
		queueItem := e.Value.(*qitem)
		p.queue.Remove(e)
		queueItem.wakemeUp <- true
	}
	p.queueMutex.Unlock()
}
