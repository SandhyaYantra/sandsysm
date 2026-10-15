package system

import (
	"sync"

	"github.com/godbus/dbus/v5"
)

var (
	conn   *dbus.Conn
	connMu sync.Mutex
)

func GetDBusConn() (*dbus.Conn, error) {
	connMu.Lock()
	defer connMu.Unlock()

	if conn != nil {
		return conn, nil
	}

	c, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	conn = c
	return conn, nil
}
