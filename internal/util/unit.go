package util

import (
	"math/big"

	"github.com/shopspring/decimal"
)

func ParseUnits(value *big.Int, decimals uint8) decimal.Decimal {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromInt32(int32(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.DivRound(mul, int32(decimals)).Truncate(int32(decimals))
	return result
}

func FormatUnits(amount decimal.Decimal, decimals uint8) *big.Int {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromInt32(int32(decimals)))
	result := amount.Mul(mul)

	wei := big.NewInt(0)
	wei.SetString(result.String(), 10)
	return wei
}
