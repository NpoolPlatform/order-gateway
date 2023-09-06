package order

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	payaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"
	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordercli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	appgoodscli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	appgoodsmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	"github.com/google/uuid"
)

type queryHandler struct {
	*Handler
	orders          []*ordermwpb.Order
	infos           []*npool.Order
	users           map[string]*usermwpb.User
	parentOrders    map[string]*ordermwpb.Order
	appGoods        map[string]*appgoodsmwpb.Good
	parentAppGoods  map[string]*appgoodsmwpb.Good
	accountPayments map[string]*payaccmwpb.Account
	coupons         map[string]*allocatedmwpb.Coupon
	coins           map[string]*appcoinmwpb.Coin
}

var invalidID = uuid.UUID{}.String()

func (h *queryHandler) getUsers(ctx context.Context) error {
	uids := []string{}
	for _, ord := range h.orders {
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
		AccountIDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: accIDs,
		},
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
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
	}, int32(0), int32(len(ids)))
	if err != nil {
		return err
	}

	for _, coup := range coupons {
		h.coupons[coup.ID] = coup
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
	}, h.Offset, h.Limit)
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
	for _, val := range h.orders {
		goodIDs = append(goodIDs, val.GetGoodID())
	}

	appGoods, _, err := appgoodscli.GetGoods(ctx, &appgoodsmwpb.Conds{
		GoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
		AppID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return err
	}

	for _, appGood := range appGoods {
		h.appGoods[appGood.AppID+appGood.GoodID] = appGood
	}
	return nil
}

func (h *queryHandler) getParentAppGoods(ctx context.Context) error {
	goodIDs := []string{}
	for _, val := range h.parentOrders {
		goodIDs = append(goodIDs, val.GetGoodID())
	}
	if len(goodIDs) == 0 {
		return nil
	}

	appGoods, _, err := appgoodscli.GetGoods(ctx, &appgoodsmwpb.Conds{
		GoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
		AppID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return err
	}

	for _, appGood := range appGoods {
		h.parentAppGoods[appGood.AppID+appGood.GoodID] = appGood
	}
	return nil
}

func (h *queryHandler) getCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, val := range h.orders {
		if _, err := uuid.Parse(val.PaymentCoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.PaymentCoinTypeID)
	}
	for _, val := range h.appGoods {
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
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
	infos := []*npool.Order{}

	for _, ord := range h.orders {
		info := &npool.Order{
			ID:                      ord.ID,
			AppID:                   ord.AppID,
			UserID:                  ord.UserID,
			GoodID:                  ord.GoodID,
			ParentOrderID:           ord.ParentOrderID,
			Units:                   ord.Units,
			GoodValue:               ord.GoodValue,
			PaymentID:               ord.PaymentID,
			PaymentCoinTypeID:       ord.PaymentCoinTypeID,
			PaymentCoinUSDCurrency:  ord.CoinUSDCurrency,
			PaymentLiveUSDCurrency:  ord.LiveCoinUSDCurrency,
			PaymentLocalUSDCurrency: ord.LocalCoinUSDCurrency,
			PaymentAmount:           ord.PaymentAmount,
			PaymentStartAmount:      ord.PaymentStartAmount,
			PaymentFinishAmount:     ord.PaymentFinishAmount,
			PayWithBalanceAmount:    ord.BalanceAmount,
			OrderType:               ord.OrderType,
			PaymentType:             ord.PaymentType,
			CreatedAt:               ord.CreatedAt,
			State:                   ord.OrderState,
			StartAt:                 ord.StartAt,
			EndAt:                   ord.EndAt,
			InvestmentType:          ord.InvestmentType,
			LastBenefitAt:           ord.LastBenefitAt,
			PaidAt:                  ord.PaidAt,
		}

		user, ok := h.users[ord.UserID]
		if !ok {
			logger.Sugar().Warnw("expand", "UserID", ord.UserID, "OrderID", ord.ID)
		}

		if user != nil {
			info.EmailAddress = user.EmailAddress
			info.PhoneNO = user.PhoneNO
		}

		appGood, ok := h.appGoods[ord.AppID+ord.GoodID]
		if !ok {
			logger.Sugar().Warnw("expand", "AppID", ord.AppID, "GoodID", ord.GoodID)
			continue
		}

		info.CoinTypeID = appGood.CoinTypeID
		info.GoodName = appGood.GoodName
		info.GoodUnit = appGood.Unit
		info.GoodServicePeriodDays = uint32(appGood.DurationDays)
		info.GoodUnitPrice = appGood.Price

		coin, ok := h.coins[info.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("expand", "AppID", info.AppID, "CoinTypeID", info.CoinTypeID)
			continue
		}

		info.CoinName = coin.Name
		info.CoinLogo = coin.Logo
		info.CoinUnit = coin.Unit
		info.CoinPresale = coin.Presale

		if ord.PaymentID != invalidID && ord.PaymentID != "" {
			coin, ok = h.coins[ord.PaymentCoinTypeID]
			if !ok {
				logger.Sugar().Warnw("expand", "AppID", info.AppID, "PaymentCoinTypeID", info.PaymentCoinTypeID)
				continue
			}
		}

		if coin != nil {
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
			porder, ok := h.parentOrders[ord.ParentOrderID]
			if ok {
				info.ParentOrderGoodID = porder.GoodID
				pgood, ok := h.parentAppGoods[ord.AppID+porder.GoodID]
				if ok {
					info.ParentOrderGoodName = pgood.GoodName
				}
			}
		}

		if ord.PaymentType == types.PaymentType_PayWithParentOrder {
			info.PayWithParent = true
		}

		infos = append(infos, info)
	}
	h.infos = infos
}

func (h *Handler) GetOrder(ctx context.Context) (*npool.Order, error) {
	order, err := ordercli.GetOrder(ctx, *h.ID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, err
	}

	handler := &queryHandler{
		Handler:         h,
		orders:          []*ordermwpb.Order{order},
		infos:           []*npool.Order{},
		users:           map[string]*usermwpb.User{},
		parentOrders:    map[string]*ordermwpb.Order{},
		appGoods:        map[string]*appgoodsmwpb.Good{},
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
		appGoods:        map[string]*appgoodsmwpb.Good{},
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
