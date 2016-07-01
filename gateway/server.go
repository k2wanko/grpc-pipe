package gateway

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/gengo/grpc-gateway/runtime"
	"github.com/k2wanko/grpc-pipe"
)

const header = "x-grpc-pipe-gateway-request-id"

type ctxKey struct {
	key string
}

var (
	ServerContextKey = &ctxKey{"server"}
)

type Server struct {
	ctx  context.Context
	mu   sync.RWMutex
	s    *grpc.Server
	cc   *grpc.ClientConn
	mux  *runtime.ServeMux
	reqs map[string]*http.Request
}

type options struct {
	grpcopts []grpc.ServerOption
	gwopts   []runtime.ServeMuxOption
	unaryInt grpc.UnaryServerInterceptor
}

type ServerOption func(*options)

func withGrpcOptions(opts ...grpc.ServerOption) ServerOption {
	return func(o *options) {
		o.grpcopts = opts
	}
}

func WithGatewayOptions(opts ...runtime.ServeMuxOption) ServerOption {
	return func(o *options) {
		o.gwopts = opts
	}
}

func UnaryInterceptor(i grpc.UnaryServerInterceptor) ServerOption {
	return func(o *options) {
		if o.unaryInt != nil {
			panic("The unary server interceptor has been set.")
		}
		o.unaryInt = i
	}
}

func ctxValInjector(srv *Server) grpc.ServerOption {
	return grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = context.WithValue(ctx, ServerContextKey, srv)
		return handler(ctx, req)
	})
}

func New(ctx context.Context, opt ...ServerOption) *Server {
	s := &Server{
		ctx:  ctx,
		reqs: make(map[string]*http.Request),
	}

	opt = append(opt, withGrpcOptions(ctxValInjector(s)))

	opts := new(options)
	for _, o := range opt {
		o(opts)
	}

	l := pipe.Listen()

	cc, err := grpc.Dial("", grpc.WithInsecure(), l.WithDialer())
	if err != nil {
		panic(err)
	}

	s.s = grpc.NewServer(opts.grpcopts...)
	s.cc = cc
	s.mux = runtime.NewServeMux(opts.gwopts...)

	go s.s.Serve(l)
	go func() {
		<-ctx.Done()
		s.s.Stop()
	}()

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//Inject Header
	for name, hs := range r.Header {
		for _, h := range hs {
			r.Header.Add("Grpc-Metadata-"+name, h)
		}
	}
	key := fmt.Sprintf("%x", &r)
	s.mu.Lock()
	s.reqs[key] = r
	s.mu.Unlock()
	r.Header.Add("Grpc-Metadata-"+header, key)
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.reqs, key)
	}()

	s.mux.ServeHTTP(w, r)
}

func (s *Server) Request(ctx context.Context) *http.Request {
	md, _ := metadata.FromContext(ctx)
	if md == nil {
		return nil
	}
	ids := md[header]
	if len(ids) < 1 {
		return nil
	}
	id := ids[0]
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.reqs[id]; ok {
		return r
	}
	return nil
}

func (s *Server) RegisterService(srv interface{}, sf interface{}, cf func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error) {
	f := reflect.ValueOf(sf)
	f.Call([]reflect.Value{
		reflect.ValueOf(s.s),
		reflect.ValueOf(srv),
	})
	cf(s.ctx, s.mux, s.cc)
}
