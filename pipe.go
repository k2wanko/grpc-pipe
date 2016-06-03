package pipe

import (
	"errors"
	"net"
	"time"

	"google.golang.org/grpc"
)

type GRPCAddr struct{}

func (a *GRPCAddr) Network() string { return "grpc-pipe" }

func (a *GRPCAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	return "grpc-pipe"
}

type GRPCListener struct {
	connCh chan net.Conn
	closer chan bool
}

func Listen() *GRPCListener {
	return &GRPCListener{
		connCh: make(chan net.Conn),
		closer: make(chan bool),
	}
}

var addr = &GRPCAddr{}

func (l *GRPCListener) Addr() net.Addr {
	return addr
}

func (l *GRPCListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connCh:
		return conn, nil
	case <-l.closer:
		return nil, errors.New("Listener closed")
	}
}

func (l *GRPCListener) Close() error {
	close(l.closer)
	return nil
}

func (l *GRPCListener) WithDialer() grpc.DialOption {
	return grpc.WithDialer(l.Dial)
}

func (l *GRPCListener) Dial(addr string, timeout time.Duration) (net.Conn, error) {
	select {
	case <-l.closer:
		return nil, errors.New("Listener closed")
	default:
	}

	srvConn, conn := net.Pipe()
	l.connCh <- srvConn
	return conn, nil
}
