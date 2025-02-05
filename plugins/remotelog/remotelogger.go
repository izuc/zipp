package remotelog

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/izuc/zipp.foundation/core/logger"

	"github.com/izuc/zipp/packages/node/clock"
	"github.com/izuc/zipp/plugins/banner"
)

// RemoteLoggerConn is a wrapper for a connection to our RemoteLog server.
type RemoteLoggerConn struct {
	conn net.Conn
}

func newRemoteLoggerConn(address string) (*RemoteLoggerConn, error) {
	c, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("could not create UDP socket to '%s'. %v", address, err)
	}

	return &RemoteLoggerConn{conn: c}, nil
}

// SendLogMsg sends log message to the remote logger.
func (r *RemoteLoggerConn) SendLogMsg(level logger.Level, name, msg string) {
	m := logBlock{
		banner.AppVersion,
		myGitHead,
		myGitConflict,
		myID,
		level.CapitalString(),
		name,
		msg,
		clock.SyncedTime(),
		remoteLogType,
	}

	_ = deps.RemoteLogger.Send(m)
}

// Send sends a message on the RemoteLoggers connection.
func (r *RemoteLoggerConn) Send(msg interface{}) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = r.conn.Write(b)
	if err != nil {
		return err
	}

	return nil
}
