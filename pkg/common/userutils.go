package common

import (
	"context"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	"github.com/google/uuid"
)

func GetUsers(ctx context.Context, userIDs []string) (map[string]*usermwpb.User, error) {
	for _, userID := range userIDs {
		if _, err := uuid.Parse(userID); err != nil {
			return nil, wlog.WrapError(err)
		}
	}

	users, _, err := usermwcli.GetUsers(ctx, &usermwpb.Conds{
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: userIDs},
	}, 0, int32(len(userIDs)))
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	userMap := map[string]*usermwpb.User{}
	for _, user := range users {
		userMap[user.EntID] = user
	}
	return userMap, nil
}
