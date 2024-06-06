package compensate

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	compensatemwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/compensate"
	compensatemwcli "github.com/NpoolPlatform/order-middleware/pkg/client/compensate"
)

type checkHandler struct {
	*Handler
}

func (h *checkHandler) checkCompensate(ctx context.Context) error {
	exist, err := compensatemwcli.ExistCompensateConds(ctx, &compensatemwpb.Conds{
		ID:     &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.ID},
		EntID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID},
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.UserID},
	})
	if err != nil {
		return err
	}
	if !exist {
		return wlog.Errorf("invalid compensate")
	}
	return nil
}
