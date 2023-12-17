package pool

import "time"

type Connection interface {
	Connect(s string) error
	IsAlive() bool
	ServerName() string
	idleTime() time.Time
	ping()
	close()
}
