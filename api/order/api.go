package order

import (
	"github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	order.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	order.RegisterGatewayServer(server, &Server{})
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return nil
}
