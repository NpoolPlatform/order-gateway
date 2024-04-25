package common

import (
	"context"
	"fmt"

	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
)

type AppGoodCheckHandler struct {
	AppUserCheckHandler
	GoodCheckHandler
	AppGoodID *string
}

func (h *AppGoodCheckHandler) CheckAppGoodWithAppGoodID(ctx context.Context, appGoodID string) error {
	exist, err := appgoodmwcli.ExistGoodConds(ctx, &appgoodmwpb.Conds{
		EntID: &basetypes.StringVal{Op: cruder.EQ, Value: appGoodID},
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	})
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("invalid appgood")
	}
	return nil
}

func (h *AppGoodCheckHandler) CheckAppGood(ctx context.Context) error {
	return h.CheckAppGoodWithAppGoodID(ctx, *h.AppGoodID)
}
