package exchange

// Exchange values.
const (
	Lighter string = "lighter"
	Paradex string = "paradex"
)

var (
	validExchanges = [2]string{Lighter, Paradex}
)

func IsValidExchanges(ex string) bool {
	for _, v := range validExchanges {
		if v == ex {
			return true
		}
	}
	return false
}
