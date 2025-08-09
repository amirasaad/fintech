package currency

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"

	"github.com/amirasaad/fintech/pkg/currency"
)

// LoadCurrencyMetaCSV loads currency metadata from a CSV file for test fixtures.
func LoadCurrencyMetaCSV(path string) ([]currency.Meta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var metas []currency.Meta
	for i, rec := range records {
		if i == 0 {
			continue // skip header
		}
		decimals, _ := strconv.Atoi(rec[3])
		active := strings.ToLower(rec[6]) == "true"
		metas = append(metas, currency.Meta{
			Code:     rec[0],
			Name:     rec[1],
			Symbol:   rec[2],
			Decimals: decimals,
			Country:  rec[4],
			Region:   rec[5],
			Active:   active,
		})
	}
	return metas, nil
}
