package format

import (
	"github.com/dustin/go-humanize"
	"github.com/shopspring/decimal"
)

func Amount(input decimal.Decimal, precision int32) string {
	value, prefix := humanize.ComputeSI(input.InexactFloat64())
	return decimal.NewFromFloat(value).Truncate(precision).String() + prefix
}
