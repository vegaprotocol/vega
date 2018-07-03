package main

import (
	"flag"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"

	pb "vega/services/trading"
)

var (
	tls          = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile     = flag.String("cert_file", "", "The TLS cert file")
	keyFile      = flag.String("key_file", "", "The TLS key file")
	jsonDBFile   = flag.String("json_db_file", "testdata/route_guide_db.json", "A json file containing a list of features")
	port         = flag.Int("port", 5678, "The server port")
	echoEndpoint = flag.String("vega", "localhost:5678", "endpoint of YourService")
)

type tradingServer struct {
}

func (ts *tradingServer) CreateOrder(ctx context.Context, order *pb.Order) (*pb.OrderResponse, error) {
	fmt.Println(order.Market)
	return &pb.OrderResponse{Works: true}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println(lis)

	// Normal GRPC thing
	grpcServer := grpc.NewServer()
	pb.RegisterTradingServer(grpcServer, &tradingServer{})
	go grpcServer.Serve(lis)

	// JSON REST thing
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err2 := pb.RegisterTradingHandlerFromEndpoint(ctx, mux, *echoEndpoint, opts)
	if err2 != nil {
		fmt.Println(err2)
	}

	go http.ListenAndServe(":8080", mux)
}
