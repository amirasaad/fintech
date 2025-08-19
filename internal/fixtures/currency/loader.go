package currency

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/amirasaad/fintech/pkg/registry"
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
			// Verify header has all required columns
			expectedColumns := 7
			if len(rec) < expectedColumns {
				errMsg := fmt.Sprintf(
					"invalid CSV format: expected at least %d columns, got %d",
					expectedColumns,
					len(rec),
				)
				return nil, errors.New(errMsg)
			}
			continue // skip header
		}

		// Skip malformed rows
		if len(rec) < 7 {
			continue
		}

		active := strings.ToLower(rec[6]) == "true"
		// Create a new currency entity with code and name
		entity := registry.NewBaseEntity(rec[0], rec[1])
		// Set the active status
		entity.SetActive(active)

		// Set additional metadata (non-core fields if any)
		// Core fields (code, name, active) are already set on the entity
		metadata := map[string]string{
			"symbol":   rec[2],
			"decimals": rec[3],
			"country":  rec[4],
			"region":   rec[5],
		}
		entity.SetMetadataMap(metadata)
		metas = append(metas, entity)
	}
	return metas, nil
}
