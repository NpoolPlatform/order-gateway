//nolint
package migrator

import (
	"context"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/NpoolPlatform/order-manager/pkg/db/ent"

	"github.com/shopspring/decimal"

	"github.com/NpoolPlatform/order-manager/pkg/db"
)

func Migrate(ctx context.Context) error {
	var err error

	if err := db.Init(); err != nil {
		return err
	}
	logger.Sugar().Infow("Migrate order", "Start", "...")
	defer func() {
		logger.Sugar().Infow("Migrate order", "Done", "...", "error", err)
	}()

	return db.WithClient(ctx, func(_ctx context.Context, cli *ent.Client) error {
		infos, err := cli.
			Order.
			Query().
			All(_ctx)
		if err != nil {
			return err
		}

		for _, info := range infos {
			if info.Units == 0 {
				continue
			}
			units := decimal.NewFromInt32(int32(info.Units))
			_, err := cli.
				Order.
				UpdateOneID(info.ID).
				SetUnitsV1(units).
				Save(_ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
