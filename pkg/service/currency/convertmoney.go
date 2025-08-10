package currency

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/money"
)

// ConvertMoney converts a Money value object to the target currency using the converter.
// Returns a new Money object in the target currency and the conversion info.
func ConvertMoney(
	converter currency.Converter,
	m money.Money,
	to currency.Code,
) (*money.Money, *currency.Info, error) {
	convInfo, err := converter.Convert(
		m.AmountFloat(),
		m.Currency(),
		to,
	)
	if err != nil {
		return nil, nil, err
	}
	converted, err := money.New(convInfo.ConvertedAmount, to)
	if err != nil {
		return nil, nil, err
	}
	return &converted, convInfo, nil
}
