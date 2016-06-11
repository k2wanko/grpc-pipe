package gateway

import (
	"net/http"
	"reflect"
	"sync"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/gengo/grpc-gateway/runtime"
	"github.com/k2wanko/grpc-pipe"
)

type Server struct {
	ctx context.Context
	mu  sync.Mutex
	s   *grpc.Server
	cc  *grpc.ClientConn
	mux *runtime.ServeMux
}

type options struct {
	grpcopts []grpc.ServerOption
	gwopts   []runtime.ServeMuxOption
}

type ServerOption func(*options)

func WithGrpcOptions(opts ...grpc.ServerOption) ServerOption {
	return func(o *options) {
		o.grpcopts = opts
	}
}

func WithGatewayOptions(opts ...runtime.ServeMuxOption) ServerOption {
	return func(o *options) {
		o.gwopts = opts
	}
}

func New(ctx context.Context, opt ...ServerOption) *Server {
	opts := new(options)
	for _, o := range opt {
		o(opts)
	}

	l := pipe.Listen()

	cc, err := grpc.Dial("", grpc.WithInsecure(), l.WithDialer())
	if err != nil {
		panic(err)
	}

	s := &Server{
		ctx: ctx,
		s:   grpc.NewServer(opts.grpcopts...),
		cc:  cc,
		mux: runtime.NewServeMux(opts.gwopts...),
	}

	go s.s.Serve(l)
	go func() {
		<-ctx.Done()
		s.s.Stop()
	}()

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) RegisterService(sf interface{}, cf func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error, srv interface{}) {
	f := reflect.ValueOf(sf)
	f.Call([]reflect.Value{
		reflect.ValueOf(s.s),
		reflect.ValueOf(srv),
	})
	cf(s.ctx, s.mux, s.cc)
}
