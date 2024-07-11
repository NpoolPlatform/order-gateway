package outofgas

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	outofgasgwpb "github.com/NpoolPlatform/message/npool/order/gw/v1/outofgas"
	outofgas1 "github.com/NpoolPlatform/order-gateway/pkg/outofgas"
	powerrentaloutofgasmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental/outofgas"
)

func (h *Handler) DeleteOutOfGas(ctx context.Context) (*outofgasgwpb.OutOfGas, error) {
	h1, err := outofgas1.NewHandler(
		ctx,
		outofgas1.WithEntID(h.EntID, true),
	)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	info, err := h1.GetOutOfGas(ctx)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	if info == nil {
		return nil, wlog.Errorf("invalid outofgas")
	}
	if err := powerrentaloutofgasmwcli.DeleteOutOfGas(ctx, &info.ID, &info.EntID); err != nil {
		return nil, wlog.WrapError(err)
	}
	return info, nil
}
