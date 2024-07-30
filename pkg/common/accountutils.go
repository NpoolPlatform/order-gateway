package common

import (
	"context"

	orderbenefitmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/orderbenefit"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	orderbenefitmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/orderbenefit"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

func GetOrderBenefits(ctx context.Context, orderIDs []string) (map[string][]*orderbenefitmwpb.Account, error) {
	orderBenefitMap := make(map[string][]*orderbenefitmwpb.Account)
	for _, orderID := range orderIDs {
		infos, _, err := orderbenefitmwcli.GetAccounts(ctx, &orderbenefitmwpb.Conds{
			OrderID: &basetypes.StringVal{
				Op:    cruder.EQ,
				Value: orderID,
			},
		}, 0, 0)
		if err != nil {
			return nil, wlog.WrapError(err)
		}
		orderBenefitMap[orderID] = infos
	}

	return orderBenefitMap, nil
}
