package order

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

func GetOrder(ctx context.Context, id string) (*npool.Order, error) {
	ord, err := ordermwcli.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	o := &npool.Order{
		ID:     ord.ID,
		UserID: ord.UserID,
		GoodID: ord.GoodID,
		Units:  ord.Units,

		ParentOrderID:     ord.ParentOrderID,
		ParentOrderGoodID: ord.ParentOrderGoodID,

		PaymentID:               ord.PaymentID,
		PaymentCoinTypeID:       ord.PaymentCoinTypeID,
		PaymentCoinUSDCurrency:  ord.PaymentCoinUSDCurrency,
		PaymentLiveUSDCurrency:  ord.PaymentLiveCoinUSDCurrency,
		PaymentLocalUSDCurrency: ord.PaymentLocalCoinUSDCurrency,
		PaymentAmount:           ord.PaymentAmount,
		PayWithParent:           ord.PayWithParent,
		PayWithBalanceAmount:    ord.PayWithBalanceAmount,

		FixAmountID:    ord.FixAmountID,
		DiscountID:     ord.DiscountID,
		SpecialOfferID: ord.SpecialOfferID,

		CreatedAt: ord.CreatedAt,
		PaidAt:    ord.PaidAt,
		State:     ord.State,
	}

	return o, nil
}
