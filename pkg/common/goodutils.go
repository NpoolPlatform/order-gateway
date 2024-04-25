package common

import (
	"context"

	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"

	"github.com/google/uuid"
)

func GetGoods(ctx context.Context, goodIDs []string) (map[string]*goodmwpb.Good, error) {
	for _, goodID := range goodIDs {
		if _, err := uuid.Parse(goodID); err != nil {
			return nil, err
		}
	}

	goods, _, err := goodmwcli.GetGoods(ctx, &goodmwpb.Conds{
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
	}, int32(0), int32(len(goodIDs)))
	if err != nil {
		return nil, err
	}
	goodMap := map[string]*goodmwpb.Good{}
	for _, good := range goods {
		goodMap[good.EntID] = good
	}
	return goodMap, nil
}
