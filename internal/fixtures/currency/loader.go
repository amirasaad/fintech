package currency

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/service/currency"
)

// LoadCurrencyMetaCSV loads currency metadata from a CSV file for test fixtures.
func LoadCurrencyMetaCSV(path string) ([]registry.Entity, error) {
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

	var metas []registry.Entity
	for i, rec := range records {
		if i == 0 {
			continue // skip header
		}
		decimals, _ := strconv.Atoi(rec[3])
		active := strings.ToLower(rec[6]) == "true"
		meta := currency.Entity{
			Code:     money.Code(rec[0]),
			Name:     rec[1],
			Symbol:   rec[2],
			Decimals: decimals,
			Country:  rec[4],
			Region:   rec[5],
			Active:   active,
		}
		entity := registry.NewBaseEntity(meta.Code.String(), meta.Name)
		entity.SetActive(meta.Active)
		entity.SetMetadata("symbol", meta.Symbol)
		entity.SetMetadata("decimals", strconv.Itoa(meta.Decimals))
		entity.SetMetadata("country", meta.Country)
		entity.SetMetadata("region", meta.Region)
		entity.SetMetadata("active", strconv.FormatBool(meta.Active))
		metas = append(metas, entity)
	}
	return metas, nil
}
