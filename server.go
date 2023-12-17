package pool

import (
	"container/list"
	"time"
)

type server struct {
	idle list.List
	busy list.List
}

func (s *server) registerBusy(c Connection) {
	s.busy.PushFront(c)
}

func (s *server) numIdle() int {
	return s.idle.Len()
}

func (s *server) numBusy() int {
	return s.busy.Len()
}

func (s *server) returnBusy(c Connection) {
	//check for connection in busy connection list and remove
	found := false
	for e := s.busy.Front(); e != nil && !found; e = e.Next() {
		found = c == e.Value.(Connection)
		if found {
			s.busy.Remove(e)
		}
	}
	//add to idle list
	s.idle.PushFront(c)
}

func (s *server) size() int {
	return s.idle.Len() + s.busy.Len()
}

func (s *server) getIdle() Connection {
	//find idle connection if found push to busy
	availableConnection := s.idle.Front()
	found := availableConnection != nil
	if found {
		connection := availableConnection.Value.(Connection)
		s.idle.Remove(availableConnection)
		s.busy.PushFront(connection)
		return connection
	}
	return nil
}

func (s *server) healthCheck(connection Connection, idlenessThreshold time.Duration) (bool, error) {
	//check if connection is idle more than threshold then do pings and check isAlive
	if time.Since(connection.idleTime()) > idlenessThreshold {
		connection.ping()
		if !connection.IsAlive() {
			s.closeConnection(connection)
			return false, nil
		}
	}

	if !connection.IsAlive() {
		return false, nil
	}
	return true, nil
}

func (s *server) closeConnection(connection Connection) {
	//unhealthy connection close it and remove from the pool
	connection.close()
	for e := s.busy.Front(); e != nil; e = e.Next() {
		if connection == e.Value.(Connection) {
			s.busy.Remove(e)
		}
	}
}
