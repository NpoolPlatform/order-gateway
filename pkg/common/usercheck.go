package common

import (
	"context"
	"fmt"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
)

type UserCheckHandler struct {
	AppCheckHandler
	UserID *string
}

func (h *UserCheckHandler) CheckUserWithUserID(ctx context.Context, userID string) error {
	exist, err := usermwcli.ExistUser(ctx, *h.AppID, userID)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("invalid user")
	}
	return nil
}

func (h *UserCheckHandler) CheckUser(ctx context.Context) error {
	return h.CheckUserWithUserID(ctx, *h.UserID)
}
