package order

import (
	"context"
	"fmt"

	payaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"
	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordercli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	"github.com/google/uuid"
)

type queryHandler struct {
	*Handler
	orders          []*ordermwpb.Order
	infos           []*npool.Order
	users           map[string]*usermwpb.User
	parentOrders    map[string]*ordermwpb.Order
	appGoods        map[string]*appgoodmwpb.Good
	parentAppGoods  map[string]*appgoodmwpb.Good
	accountPayments map[string]*payaccmwpb.Account
	coupons         map[string]*allocatedmwpb.Coupon
	coins           map[string]*appcoinmwpb.Coin
}

func (h *queryHandler) getUsers(ctx context.Context) error {
	uids := []string{}
	for _, ord := range h.orders {
		if _, err := uuid.Parse(ord.UserID); err != nil {
			continue
		}
		uids = append(uids, ord.UserID)
	}
	users, _, err := usermwcli.GetUsers(ctx, &usermwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: uids},
	}, 0, int32(len(uids)))
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("invalid users")
	}

	for _, user := range users {
		h.users[user.ID] = user
	}
	return nil
}

func (h *queryHandler) getAccountPayments(ctx context.Context) error {
	accIDs := []string{}
	for _, ord := range h.orders {
		if _, err := uuid.Parse(ord.PaymentAccountID); err != nil {
			continue
		}
		accIDs = append(accIDs, ord.PaymentAccountID)
	}

	accounts, _, err := payaccmwcli.GetAccounts(ctx, &payaccmwpb.Conds{
		AccountIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: accIDs},
	}, 0, int32(len(accIDs)))
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		h.accountPayments[acc.AccountID] = acc
	}
	return nil
}

func (h *queryHandler) getCoupons(ctx context.Context) error {
	ids := []string{}
	for _, ord := range h.orders {
		ids = append(ids, ord.CouponIDs...)
	}

	coupons, _, err := allocatedmwcli.GetCoupons(ctx, &allocatedmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
	}, int32(0), int32(len(ids)))
	if err != nil {
		return err
	}

	for _, coup := range coupons {
		h.coupons[coup.EntID] = coup
	}
	return nil
}

func (h *queryHandler) getParentOrders(ctx context.Context) error {
	ids := []string{}
	for _, ord := range h.orders {
		if ord.ParentOrderID != uuid.Nil.String() {
			ids = append(ids, ord.ParentOrderID)
		}
	}
	orders, _, err := ordercli.GetOrders(ctx, &ordermwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}
	for _, order := range orders {
		h.parentOrders[order.ID] = order
	}
	return nil
}

func (h *queryHandler) getAppGoods(ctx context.Context) error {
	goodIDs := []string{}
	for _, ord := range h.orders {
		if _, err := uuid.Parse(ord.AppGoodID); err != nil {
			continue
		}
		goodIDs = append(goodIDs, ord.AppGoodID)
	}

	appGoods, _, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return err
	}

	for _, appGood := range appGoods {
		h.appGoods[appGood.ID] = appGood
	}
	return nil
}

func (h *queryHandler) getParentAppGoods(ctx context.Context) error {
	goodIDs := []string{}
	for _, ord := range h.parentOrders {
		if _, err := uuid.Parse(ord.AppGoodID); err != nil {
			continue
		}
		goodIDs = append(goodIDs, ord.AppGoodID)
	}
	if len(goodIDs) == 0 {
		return nil
	}

	appGoods, _, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return err
	}

	for _, appGood := range appGoods {
		h.parentAppGoods[appGood.ID] = appGood
	}
	return nil
}

func (h *queryHandler) getCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, ord := range h.orders {
		if _, err := uuid.Parse(ord.PaymentCoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, ord.PaymentCoinTypeID)
	}
	for _, ord := range h.appGoods {
		if _, err := uuid.Parse(ord.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, ord.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}

	for _, coin := range coins {
		h.coins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *queryHandler) formalize(ctx context.Context) { //nolint
	for _, ord := range h.orders {
		info := &npool.Order{
			ID:                      ord.ID,
			AppID:                   ord.AppID,
			UserID:                  ord.UserID,
			GoodID:                  ord.GoodID,
			AppGoodID:               ord.AppGoodID,
			ParentOrderID:           ord.ParentOrderID,
			Units:                   ord.Units,
			GoodValue:               ord.GoodValue,
			GoodValueUSD:            ord.GoodValueUSD,
			UserSetCanceled:         ord.UserSetCanceled,
			AdminSetCanceled:        ord.AdminSetCanceled,
			PaymentID:               ord.PaymentID,
			PaymentCoinTypeID:       ord.PaymentCoinTypeID,
			PaymentCoinUSDCurrency:  ord.CoinUSDCurrency,
			PaymentLiveUSDCurrency:  ord.LiveCoinUSDCurrency,
			PaymentLocalUSDCurrency: ord.LocalCoinUSDCurrency,
			PaymentAmount:           ord.PaymentAmount,
			PaymentStartAmount:      ord.PaymentStartAmount,
			PaymentFinishAmount:     ord.PaymentFinishAmount,
			PayWithBalanceAmount:    ord.BalanceAmount,
			TransferAmount:          ord.TransferAmount,
			OrderType:               ord.OrderType,
			OrderState:              ord.OrderState,
			CancelState:             ord.CancelState,
			PaymentType:             ord.PaymentType,
			PaymentState:            ord.PaymentState,
			CreatedAt:               ord.CreatedAt,
			StartAt:                 ord.StartAt,
			EndAt:                   ord.EndAt,
			InvestmentType:          ord.InvestmentType,
			LastBenefitAt:           ord.LastBenefitAt,
			PaidAt:                  ord.PaidAt,
		}

		if user, ok := h.users[ord.UserID]; ok {
			info.EmailAddress = user.EmailAddress
			info.PhoneNO = user.PhoneNO
		}
		appGood, ok := h.appGoods[ord.AppGoodID]
		if !ok {
			continue
		}

		info.CoinTypeID = appGood.CoinTypeID
		info.GoodName = appGood.GoodName
		info.GoodUnit = appGood.Unit
		info.GoodServicePeriodDays = uint32(appGood.DurationDays)
		info.GoodUnitPrice = appGood.Price

		if coin, ok := h.coins[info.CoinTypeID]; ok {
			info.CoinName = coin.Name
			info.CoinLogo = coin.Logo
			info.CoinUnit = coin.Unit
			info.CoinPresale = coin.Presale
		}

		if coin, ok := h.coins[ord.PaymentCoinTypeID]; ok {
			info.PaymentCoinName = coin.Name
			info.PaymentCoinLogo = coin.Logo
			info.PaymentCoinUnit = coin.Unit
		}

		acc, ok := h.accountPayments[ord.PaymentAccountID]
		if ok {
			info.PaymentAddress = acc.Address
		}
		for _, id := range ord.CouponIDs {
			coup, ok := h.coupons[id]
			if !ok {
				continue
			}

			info.Coupons = append(info.Coupons, &npool.Coupon{
				CouponID:    id,
				CouponType:  coup.CouponType,
				CouponName:  coup.CouponName,
				CouponValue: coup.Denomination,
			})
		}
		if ord.ParentOrderID != uuid.Nil.String() {
			if porder, ok := h.parentOrders[ord.ParentOrderID]; ok {
				info.ParentOrderGoodID = porder.GoodID
				if pgood, ok := h.parentAppGoods[ord.AppGoodID]; ok {
					info.ParentOrderAppGoodID = pgood.ID
					info.ParentOrderGoodName = pgood.GoodName
				}
			}
		}
		if ord.PaymentType == types.PaymentType_PayWithParentOrder {
			info.PayWithParent = true
		}
		h.infos = append(h.infos, info)
	}
}

func (h *Handler) GetOrder(ctx context.Context) (*npool.Order, error) {
	order, err := ordercli.GetOrder(ctx, *h.ID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, err
	}
	if *h.AppID != order.AppID || *h.UserID != order.UserID {
		return nil, fmt.Errorf("permission denied")
	}

	handler := &queryHandler{
		Handler:         h,
		orders:          []*ordermwpb.Order{order},
		infos:           []*npool.Order{},
		users:           map[string]*usermwpb.User{},
		parentOrders:    map[string]*ordermwpb.Order{},
		parentAppGoods:  map[string]*appgoodmwpb.Good{},
		appGoods:        map[string]*appgoodmwpb.Good{},
		accountPayments: map[string]*payaccmwpb.Account{},
		coupons:         map[string]*allocatedmwpb.Coupon{},
		coins:           map[string]*appcoinmwpb.Coin{},
	}
	if err := handler.getUsers(ctx); err != nil {
		return nil, err
	}
	if err := handler.getParentOrders(ctx); err != nil {
		return nil, err
	}
	if err := handler.getParentAppGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAccountPayments(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoupons(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoins(ctx); err != nil {
		return nil, err
	}

	handler.formalize(ctx)
	if len(handler.infos) == 0 {
		return nil, nil
	}

	return handler.infos[0], nil
}

func (h *Handler) GetOrders(ctx context.Context) ([]*npool.Order, uint32, error) {
	conds := &ordermwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}

	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	if h.IDs != nil && len(h.IDs) != 0 {
		conds.IDs = &basetypes.StringSliceVal{Op: cruder.IN, Value: h.IDs}
	}
	ords, total, err := ordercli.GetOrders(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(ords) == 0 {
		return []*npool.Order{}, 0, nil
	}

	handler := &queryHandler{
		Handler:         h,
		orders:          ords,
		infos:           []*npool.Order{},
		users:           map[string]*usermwpb.User{},
		parentOrders:    map[string]*ordermwpb.Order{},
		parentAppGoods:  map[string]*appgoodmwpb.Good{},
		appGoods:        map[string]*appgoodmwpb.Good{},
		accountPayments: map[string]*payaccmwpb.Account{},
		coupons:         map[string]*allocatedmwpb.Coupon{},
		coins:           map[string]*appcoinmwpb.Coin{},
	}
	if err := handler.getUsers(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getParentOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getParentAppGoods(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAccountPayments(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getCoupons(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getCoins(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize(ctx)
	if len(handler.infos) == 0 {
		return nil, total, nil
	}

	return handler.infos, total, nil
}
