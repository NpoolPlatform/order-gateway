//nolint:dupl
package common

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	orderusermwpb "github.com/NpoolPlatform/message/npool/miningpool/mw/v1/orderuser"
	orderusermwcli "github.com/NpoolPlatform/miningpool-middleware/pkg/client/orderuser"
	"github.com/google/uuid"
)

func GetMiningPoolOrderUsers(ctx context.Context, orderuserIDs []string) (map[string]*orderusermwpb.OrderUser, error) {
	for _, orderuserID := range orderuserIDs {
		if _, err := uuid.Parse(orderuserID); err != nil {
			return nil, wlog.WrapError(err)
		}
	}

	coins, _, err := orderusermwcli.GetOrderUsers(ctx, &orderusermwpb.Conds{
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: orderuserIDs},
	}, int32(0), int32(len(orderuserIDs)))
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	orderuserMap := map[string]*orderusermwpb.OrderUser{}
	for _, coin := range coins {
		orderuserMap[coin.EntID] = coin
	}
	return orderuserMap, nil
}
