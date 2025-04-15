package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/sund3RRR/crafty-reverse-proxy/config"
)

const tickerCooldown = 1 * time.Second
const awaitTimeout = 5 * time.Minute
const dialTimeout = 1 * time.Second

type (
	Logger interface {
		Debug(format string, args ...any)
		Warn(format string, args ...any)
		Info(format string, args ...any)
		Error(format string, args ...any)
	}
	Crafty interface {
		StartMcServer(port int) error
		StopMcServer(port int) error
	}
)

type ProxyServer struct {
	listenAddr string
	targetAddr string
	protocol   string

	logger            Logger
	connectController *ConnectController
}

func NewProxyServer(cfg config.Config, proxyCfg config.ServerType, logger Logger, crafty Crafty) *ProxyServer {
	serverOperator := NewServerOperator(proxyCfg, cfg.Timeout, logger, crafty)
	ps := &ProxyServer{
		protocol:          proxyCfg.Protocol,
		listenAddr:        fmt.Sprintf("%s:%d", proxyCfg.Listener.Addr, proxyCfg.Listener.Port),
		targetAddr:        fmt.Sprintf("%s:%d", proxyCfg.CraftyHost.Addr, proxyCfg.CraftyHost.Port),
		logger:            logger,
		connectController: NewConnectController(logger, serverOperator, dialTimeout),
	}

	return ps
}

func (ps *ProxyServer) ListenAndProxy(ctx context.Context) error {
	ps.connectController.StartLoop(ctx)

	listener, err := net.Listen(ps.protocol, ps.listenAddr)
	if err != nil {
		return fmt.Errorf("%w with protocol %s, err: %w", ErrStartingServer, ps.protocol, err)
	}
	defer func() {
		listener.Close()
		ps.logger.Info("Listener closed for external address: %s", ps.targetAddr)
	}()

	ps.logger.Info("%s: reverse proxy running on %s, forwarding to %s", ps.protocol, ps.listenAddr, ps.targetAddr)

	for {
		client, err := listener.Accept()
		if err != nil {
			ps.logger.Error("Failed to accept connection: %v", err)
			continue
		}

		go func() {
			if err := ps.handleClient(ctx, client); err != nil {
				ps.logger.Error("Failed to handle client: %v", err)
			}
		}()
	}
}

func (ps *ProxyServer) handleClient(ctx context.Context, client net.Conn) (err error) {
	defer client.Close()

	serverConnection, err := ps.connectController.GetConnection(ctx)
	defer ps.connectController.PutConnection(ctx, serverConnection)
	if err != nil {
		return err
	}

	ps.logger.Info("Starting proxy from %s to %s", client.RemoteAddr(), serverConnection.RemoteAddr())

	completed := make(chan struct{})
	go func() {
		defer func() {
			completed <- struct{}{}
			close(completed)
		}()
		_, err := io.Copy(client, serverConnection)
		if err != nil {
			ps.logger.Warn("An error occurred copying from server to client: %v", err)
		}
		ps.logger.Info("Proxying from %s to %s completed", client.RemoteAddr(), serverConnection.RemoteAddr())
	}()

	_, err = io.Copy(serverConnection, client)
	if err != nil {
		ps.logger.Error("Error copying from client to server: %s", err)
	}

	<-completed

	return nil
}
