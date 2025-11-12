package strategy

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/model"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/shopspring/decimal"
)

const (
	MaxGridNumLimit = 150
)

func InitGridStrategy(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) error {
	// 获取精度信息
	mm, err := helper.GetMarketMetadata(ctx, svcCtx, record.Exchange, record.Symbol)
	if err != nil {
		return err
	}

	initialOrderSize := record.InitialOrderSize.Truncate(int32(mm.SupportedSizeDecimals))
	if initialOrderSize.LessThan(mm.MinBaseAmount) {
		return errors.New("order size too small")
	}

	// 生成价格列表
	var prices []decimal.Decimal
	switch record.QuantityMode {
	case strategy.QuantityModeGeometric:
		prices, err = GenerateGeometricGrid(record.PriceLower, record.PriceUpper, record.GridNum, int32(mm.SupportedPriceDecimals))
	case strategy.QuantityModeArithmetic:
		prices, err = GenerateArithmeticGrid(record.PriceLower, record.PriceUpper, record.GridNum, int32(mm.SupportedPriceDecimals))
	default:
		return errors.New("invalid quantity mode")
	}
	if err != nil {
		return err
	}

	// 初始交易账户
	adapter, err := helper.NewExchangeAdapterFromStrategy(svcCtx, record)
	if err != nil {
		return err
	}

	// 设置杠杆倍数
	var marginMode exchange.MarginMode
	switch record.MarginMode {
	case strategy.MarginModeCross:
		marginMode = exchange.MarginModeCross
	case strategy.MarginModeIsolated:
		marginMode = exchange.MarginModeIsolated
	default:
		return errors.New("invalid margin mode")
	}
	err = adapter.UpdateLeverage(ctx, record.Symbol, uint(record.Leverage), marginMode)
	if err != nil {
		return err
	}

	// 生成网格数据
	gridLevels := make([]ent.Grid, 0, len(prices))
	for level, price := range prices {
		item := ent.Grid{
			StrategyId: record.GUID,
			Exchange:   record.Exchange,
			Symbol:     record.Symbol,
			Account:    record.Account,
			Level:      level,
			Price:      price,
			Quantity:   initialOrderSize,
		}
		gridLevels = append(gridLevels, item)
	}

	// 创建网格仓位
	indexMap := make(map[int]int, 0)
	orderParamsList := make([]helper.CreateLimitOrderParams, 0)
	switch record.Mode {
	case strategy.ModeLong:
		for idx := 0; idx < len(gridLevels)-1; idx++ {
			indexMap[len(orderParamsList)] = idx
			orderParamsList = append(orderParamsList, helper.CreateLimitOrderParams{
				Symbol:     record.Symbol,
				IsAsk:      false,
				ReduceOnly: false,
				Price:      gridLevels[idx].Price,
				Size:       gridLevels[idx].Quantity,
			})

		}
	case strategy.ModeShort:
		for idx := len(gridLevels) - 1; idx > 0; idx-- {
			indexMap[len(orderParamsList)] = idx
			orderParamsList = append(orderParamsList, helper.CreateLimitOrderParams{
				Symbol:     record.Symbol,
				IsAsk:      true,
				ReduceOnly: false,
				Price:      gridLevels[idx].Price,
				Size:       gridLevels[idx].Quantity,
			})
		}
	default:
		return errors.New("invalid grid mode")
	}
	orderIds, _, err := adapter.CreateOrderBatch(ctx, orderParamsList, nil)
	if err != nil {
		return err
	}

	// 更新订单ID
	for idx := range orderIds {
		switch record.Mode {
		case strategy.ModeLong:
			gridLevels[indexMap[idx]].BuyClientOrderId = &orderIds[idx]
		case strategy.ModeShort:
			gridLevels[indexMap[idx]].SellClientOrderId = &orderIds[idx]
		}
	}

	// 更新策略状态
	return util.Tx(ctx, svcCtx.DbClient, func(tx *ent.Tx) error {
		m := model.NewGridModel(tx.Grid)
		err := m.DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		if err = m.CreateBulk(ctx, gridLevels); err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatus(ctx, record.ID, strategy.StatusActive)
	})
}

func calculateGeometricRatio(lower, upper decimal.Decimal, gridNum int) decimal.Decimal {
	ratio := upper.Div(lower)
	ratioFloat, _ := ratio.Float64()

	rFloat := math.Pow(ratioFloat, 1.0/float64(gridNum))
	return decimal.NewFromFloat(rFloat)
}

func GenerateGeometricGrid(priceLower, priceUpper decimal.Decimal, gridNum int, priceDecimals int32) ([]decimal.Decimal, error) {
	// 参数验证
	if gridNum <= 0 {
		return nil, errors.New("gridNum must be positive")
	}

	if gridNum > MaxGridNumLimit {
		return nil, errors.New("gridNum exceeds maximum limit")
	}
	if priceLower.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("priceLower must be positive for geometric grid")
	}
	if priceLower.GreaterThanOrEqual(priceUpper) {
		return nil, errors.New("priceLower must be less than priceUpper")
	}
	if priceDecimals < 0 {
		return nil, errors.New("priceDecimals must be non-negative")
	}

	// 截断价格边界
	lowerTruncated := priceLower.Truncate(priceDecimals)
	upperTruncated := priceUpper.Truncate(priceDecimals)

	if lowerTruncated.GreaterThanOrEqual(upperTruncated) {
		return nil, errors.New("price range too small for given decimals")
	}

	// 使用二分法计算公比
	r := calculateGeometricRatio(lowerTruncated, upperTruncated, gridNum)

	// 生成网格档位
	levels := make([]decimal.Decimal, 0, gridNum+1)
	levels = append(levels, lowerTruncated)

	currentPrice := lowerTruncated
	lastPrice := lowerTruncated

	for i := 1; i < gridNum; i++ {
		currentPrice = currentPrice.Mul(r)
		truncatedPrice := currentPrice.Truncate(priceDecimals)

		// 确保价格递增
		if truncatedPrice.LessThanOrEqual(lastPrice) {
			// 如果截断后价格没有增加，强制增加最小单位
			minStep := decimal.New(1, -priceDecimals)
			truncatedPrice = lastPrice.Add(minStep)
		}

		// 检查是否超过上限
		if truncatedPrice.GreaterThanOrEqual(upperTruncated) {
			break
		}

		levels = append(levels, truncatedPrice)
		lastPrice = truncatedPrice
	}

	// 添加上限
	if upperTruncated.GreaterThan(levels[len(levels)-1]) {
		levels = append(levels, upperTruncated)
	}

	return levels, nil
}

func GenerateArithmeticGrid(priceLower, priceUpper decimal.Decimal, gridNum int, priceDecimals int32) ([]decimal.Decimal, error) {
	// 参数验证
	if gridNum <= 0 {
		return nil, errors.New("gridNum must be positive")
	}
	if gridNum > MaxGridNumLimit {
		return nil, errors.New("gridNum exceeds maximum limit")
	}
	if priceLower.GreaterThanOrEqual(priceUpper) {
		return nil, errors.New("priceLower must be less than priceUpper")
	}
	if priceDecimals < 0 {
		return nil, errors.New("priceDecimals must be non-negative")
	}

	// 截断价格边界
	lowerTruncated := priceLower.Truncate(priceDecimals)
	upperTruncated := priceUpper.Truncate(priceDecimals)

	// 检查截断后的价格范围是否有效
	if lowerTruncated.GreaterThanOrEqual(upperTruncated) {
		return nil, errors.New("price range too small for given decimals")
	}

	// 计算价格间隔
	priceRange := upperTruncated.Sub(lowerTruncated)
	step := priceRange.Div(decimal.NewFromInt(int64(gridNum)))

	// 检查步长是否太小
	minStep := decimal.New(1, -priceDecimals)
	if step.LessThan(minStep) {
		fmt.Println(step, minStep)
		return nil, errors.New("price range too small for the number of grids")
	}

	// 生成网格档位
	levels := make([]decimal.Decimal, 0, gridNum+1)

	// 添加第一个档位（下限）
	levels = append(levels, lowerTruncated)

	// 生成中间档位
	for i := 1; i < gridNum; i++ {
		price := lowerTruncated.Add(step.Mul(decimal.NewFromInt(int64(i))))
		truncatedPrice := price.Truncate(priceDecimals)

		// 确保价格递增且不超过上限
		if truncatedPrice.GreaterThan(levels[len(levels)-1]) && truncatedPrice.LessThan(upperTruncated) {
			levels = append(levels, truncatedPrice)
		}
	}

	// 添加最后一个档位（上限）
	if upperTruncated.GreaterThan(levels[len(levels)-1]) {
		levels = append(levels, upperTruncated)
	}

	return levels, nil
}
