package api

import (
	"context"

	order "github.com/NpoolPlatform/message/npool/order/gw/v1"
	appconfig1 "github.com/NpoolPlatform/order-gateway/api/app/config"
	feeorder1 "github.com/NpoolPlatform/order-gateway/api/fee"
	powerrentalorder1 "github.com/NpoolPlatform/order-gateway/api/powerrental"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	order.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	order.RegisterGatewayServer(server, &Server{})
	feeorder1.Register(server)
	powerrentalorder1.Register(server)
	appconfig1.Register(server)
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := order.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	if err := feeorder1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := powerrentalorder1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := appconfig1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
