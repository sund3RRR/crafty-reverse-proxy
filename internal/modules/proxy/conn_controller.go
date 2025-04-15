package proxy

import (
	"context"
	"net"
	"time"
)

// Constants representing the possible states of the state machine.
const (
	StateOff State = iota
	StateStartingUp
	StateRunning
	StateEmpty
	StateShuttingDown
)

func String(state State) string {
	switch state {
	case StateOff:
		return "Off"
	case StateStartingUp:
		return "StartingUp"
	case StateRunning:
		return "Running"
	case StateEmpty:
		return "Empty"
	case StateShuttingDown:
		return "ShuttingDown"
	default:
		return "unknown"
	}
}

// State represents the state of the state machine
type State = int32

type IServerOperator interface {
	StartMinecraftServer() error
	IsServerRunning() bool
	ConnectToServer() (net.Conn, error)
	AwaitForServerStart(ctx context.Context, timeout, cooldown time.Duration) error
	ScheduleShutdown()
	StopShuttingDown()
}

type ConnConfig struct {
	Protocol    string
	TargetAddr  string
	DialTimeout time.Duration
}

type connPackage struct {
	conn net.Conn
	err  error
}

type ConnectController struct {
	playerCount    int
	state          State
	dialTimeout    time.Duration
	logger         Logger
	serverOperator IServerOperator
	getConnCh      chan struct{}
	connCh         chan connPackage
	putConnCh      chan net.Conn
}

func NewConnectController(logger Logger, serverOperator IServerOperator, dialTimeout time.Duration) *ConnectController {
	getInitialState := func(serverOperator IServerOperator) State {
		if serverOperator.IsServerRunning() {
			serverOperator.ScheduleShutdown()
			return StateEmpty
		}
		return StateOff
	}

	return &ConnectController{
		playerCount:    0,
		state:          getInitialState(serverOperator),
		dialTimeout:    dialTimeout,
		logger:         logger,
		serverOperator: serverOperator,
		getConnCh:      make(chan struct{}),
		connCh:         make(chan connPackage),
		putConnCh:      make(chan net.Conn),
	}
}

func (cc *ConnectController) GetConnection(ctx context.Context) (net.Conn, error) {
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

func (cc *ConnectController) PutConnection(ctx context.Context, conn net.Conn) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, cc.dialTimeout)
	defer cancel()

	select {
	case <-ctxWithTimeout.Done():
		return context.Canceled
	case cc.putConnCh <- conn:
		return nil
	}
}

func (cc *ConnectController) StartLoop(ctx context.Context) {
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
					cc.setState(StateEmpty)
					cc.serverOperator.ScheduleShutdown()
				}
				conn.Close()
			}
		}
	}
}

func (cc *ConnectController) processState(ctx context.Context) (net.Conn, error) {
	for {
		switch cc.getState() {
		case StateOff, StateShuttingDown:
			if err := cc.serverOperator.StartMinecraftServer(); err != nil {
				return nil, err
			}
			cc.setState(StateStartingUp)
		case StateStartingUp:
			if err := cc.serverOperator.AwaitForServerStart(ctx, awaitTimeout, tickerCooldown); err != nil {
				return nil, err
			}
			cc.setState(StateEmpty)
		case StateEmpty:
			cc.serverOperator.StopShuttingDown()
			cc.setState(StateRunning)
		case StateRunning:
			serverConnection, err := cc.serverOperator.ConnectToServer()
			if err != nil {
				cc.setState(StateOff)
				return nil, err
			}
			cc.playerCount++
			return serverConnection, nil
		}
	}
}

func (cc *ConnectController) setState(newState State) {
	cc.state = newState
}

func (cc *ConnectController) getState() State {
	return cc.state
}
