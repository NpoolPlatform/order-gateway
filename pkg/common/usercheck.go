package common

import (
	"context"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
)

type UserCheckHandler struct {
	AppCheckHandler
	UserID *string
}

func (h *UserCheckHandler) CheckUserWithUserID(ctx context.Context, userID string) error {
	exist, err := usermwcli.ExistUser(ctx, *h.AppID, userID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if !exist {
		return wlog.Errorf("invalid user")
	}
	return nil
}

func (h *UserCheckHandler) CheckUser(ctx context.Context) error {
	return h.CheckUserWithUserID(ctx, *h.UserID)
}
