package powerrental

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	paymentaccountmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	appmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/app"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	coinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin"
	topmostmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/topmost"
	apppowerrentalmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/powerrental"
	allocatedcouponmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	ordercoupongwpb "github.com/NpoolPlatform/message/npool/order/gw/v1/order/coupon"
	paymentgwpb "github.com/NpoolPlatform/message/npool/order/gw/v1/payment"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	powerrentalordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	powerrentalordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental"

	"github.com/google/uuid"
)

// TODO: expand all field

type queryHandler struct {
	*Handler
	powerRentalOrders []*powerrentalordermwpb.PowerRentalOrder
	infos             []*npool.PowerRentalOrder
	apps              map[string]*appmwpb.App
	users             map[string]*usermwpb.User
	topMosts          map[string]*topmostmwpb.TopMost
	allocatedCoupons  map[string]*allocatedcouponmwpb.Coupon
	coins             map[string]*coinmwpb.Coin
	paymentAccounts   map[string]*paymentaccountmwpb.Account
	appPowerRentals   map[string]*apppowerrentalmwpb.PowerRental
}

func (h *queryHandler) getApps(ctx context.Context) (err error) {
	h.apps, err = ordergwcommon.GetApps(ctx, func() (appIDs []string) {
		for _, powerRentalOrder := range h.powerRentalOrders {
			appIDs = append(appIDs, powerRentalOrder.AppID)
		}
		return
	}())
	return err
}

func (h *queryHandler) getUsers(ctx context.Context) (err error) {
	h.users, err = ordergwcommon.GetUsers(ctx, func() (userIDs []string) {
		for _, powerRentalOrder := range h.powerRentalOrders {
			userIDs = append(userIDs, powerRentalOrder.UserID)
		}
		return
	}())
	return err
}

func (h *queryHandler) getAppPowerRentals(ctx context.Context) (err error) {
	h.appPowerRentals, err = ordergwcommon.GetAppPowerRentals(ctx, func() (appGoodIDs []string) {
		for _, powerRentalOrder := range h.powerRentalOrders {
			appGoodIDs = append(appGoodIDs, powerRentalOrder.AppGoodID)
		}
		return
	}())
	return err
}

func (h *queryHandler) getTopMosts(ctx context.Context) (err error) {
	h.topMosts, err = ordergwcommon.GetTopMosts(ctx, func() (topMostIDs []string) {
		for _, powerRentalOrder := range h.powerRentalOrders {
			if _, err := uuid.Parse(powerRentalOrder.PromotionID); err != nil {
				continue
			}
			topMostIDs = append(topMostIDs, powerRentalOrder.PromotionID)
		}
		return
	}())
	return err
}

func (h *queryHandler) getAllocatedCoupons(ctx context.Context) (err error) {
	h.allocatedCoupons, err = ordergwcommon.GetAllocatedCoupons(ctx, func() (allocatedCouponIDs []string) {
		for _, powerRentalOrder := range h.powerRentalOrders {
			for _, coupon := range powerRentalOrder.Coupons {
				allocatedCouponIDs = append(allocatedCouponIDs, coupon.CouponID)
			}
		}
		return
	}())
	return err
}

func (h *queryHandler) getCoins(ctx context.Context) (err error) {
	h.coins, err = ordergwcommon.GetCoins(ctx, func() (coinTypeIDs []string) {
		for _, powerRentalOrder := range h.powerRentalOrders {
			for _, balance := range powerRentalOrder.PaymentBalances {
				coinTypeIDs = append(coinTypeIDs, balance.CoinTypeID)
			}
			for _, transfer := range powerRentalOrder.PaymentTransfers {
				coinTypeIDs = append(coinTypeIDs, transfer.CoinTypeID)
			}
		}
		return
	}())
	return err
}

func (h *queryHandler) getPaymentAccounts(ctx context.Context) (err error) {
	h.paymentAccounts, err = ordergwcommon.GetPaymentAccounts(ctx, func() (accountIDs []string) {
		for _, powerRentalOrder := range h.powerRentalOrders {
			for _, transfer := range powerRentalOrder.PaymentTransfers {
				accountIDs = append(accountIDs, transfer.CoinTypeID)
			}
		}
		return
	}())
	return err
}

func (h *queryHandler) formalize() {
	for _, powerRentalOrder := range h.powerRentalOrders {
		info := &npool.PowerRentalOrder{
			ID:             powerRentalOrder.ID,
			EntID:          powerRentalOrder.EntID,
			AppID:          powerRentalOrder.AppID,
			UserID:         powerRentalOrder.UserID,
			GoodID:         powerRentalOrder.GoodID,
			GoodType:       powerRentalOrder.GoodType,
			AppGoodID:      powerRentalOrder.AppGoodID,
			OrderID:        powerRentalOrder.OrderID,
			OrderType:      powerRentalOrder.OrderType,
			PaymentType:    powerRentalOrder.PaymentType,
			CreateMethod:   powerRentalOrder.CreateMethod,
			Simulate:       powerRentalOrder.Simulate,
			OrderState:     powerRentalOrder.OrderState,
			StartMode:      powerRentalOrder.StartMode,
			StartAt:        powerRentalOrder.StartAt,
			LastBenefitAt:  powerRentalOrder.LastBenefitAt,
			BenefitState:   powerRentalOrder.BenefitState,
			AppGoodStockID: powerRentalOrder.AppGoodStockID,
			// TODO: mining pool information
			Units:             powerRentalOrder.Units,
			GoodValueUSD:      powerRentalOrder.GoodValueUSD,
			PaymentAmountUSD:  powerRentalOrder.PaymentAmountUSD,
			DiscountAmountUSD: powerRentalOrder.DiscountAmountUSD,
			PromotionID:       powerRentalOrder.PromotionID,
			DurationSeconds:   powerRentalOrder.DurationSeconds,
			InvestmentType:    powerRentalOrder.InvestmentType,
			CancelState:       powerRentalOrder.CancelState,
			CanceledAt:        powerRentalOrder.CanceledAt,
			EndAt:             powerRentalOrder.EndAt,
			PaidAt:            powerRentalOrder.PaidAt,
			UserSetPaid:       powerRentalOrder.UserSetPaid,
			UserSetCanceled:   powerRentalOrder.UserSetCanceled,
			AdminSetCanceled:  powerRentalOrder.AdminSetCanceled,
			PaymentState:      powerRentalOrder.PaymentState,
			OutOfGasSeconds:   powerRentalOrder.OutOfGasSeconds,
			CompensateSeconds: powerRentalOrder.CompensateSeconds,
			// TODO: fee durations
			CreatedAt: powerRentalOrder.CreatedAt,
			UpdatedAt: powerRentalOrder.UpdatedAt,
		}
		app, ok := h.apps[powerRentalOrder.AppID]
		if ok {
			info.AppName = app.Name
		}
		user, ok := h.users[powerRentalOrder.UserID]
		if ok {
			info.EmailAddress = user.EmailAddress
			info.PhoneNO = user.PhoneNO
		}
		appPowerRental, ok := h.appPowerRentals[powerRentalOrder.AppGoodID]
		if ok {
			info.GoodName = appPowerRental.GoodName
			info.AppGoodName = appPowerRental.AppGoodName
			info.DurationDisplayType = appPowerRental.DurationDisplayType
			info.Durations, info.DurationUnit = ordergwcommon.GoodDurationDisplayType2Unit(
				appPowerRental.DurationDisplayType, info.DurationSeconds,
			)
			info.BenefitType = appPowerRental.BenefitType
		}
		topMost, ok := h.topMosts[powerRentalOrder.PromotionID]
		if ok {
			info.TopMostTitle = topMost.Title
			info.TopMostTargetUrl = topMost.TargetUrl
		}
		for _, coupon := range powerRentalOrder.Coupons {
			orderCoupon := &ordercoupongwpb.OrderCouponInfo{
				AllocatedCouponID: coupon.CouponID,
				CreatedAt:         coupon.CreatedAt,
			}
			allocatedCoupon, ok := h.allocatedCoupons[coupon.CouponID]
			if ok {
				orderCoupon.CouponType = allocatedCoupon.CouponType
				orderCoupon.Denomination = allocatedCoupon.Denomination
				orderCoupon.CouponName = allocatedCoupon.CouponName
			}
			info.Coupons = append(info.Coupons, orderCoupon)
		}
		for _, balance := range powerRentalOrder.PaymentBalances {
			paymentBalance := &paymentgwpb.PaymentBalanceInfo{
				CoinTypeID:      balance.CoinTypeID,
				Amount:          balance.Amount,
				CoinUSDCurrency: balance.CoinUSDCurrency,
				CreatedAt:       balance.CreatedAt,
			}
			coin, ok := h.coins[balance.CoinTypeID]
			if ok {
				paymentBalance.CoinName = coin.Name
				paymentBalance.CoinUnit = coin.Unit
				paymentBalance.CoinLogo = coin.Logo
				paymentBalance.CoinENV = coin.ENV
			}
			info.PaymentBalances = append(info.PaymentBalances, paymentBalance)
		}
		for _, transfer := range powerRentalOrder.PaymentTransfers {
			paymentTransfer := &paymentgwpb.PaymentTransferInfo{
				CoinTypeID:      transfer.CoinTypeID,
				Amount:          transfer.Amount,
				AccountID:       transfer.AccountID,
				CoinUSDCurrency: transfer.CoinUSDCurrency,
				CreatedAt:       transfer.CreatedAt,
			}
			coin, ok := h.coins[transfer.CoinTypeID]
			if ok {
				paymentTransfer.CoinName = coin.Name
				paymentTransfer.CoinUnit = coin.Unit
				paymentTransfer.CoinLogo = coin.Logo
				paymentTransfer.CoinENV = coin.ENV
			}
			account, ok := h.paymentAccounts[transfer.AccountID]
			if ok {
				paymentTransfer.Address = account.Address
			}
			info.PaymentTransfers = append(info.PaymentTransfers, paymentTransfer)
		}
		h.infos = append(h.infos, info)
	}
}

func (h *Handler) GetPowerRentalOrder(ctx context.Context) (*npool.PowerRentalOrder, error) {
	if err := h.CheckOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	info, err := powerrentalordermwcli.GetPowerRentalOrder(ctx, *h.OrderID)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	if info == nil {
		return nil, wlog.Errorf("invalid powerrentalorder")
	}

	handler := &queryHandler{
		Handler:           h,
		powerRentalOrders: []*powerrentalordermwpb.PowerRentalOrder{info},
	}

	if err := handler.getApps(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getUsers(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getAppPowerRentals(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getTopMosts(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getCoins(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getPaymentAccounts(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}

	handler.formalize()
	if len(handler.infos) == 0 {
		return nil, wlog.Errorf("invalid order")
	}

	return handler.infos[0], nil
}

func (h *Handler) GetPowerRentalOrders(ctx context.Context) ([]*npool.PowerRentalOrder, uint32, error) {
	conds := &powerrentalordermwpb.Conds{}
	if h.OrderCheckHandler.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.AppID}
	}
	if h.OrderCheckHandler.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.UserID}
	}
	if h.AppGoodID != nil {
		conds.AppGoodID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID}
	}
	if len(h.OrderIDs) > 0 {
		conds.OrderIDs = &basetypes.StringSliceVal{Op: cruder.IN, Value: h.OrderIDs}
	}
	infos, total, err := powerrentalordermwcli.GetPowerRentalOrders(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, wlog.WrapError(err)
	}

	handler := &queryHandler{
		Handler:           h,
		powerRentalOrders: infos,
	}

	if err := handler.getApps(ctx); err != nil {
		return nil, 0, wlog.WrapError(err)
	}
	if err := handler.getUsers(ctx); err != nil {
		return nil, 0, wlog.WrapError(err)
	}
	if err := handler.getAppPowerRentals(ctx); err != nil {
		return nil, 0, wlog.WrapError(err)
	}
	if err := handler.getTopMosts(ctx); err != nil {
		return nil, 0, wlog.WrapError(err)
	}
	if err := handler.getCoins(ctx); err != nil {
		return nil, 0, wlog.WrapError(err)
	}
	if err := handler.getPaymentAccounts(ctx); err != nil {
		return nil, 0, wlog.WrapError(err)
	}

	handler.formalize()

	return handler.infos, total, nil
}
