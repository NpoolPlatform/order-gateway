package appconfig

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
)

type checkHandler struct {
	*Handler
}

func (h *checkHandler) checkAppConfig(ctx context.Context) error {
	exist, err := appconfigmwcli.ExistAppConfigConds(ctx, &appconfigmwpb.Conds{
		ID:    &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.ID},
		EntID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID},
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	})
	if err != nil {
		return wlog.WrapError(err)
	}
	if !exist {
		return wlog.Errorf("invalid appconfig")
	}
	return nil
}
