package common

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	allocatedcouponmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	allocatedcouponmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
)

type AllocatedCouponCheckHandler struct {
	UserCheckHandler
	AllocatedCouponID *string
}

func (h *AllocatedCouponCheckHandler) CheckAllocatedCouponWithAllocatedCouponID(ctx context.Context, allocatedCouponID string) error {
	// TODO: should be replaced with exist api
	info, err := allocatedcouponmwcli.GetCouponOnly(ctx, &allocatedcouponmwpb.Conds{
		EntID:  &basetypes.StringVal{Op: cruder.EQ, Value: allocatedCouponID},
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
	})
	if err != nil {
		return wlog.WrapError(err)
	}
	if info == nil {
		return wlog.Errorf("invalid allocatedcoupon")
	}
	return nil
}
