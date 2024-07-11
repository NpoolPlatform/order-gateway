package outofgas

import (
	"context"

	"github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental/outofgas"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	outofgas.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	outofgas.RegisterGatewayServer(server, &Server{})
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return outofgas.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts)
}
