package common

import (
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
)

func ValidateAdminCreateOrderType(orderType types.OrderType) error {
	switch orderType {
	case types.OrderType_Offline:
	case types.OrderType_Airdrop:
	default:
		return wlog.Errorf("invalid ordertype")
	}
	return nil
}
