package contracts

// ConversionInfo holds details about a currency conversion performed during a transaction.
type ConversionInfo struct {
	OriginalAmount    float64
	OriginalCurrency  string
	ConvertedAmount   float64
	ConvertedCurrency string
	ConversionRate    float64
}
