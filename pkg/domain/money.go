package domain

// Money represents a monetary amount with a currency.
type Money struct {
	Amount   float64
	Currency string
}

// NewMoney creates a new Money value object, validating the currency and amount.
func NewMoney(
	amount float64,
	currency string,
) (
	money Money,
	err error,
) {
	if !IsValidCurrencyCode(currency) {
		err = ErrInvalidCurrencyCode
		return
	}
	money = Money{Amount: amount, Currency: currency}
	return
}
