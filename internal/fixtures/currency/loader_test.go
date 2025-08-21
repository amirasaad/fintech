package currency_test

import (
	"os"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCurrencyMetaCSV(t *testing.T) {
	// Create a temporary test CSV file
	csvContent := `code,name,symbol,decimals,country,region,active
USD,US Dollar,$,2,United States,Americas,true
EUR,Euro,€,2,Germany,Europe,true`

	tmpFile, err := os.CreateTemp("", "test_currency_*.csv")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Test loading the CSV
	entities, err := currency.LoadCurrencyMetaCSV(tmpFile.Name())
	require.NoError(t, err)
	assert.Len(t, entities, 2)

	// Verify first currency (USD)
	usd := entities[0]
	assert.Equal(t, "USD", usd.ID())
	assert.Equal(t, "US Dollar", usd.Name())
	assert.True(t, usd.Active())

	// Verify USD metadata
	meta := usd.Metadata()
	assert.Equal(t, "$", meta["symbol"])
	assert.Equal(t, "2", meta["decimals"])
	assert.Equal(t, "United States", meta["country"])
	assert.Equal(t, "Americas", meta["region"])

	// Verify second currency (EUR)
	eur := entities[1]
	assert.Equal(t, "EUR", eur.ID())
	assert.Equal(t, "Euro", eur.Name())
	assert.True(t, eur.Active())

	// Verify EUR metadata
	meta = eur.Metadata()
	assert.Equal(t, "€", meta["symbol"])
	assert.Equal(t, "2", meta["decimals"])
	assert.Equal(t, "Germany", meta["country"])
	assert.Equal(t, "Europe", meta["region"])

	// Verify timestamps are set
	assert.False(t, usd.CreatedAt().IsZero())
	assert.False(t, usd.UpdatedAt().IsZero())
}

func TestLoadCurrencyMetaCSV_InvalidFile(t *testing.T) {
	_, err := currency.LoadCurrencyMetaCSV("nonexistent_file.csv")
	assert.Error(t, err)
}

func TestLoadCurrencyMetaCSV_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "empty_*.csv")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	entities, err := currency.LoadCurrencyMetaCSV(tmpFile.Name())
	require.NoError(t, err)
	assert.Empty(t, entities)
}

func TestLoadCurrencyMetaCSV_InvalidCSV(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_*.csv")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	// Write a CSV with not enough columns
	_, err = tmpFile.WriteString("code,name\nUSD,US Dollar")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	_, err = currency.LoadCurrencyMetaCSV(tmpFile.Name())
	assert.Error(t, err, "Expected error for CSV with insufficient columns")
}
