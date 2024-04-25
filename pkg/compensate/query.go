package compensate

import (
	"context"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	appmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/app"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/compensate"
	compensatemwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/compensate"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	compensatemwcli "github.com/NpoolPlatform/order-middleware/pkg/client/compensate"
)

type queryHandler struct {
	*Handler
	compensates []*compensatemwpb.Compensate
	infos       []*npool.Compensate
	apps        map[string]*appmwpb.App
	users       map[string]*usermwpb.User
	appGoods    map[string]*appgoodmwpb.Good
}

func (h *queryHandler) getApps(ctx context.Context) (err error) {
	h.apps, err = ordergwcommon.GetApps(ctx, func() (appIDs []string) {
		for _, compensate := range h.compensates {
			appIDs = append(appIDs, compensate.AppID)
		}
		return
	}())
	return err
}

func (h *queryHandler) getUsers(ctx context.Context) (err error) {
	h.users, err = ordergwcommon.GetUsers(ctx, func() (userIDs []string) {
		for _, compensate := range h.compensates {
			userIDs = append(userIDs, compensate.UserID)
		}
		return
	}())
	return err
}

func (h *queryHandler) getAppGoods(ctx context.Context) (err error) {
	h.appGoods, err = ordergwcommon.GetAppGoods(ctx, func() (appGoodIDs []string) {
		for _, compensate := range h.compensates {
			appGoodIDs = append(appGoodIDs, compensate.AppGoodID)
		}
		return
	}())
	return err
}

func (h *queryHandler) formalize() {
	for _, compensate := range h.compensates {
		app, ok := h.apps[compensate.AppID]
		if !ok {
			continue
		}
		user, ok := h.users[compensate.UserID]
		if !ok {
			continue
		}
		appGood, ok := h.appGoods[compensate.AppGoodID]
		if !ok {
			continue
		}
		h.infos = append(h.infos, &npool.Compensate{
			ID:               compensate.ID,
			EntID:            compensate.EntID,
			AppID:            compensate.AppID,
			AppName:          app.Name,
			UserID:           compensate.UserID,
			EmailAddress:     user.EmailAddress,
			PhoneNO:          user.PhoneNO,
			GoodID:           compensate.GoodID,
			GoodType:         compensate.GoodType,
			GoodName:         appGood.GoodName,
			AppGoodID:        compensate.AppGoodID,
			AppGoodName:      appGood.AppGoodName,
			OrderID:          compensate.OrderID,
			CompensateFromID: compensate.CompensateFromID,
			CompensateType:   compensate.CompensateType,
			CreatedAt:        compensate.CreatedAt,
			UpdatedAt:        compensate.UpdatedAt,
			// TODO: add compensate name
		})
	}
}

func (h *Handler) GetCompensates(ctx context.Context) ([]*npool.Compensate, uint32, error) {
	conds := &compensatemwpb.Conds{}
	if h.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	if h.GoodID != nil {
		conds.GoodID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.GoodID}
	}
	if h.AppGoodID != nil {
		conds.AppGoodID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID}
	}
	if h.OrderID != nil {
		conds.OrderID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderID}
	}

	infos, total, err := compensatemwcli.GetCompensates(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}

	handler := &queryHandler{
		Handler:     h,
		compensates: infos,
	}
	if err := handler.getApps(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getUsers(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.infos, total, nil
}
