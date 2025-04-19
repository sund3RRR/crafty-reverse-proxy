// Package connector provides the main logic for handling player connections to the Minecraft server.
package connector

import (
	"context"
	"net"
	"sync/atomic"
	"time"
)

// Logger defines the logging interface used throughout the Connector.
type Logger interface {
	Debug(format string, args ...any)
	Warn(format string, args ...any)
	Info(format string, args ...any)
	Error(format string, args ...any)
}

// ServerOperator defines the interface to manage the lifecycle of a Minecraft server.
type ServerOperator interface {
	StartMinecraftServer() error
	IsServerRunning() bool
	ConnectToServer() (net.Conn, error)
	AwaitForServerStart(ctx context.Context) error
	ScheduleShutdown(shutdownEmitter chan<- struct{})
	StopShuttingDown()
}

// ConnConfig represents the configuration required to establish a connection.
type ConnConfig struct {
	Protocol    string
	TargetAddr  string
	DialTimeout time.Duration
}

type connPackage struct {
	conn net.Conn
	err  error
}

// Connector handles player connections to the Minecraft server,
// managing server state and lifecycle transitions based on connection requests.
type Connector struct {
	playerCount    int
	autoshutdown   bool
	state          state
	dialTimeout    time.Duration
	logger         Logger
	serverOperator ServerOperator
	getConnCh      chan struct{}
	shutdownCh     chan struct{}
	connCh         chan connPackage
	putConnCh      chan net.Conn
}

// New creates and initializes a new Connector instance.
func New(logger Logger, autoshutdown bool, serverOperator ServerOperator, dialTimeout time.Duration) *Connector {
	return &Connector{
		playerCount:    0,
		autoshutdown:   autoshutdown,
		state:          stateOff,
		dialTimeout:    dialTimeout,
		logger:         logger,
		serverOperator: serverOperator,
		getConnCh:      make(chan struct{}),
		shutdownCh:     make(chan struct{}),
		connCh:         make(chan connPackage),
		putConnCh:      make(chan net.Conn),
	}
}

// GetConnection requests a connection to the Minecraft server.
// If the server is off, it will be started and waited on.
func (cc *Connector) GetConnection(ctx context.Context) (net.Conn, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, cc.dialTimeout)
	defer cancel()

	select {
	case <-ctxWithTimeout.Done():
		return nil, context.Canceled
	case cc.getConnCh <- struct{}{}:
	}

	select {
	case <-ctxWithTimeout.Done():
		return nil, context.Canceled
	case conn := <-cc.connCh:
		return conn.conn, conn.err
	}
}

// PutConnection returns a connection (usually when the player disconnects).
// If no players remain, a shutdown is scheduled.
func (cc *Connector) PutConnection(ctx context.Context, conn net.Conn) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, cc.dialTimeout)
	defer cancel()

	select {
	case <-ctxWithTimeout.Done():
		return context.Canceled
	case cc.putConnCh <- conn:
		return nil
	}
}

// StartLoop begins the main loop that handles connection and disconnection events.
// This method should be called once at application startup.
func (cc *Connector) StartLoop(ctx context.Context) {
	if cc.serverOperator.IsServerRunning() {
		cc.shutdownMiddleware()
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-cc.getConnCh:
				conn, err := cc.processState(ctx)
				cc.connCh <- connPackage{conn: conn, err: err}
			case conn := <-cc.putConnCh:
				if conn != nil {
					cc.playerCount--
					if cc.playerCount == 0 {
						cc.shutdownMiddleware()
					}
					conn.Close()
				}
			case <-cc.shutdownCh:
				if cc.getState() == stateEmpty {
					cc.setState(stateOff)
				}
			}
		}
	}()
}

// processState transitions through the server's lifecycle states until a connection is established.
func (cc *Connector) processState(ctx context.Context) (net.Conn, error) {
	for {
		switch cc.getState() {
		case stateOff:
			if err := cc.serverOperator.StartMinecraftServer(); err != nil {
				return nil, err
			}
			cc.setState(stateStartingUp)
		case stateStartingUp:
			if err := cc.serverOperator.AwaitForServerStart(ctx); err != nil {
				return nil, err
			}
			cc.setState(stateEmpty)
		case stateEmpty:
			cc.serverOperator.StopShuttingDown()
			cc.setState(stateRunning)
		case stateRunning:
			serverConnection, err := cc.serverOperator.ConnectToServer()
			if err != nil {
				cc.setState(stateOff)
				return nil, err
			}
			cc.playerCount++
			return serverConnection, nil
		}
	}
}

func (cc *Connector) shutdownMiddleware() {
	cc.setState(stateEmpty)
	if cc.autoshutdown {
		cc.serverOperator.ScheduleShutdown(cc.shutdownCh)
	}
}

// setState updates the internal state of the connector.
func (cc *Connector) setState(newState state) {
	atomic.StoreInt32(&cc.state, newState)
}

// getState retrieves the current internal state of the connector.
func (cc *Connector) getState() state {
	return atomic.LoadInt32(&cc.state)
}
