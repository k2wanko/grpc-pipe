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
		ctx:  ctx,
		s:    grpc.NewServer(opts.grpcopts...),
		cc:   cc,
		mux:  runtime.NewServeMux(opts.gwopts...),
		reqs: make(map[string]*http.Request),
	}

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

func (s *Server) RegisterService(sf interface{}, cf func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error, srv interface{}) {
	f := reflect.ValueOf(sf)
	f.Call([]reflect.Value{
		reflect.ValueOf(s.s),
		reflect.ValueOf(srv),
	})
	cf(s.ctx, s.mux, s.cc)
}
