package common

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	coinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"
)

var coinCheckMap = map[string]func(string) error{
	"ironfish": func(address string) error {
		const ironfishAddrLen = 64
		if len(address) != ironfishAddrLen {
			return fmt.Errorf("invalid address")
		}
		if _, err := hex.DecodeString(address); err != nil {
			return err
		}
		return nil
	},
}

func getCoinName(targetCoinName string) string {
	for coinName := range coinCheckMap {
		contains := strings.Contains(targetCoinName, coinName)
		if contains {
			return coinName
		}
	}
	return ""
}

func ValidateAddress(targetCoinName, address string) error {
	coinName := getCoinName(targetCoinName)
	if coinName == "" {
		return nil
	}
	return coinCheckMap[coinName](address)
}

func CheckAddress(ctx context.Context, coinTypeID, address string) error {
	coin, err := coinmwcli.GetCoin(ctx, coinTypeID)
	if err != nil {
		return err
	}
	if coin == nil {
		return fmt.Errorf("invalid coin")
	}

	if coin.CheckNewAddressBalance {
		err := ValidateAddress(coin.Name, address)
		if err != nil {
			return fmt.Errorf("invalid %v address", coin.Name)
		}
		return nil
	}

	logger.Sugar().Error("ssssssssssssssssssssssssssssssssssssssssssssssssss1")

	bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    coin.Name,
		Address: address,
	})

	logger.Sugar().Error(coin.Name, address)
	logger.Sugar().Error("ssssssssssssssssssssssssssssssssssssssssssssssssss2")

	if err != nil {
		return err
	}
	if bal == nil {
		return fmt.Errorf("invalid address")
	}

	return nil
}
