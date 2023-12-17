package pool

import "time"

// Protocol takes care of implementing the specific functionalities
type ConnFake struct {
	Name     string
	Alive    bool
	Birth    time.Time
	Idle     time.Time
	pingHook func()
	IsClosed bool
}

func (c *ConnFake) Connect(s string) error {
	return nil
}

func (c *ConnFake) IsAlive() bool {
	return c.Alive
}

func (c *ConnFake) ServerName() string {
	return c.Name
}

func (c *ConnFake) idleTime() time.Time {
	return c.Idle
}

func (c *ConnFake) ping() {
	//ping the db server
	c.pingHook()
}

func (c *ConnFake) close() {
	//close the connection
	if c.IsClosed == false {
		c.IsClosed = true
	}
}
