package compensate

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	malfunctionmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/malfunction"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	malfunctionmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/malfunction"
	powerrentalcompensatemwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental/compensate"
	powerrentalcompensatemwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental/compensate"

	"github.com/google/uuid"
)

type createHandler struct {
	*Handler
	goodMalfunction   *malfunctionmwpb.Malfunction
	compensateSeconds uint32
}

func (h *createHandler) getGoodMalfunction(ctx context.Context) (err error) {
	h.goodMalfunction, err = malfunctionmwcli.GetMalfunction(ctx, *h.CompensateFromID)
	if err != nil {
		return err
	}
	if h.goodMalfunction == nil || h.goodMalfunction.CompensateSeconds <= 0 {
		return wlog.Errorf("invalid goodmalfunction")
	}
	h.compensateSeconds = h.goodMalfunction.CompensateSeconds
	return nil
}

func (h *createHandler) getCompensateType(ctx context.Context) error {
	switch *h.CompensateType {
	case types.CompensateType_CompensateMalfunction:
		return h.getGoodMalfunction(ctx)
	case types.CompensateType_CompensateWalfare:
		fallthrough //nolint
	case types.CompensateType_CompensateStarterDelay:
		return wlog.Errorf("not implemented")
	default:
		return wlog.Errorf("invalid compensatetype")
	}
	return nil
}

func (h *Handler) CreateCompensate(ctx context.Context) error {
	handler := &createHandler{
		Handler: h,
	}
	if err := handler.getCompensateType(ctx); err != nil {
		return wlog.WrapError(err)
	}
	if h.EntID == nil {
		h.EntID = func() *string { s := uuid.NewString(); return &s }()
	}
	return powerrentalcompensatemwcli.CreateCompensate(ctx, &powerrentalcompensatemwpb.CompensateReq{
		EntID:             h.EntID,
		GoodID:            h.GoodID,
		OrderID:           h.OrderID,
		CompensateFromID:  h.CompensateFromID,
		CompensateType:    h.CompensateType,
		CompensateSeconds: &handler.compensateSeconds,
	})
}
