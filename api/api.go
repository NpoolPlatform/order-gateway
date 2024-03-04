package api

import (
	"context"

	order "github.com/NpoolPlatform/message/npool/order/gw/v1"

	order1 "github.com/NpoolPlatform/order-gateway/api/order"
	config1 "github.com/NpoolPlatform/order-gateway/api/simulate/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	order.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	order.RegisterGatewayServer(server, &Server{})
	order1.Register(server)
	config1.Register(server)
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := order.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	if err := order1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := config1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
