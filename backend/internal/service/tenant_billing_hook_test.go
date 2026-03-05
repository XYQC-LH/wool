package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestDecimalToMicroAndBack(t *testing.T) {
	value := decimal.RequireFromString("12.345678")
	micro := decimalToMicro(value)

	require.Equal(t, int64(12345678), micro)
	require.True(t, microToDecimal(micro).Equal(value))
}

func TestDecimalToMicroRoundUp(t *testing.T) {
	value := decimal.RequireFromString("0.0000001")
	require.Equal(t, int64(1), decimalToMicro(value))
}
