package gateway

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/k2wanko/grpc-pipe/testdata/echo"
	"golang.org/x/net/context"
)

type echo struct{}

func (*echo) Echo(ctx context.Context, m *pb.Message) (*pb.Message, error) {
	return m, nil
}

func newTestRequest(b string) *http.Request {
	r, _ := http.NewRequest(
		"POST",
		"/echo",
		bytes.NewBuffer([]byte(b)))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestServer(t *testing.T) {
	ctx := context.Background()
	s := New(ctx)
	s.RegisterService(pb.RegisterEchoServiceServer, pb.RegisterEchoServiceHandler, new(echo))

	b := `{"value":"Hi"}`
	r := newTestRequest(b)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if code, want := w.Code, 200; code != want {
		t.Errorf("code = %d; want %d", code, want)
	}

	if body, want := w.Body.String(), b; body != want {
		t.Errorf("body = %s; want %s", body, want)
	}
}
