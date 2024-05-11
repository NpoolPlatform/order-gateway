package api

import (
	"context"

	order "github.com/NpoolPlatform/message/npool/order/gw/v1"
	appconfig1 "github.com/NpoolPlatform/order-gateway/api/app/config"
	compensate1 "github.com/NpoolPlatform/order-gateway/api/compensate"
	feeorder1 "github.com/NpoolPlatform/order-gateway/api/fee"
	order1 "github.com/NpoolPlatform/order-gateway/api/order"
	ordercoupon1 "github.com/NpoolPlatform/order-gateway/api/order/coupon"
	outofgas1 "github.com/NpoolPlatform/order-gateway/api/outofgas"
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
	order1.Register(server)
	ordercoupon1.Register(server)
	compensate1.Register(server)
	outofgas1.Register(server)
	powerrentalorder1.Register(server)
	appconfig1.Register(server)
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := order.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	if err := order1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := ordercoupon1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := feeorder1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := compensate1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := outofgas1.RegisterGateway(mux, endpoint, opts); err != nil {
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
