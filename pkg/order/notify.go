package order

import (
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/NpoolPlatform/go-service-framework/pkg/pubsub"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
)

func notifyCouponsUsed(coupons map[string]*allocatedmwpb.Coupon, orderID *string) {
	if len(coupons) == 0 {
		return
	}

	reqs := []*allocatedmwpb.CouponReq{}
	used := true
	for _, coup := range coupons {
		reqs = append(reqs, &allocatedmwpb.CouponReq{
			ID:            &coup.ID,
			Used:          &used,
			UsedByOrderID: orderID,
		})
	}
	if err := pubsub.WithPublisher(func(publisher *pubsub.Publisher) error {
		return publisher.Update(
			basetypes.MsgID_UpdateCouponsUsedReq.String(),
			nil,
			nil,
			nil,
			reqs,
		)
	}); err != nil {
		logger.Sugar().Errorw(
			"notifyCouponsUsed",
			"reqs", reqs,
			"Error", err,
		)
	}
}
