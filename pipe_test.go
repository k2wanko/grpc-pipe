package pipe

import (
	"log"
	"net"
	"os"
	"sync"
	"testing"

	pb "github.com/k2wanko/grpc-pipe/testdata/echo"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ net.Addr = &GRPCAddr{}
var _ net.Listener = &GRPCListener{}

type echoSrv struct{}

func (s *echoSrv) Echo(ctx context.Context, msg *pb.Message) (*pb.Message, error) {
	return msg, nil
}

var (
	testL  = Listen()
	testCc pb.EchoServiceClient
)

func TestMain(m *testing.M) {
	s := grpc.NewServer()
	go s.Serve(testL)
	pb.RegisterEchoServiceServer(s, &echoSrv{})
	defer s.Stop()
	os.Exit(m.Run())
}

func TestListener(t *testing.T) {
	cc, err := grpc.Dial("", grpc.WithInsecure(), testL.WithDialer())
	if err != nil {
		t.Fatal(err)
		return
	}

	c := pb.NewEchoServiceClient(cc)
	ctx := context.Background()
	m := &pb.Message{Value: "Test"}
	res, err := c.Echo(ctx, m)
	if err != nil {
		log.Fatal(err)
	}

	if text, want := m.Value, res.Value; text != want {
		t.Errorf("text=%s; want %s", text, want)
	}
}

func TestConcurrency(t *testing.T) {
	size := 10
	var wg sync.WaitGroup
	wg.Add(size)
	for i := 0; i < size; i++ {
		go func() {
			defer wg.Done()
			cc, err := grpc.Dial("", grpc.WithInsecure(), testL.WithDialer())
			if err != nil {
				t.Fatal(err)
				return
			}
			c := pb.NewEchoServiceClient(cc)
			ctx := context.Background()
			m := &pb.Message{Value: "Test"}
			res, err := c.Echo(ctx, m)
			if err != nil {
				log.Fatal(err)
			}

			if text, want := m.Value, res.Value; text != want {
				t.Errorf("text=%s; want %s", text, want)
			}
		}()
	}
	wg.Wait()
}
