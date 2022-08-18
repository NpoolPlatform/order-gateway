package migrator

import (
	"context"

	"github.com/NpoolPlatform/cloud-hashing-order/pkg/db"
)

func Migrate(ctx context.Context) error {
	cli, err := db.Client()
	if err != nil {
		return err
	}

	orders, err := cli.
		Order.
		Query().
		All(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		_, err := cli.
			Order.
			UpdateOneID(order.ID).
			SetEnd(order.Start + 365*24*60*60).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
