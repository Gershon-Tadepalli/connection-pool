package pool

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

const DefaultLivenessCheckThreshold = math.MaxInt64

func TestNew(t *testing.T) {
	type args struct {
		connect  Connect
		poolSize int
	}
	tests := []struct {
		name string
		args args
		want *pool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.connect, tt.args.poolSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pool_borrow_return(outer *testing.T) {
	birthdate := time.Now()
	succedingConnect := func(s string) (Connection, error) {
		return &ConnFake{Name: s, Alive: true, Birth: birthdate}, nil
	}

	outer.Run("single thread borrow+return", func(t *testing.T) {
		maxConnectionPoolSize := 2
		p := New(succedingConnect, maxConnectionPoolSize)
		conn, err := p.borrow("srv1", DefaultLivenessCheckThreshold)
		assertConnection(t, conn, err)
		p.Return(conn)

		server := p.servers["srv1"]
		if server.numIdle() != 1 {
			t.Fatal("should be one ready connection in server")
		}
	})

	outer.Run("first thread borrow,second thread blocks on borrow", func(t *testing.T) {
		maxConnectionPoolSize := 2
		p := New(succedingConnect, maxConnectionPoolSize)
		wg := sync.WaitGroup{}
		wg.Add(1)
		//first thread borrows
		c1, err1 := p.borrow("srv1", DefaultLivenessCheckThreshold)
		assertConnection(t, c1, err1)

		go func() {
			//second thread borrows
			c2, err2 := p.borrow("srv1", DefaultLivenessCheckThreshold)
			assertConnection(t, c2, err2)
			wg.Done()
		}()

		p.Return(c1)
		wg.Wait()
	})

	outer.Run("multiple threads borrows and returns", func(t *testing.T) {
		fmt.Println("multiple clients connecting to server1")
		maxConnectionPoolSize := 2
		serverName := "srv1"
		p := New(succedingConnect, maxConnectionPoolSize)
		wg := sync.WaitGroup{}
		numWorkers := 5
		wg.Add(5)

		worker := func() {
			for i := 0; i < 5; i++ {
				c, err := p.borrow(serverName, DefaultLivenessCheckThreshold)
				assertConnection(t, c, err)
				time.Sleep(time.Duration(rand.Int()%7) * time.Millisecond)
				p.Return(c)

			}
			wg.Done()
		}
		for i := 0; i < numWorkers; i++ {
			go worker()
		}
		wg.Wait()

		srv := p.servers[serverName]
		if srv.numIdle() != maxConnectionPoolSize {
			t.Error("connection still in use in the server")
		}
	})

	outer.Run("Borrows the first successfully alive connection", func(t *testing.T) {
		//setup 3 idle connections where 1 - dead, 2 -alive and ready , 3- alive but not ready
		idlenessThreshold := 1 * time.Hour
		idleness := time.Now().Add(-2 * idlenessThreshold)
		maxConnectionPoolSize := 1
		deadAfterPing := deadConnectionAfterPing("deadAfterPing", idleness)
		stayingAlive := &ConnFake{Name: "stayingAlive", Alive: true, Idle: idleness, pingHook: func() {}}
		evenIamAlive := &ConnFake{Name: "evenIamAlive", Alive: true, Idle: idleness, pingHook: func() { t.Errorf("don't disturb me") }}
		pool := New(nil, maxConnectionPoolSize)
		setIdleConnections(pool, map[string][]Connection{"server-1": {
			deadAfterPing,
			stayingAlive,
			evenIamAlive,
		}})
		result, err := pool.tryborrow("server-1", idlenessThreshold)
		if err != nil {
			t.Errorf("expected nil but thrown error: %s", err)
		}
		if !reflect.DeepEqual(result, stayingAlive) {
			t.Errorf("expected value %v to equal value %v", result, stayingAlive)
		}
	})
}

func assertConnection(t *testing.T, c Connection, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("No connection")
	}
}
func deadConnectionAfterPing(name string, idleness time.Time) *ConnFake {
	result := &ConnFake{Name: name, Alive: true, Idle: idleness}
	result.pingHook = func() {
		result.Alive = false
	}
	return result
}

func setIdleConnections(pool *pool, servers map[string][]Connection) {
	poolServers := make(map[string]*server, len(servers))
	for servername, connections := range servers {
		srv := newServer()
		for i := len(connections) - 1; i >= 0; i-- {
			registerIdle(srv, connections[i])
		}
		poolServers[servername] = srv
	}
	pool.servers = poolServers
}

func registerIdle(srv *server, connection Connection) {
	srv.registerBusy(connection)
	srv.returnBusy(connection)
}
