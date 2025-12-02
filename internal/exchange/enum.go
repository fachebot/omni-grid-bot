package exchange

// Exchange values.
const (
	Lighter     string = "lighter"
	Paradex     string = "paradex"
	Variational string = "variational"
)

var (
	validExchanges = []string{Lighter, Paradex, Variational}
)

func IsValidExchanges(ex string) bool {
	for _, v := range validExchanges {
		if v == ex {
			return true
		}
	}
	return false
}
