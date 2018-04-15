package exchange

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/MDLlife/MDL/src/util/droplet"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestCalculateMDLValue(t *testing.T) {
	cases := []struct {
		maxDecimals int
		satoshis    int64
		rate        string
		result      uint64
		err         error
	}{
		{
			maxDecimals: 0,
			satoshis:    -1,
			rate:        "1",
			err:         errors.New("satoshis must be greater than or equal to 0"),
		},

		{
			maxDecimals: 0,
			satoshis:    1,
			rate:        "-1",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			satoshis:    1,
			rate:        "0",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			satoshis:    1,
			rate:        "invalidrate",
			err:         errors.New("can't convert invalidrate to decimal: exponent is not numeric"),
		},
		{
			maxDecimals: 0,
			satoshis:    1,
			rate:        "12k",
			err:         errors.New("can't convert 12k to decimal"),
		},
		{
			maxDecimals: 0,
			satoshis:    1,
			rate:        "1b",
			err:         errors.New("can't convert 1b to decimal"),
		},
		{
			maxDecimals: 0,
			satoshis:    1,
			rate:        "",
			err:         errors.New("can't convert  to decimal"),
		},

		{
			maxDecimals: 0,
			satoshis:    0,
			rate:        "1",
			result:      0,
		},

		{
			maxDecimals: 0,
			satoshis:    1e8,
			rate:        "1",
			result:      1e6,
		},

		{
			maxDecimals: 0,
			satoshis:    1e8,
			rate:        "500",
			result:      500e6,
		},

		{
			maxDecimals: 0,
			satoshis:    100e8,
			rate:        "500",
			result:      50000e6,
		},

		{
			maxDecimals: 0,
			satoshis:    2e5,   // 0.002 BTC
			rate:        "500", // 500 MDL/BTC = 1 MDL / 0.002 BTC
			result:      1e6,   // 1 MDL
		},

		{
			maxDecimals: 0,
			satoshis:    1e8, // 1 BTC
			rate:        "1/2",
			result:      0, // 0.5 MDL
		},
		{
			maxDecimals: 0,
			satoshis:    12345e8, // 12345 BTC
			rate:        "1/2",
			result:      6172e6, // 6172 MDL
		},
		{
			maxDecimals: 0,
			satoshis:    1e8,
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 0,
			satoshis:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      63e6, // 63 MDL
		},
		{
			maxDecimals: 0,
			satoshis:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      12345e6, // 12345 MDL
		},
		{
			maxDecimals: 0,
			satoshis:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      58436e6, // 58436 MDL
		},

		{
			maxDecimals: 1,
			satoshis:    1e8, // 1 BTC
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 1,
			satoshis:    12345e8, // 12345 BTC
			rate:        "1/2",
			result:      6172e6 + 5e5, // 6172.5 MDL
		},
		{
			maxDecimals: 1,
			satoshis:    1e8,
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 1,
			satoshis:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      63e6 + 2e5, // 63.2 MDL
		},
		{
			maxDecimals: 1,
			satoshis:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      12345e6 + 6e5, // 12345.6 MDL
		},
		{
			maxDecimals: 1,
			satoshis:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      58436e6 + 2e5, // 58436.2 MDL
		},

		{
			maxDecimals: 2,
			satoshis:    1e8, // 1 BTC
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 2,
			satoshis:    12345e8, // 12345 BTC
			rate:        "1/2",
			result:      6172e6 + 5e5, // 6172.5 MDL
		},
		{
			maxDecimals: 2,
			satoshis:    1e8,
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 2,
			satoshis:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      63e6 + 2e5, // 63.2 MDL
		},
		{
			maxDecimals: 2,
			satoshis:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      12345e6 + 6e5 + 7e4, // 12345.67 MDL
		},
		{
			maxDecimals: 2,
			satoshis:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      58436e6 + 2e5 + 1e4, // 58436.21 MDL
		},

		{
			maxDecimals: 3,
			satoshis:    1e8, // 1 BTC
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 3,
			satoshis:    12345e8, // 12345 BTC
			rate:        "1/2",
			result:      6172e6 + 5e5, // 6172.5 MDL
		},
		{
			maxDecimals: 3,
			satoshis:    1e8,
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 3,
			satoshis:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      63e6 + 2e5 + 9e3, // 63.209 MDL
		},
		{
			maxDecimals: 3,
			satoshis:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      12345e6 + 6e5 + 7e4 + 8e3, // 12345.678 MDL
		},
		{
			maxDecimals: 3,
			satoshis:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      58436e6 + 2e5 + 1e4 + 4e3, // 58436.214 MDL
		},

		{
			maxDecimals: 4,
			satoshis:    1e8,
			rate:        "0.0001",
			result:      1e2, // 0.0001 MDL
		},

		{
			maxDecimals: 3,
			satoshis:    125e4,
			rate:        "1250",
			result:      15e6 + 6e5 + 2e4 + 5e3, // 15.625 MDL
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("satoshis=%d rate=%s maxDecimals=%d", tc.satoshis, tc.rate, tc.maxDecimals)
		t.Run(name, func(t *testing.T) {
			result, err := CalculateBtcMDLValue(tc.satoshis, tc.rate, tc.maxDecimals)
			if tc.err == nil {
				require.NoError(t, err)
				require.Equal(t, tc.result, result, "%d != %d", tc.result, result)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.err, err)
				require.Equal(t, uint64(0), result, "%d != 0", result)
			}
		})
	}
}
func TestCalculateEthMDLValue(t *testing.T) {
	cases := []struct {
		maxDecimals int
		wei         *big.Int
		rate        string
		result      uint64
		err         error
	}{
		{
			maxDecimals: 0,
			wei:         big.NewInt(-1),
			rate:        "1",
			err:         errors.New("wei must be greater than or equal to 0"),
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(1),
			rate:        "-1",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(1),
			rate:        "0",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(1),
			rate:        "invalidrate",
			err:         errors.New("can't convert invalidrate to decimal: exponent is not numeric"),
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1),
			rate:        "100k",
			err:         errors.New("can't convert 100k to decimal"),
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1),
			rate:        "0.1b",
			err:         errors.New("can't convert 0.1b to decimal"),
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1),
			rate:        "",
			err:         errors.New("can't convert  to decimal"),
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(0),
			rate:        "1",
			result:      0,
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1000),
			rate:        "0.001",
			result:      0,
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(1e18),
			rate:        "1",
			result:      1e6,
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(1e18),
			rate:        "500",
			result:      500e6,
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(1).Mul(big.NewInt(100), big.NewInt(1e18)),
			rate:        "500",
			result:      50000e6,
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(2e15), // 0.002 ETH
			rate:        "500",            // 500 MDL/ETH = 1 MDL / 0.002 ETH
			result:      1e6,              // 1 MDL
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(1e18), // 1 ETH
			rate:        "1/2",
			result:      0, // 0.5 MDL
		},

		{
			maxDecimals: 0,
			wei:         big.NewInt(11345e13), // 0.11345 ETH
			rate:        "100",
			result:      11e6, // 11 MDL
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "200",
			result:      44904e6, // 44904 MDL
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "1568",
			result:      352053e6, // 352053(224.5236 * 1568=352053.0048) MDL
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "0.15",
			result:      33e6, // 33 MDL
		},
		{
			maxDecimals: 0,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "2/3",
			result:      149e6, // 149 MDL
		},
		{
			maxDecimals: 1,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "200",
			result:      44904e6 + 7e5, // 44904.7 MDL
		},
		{
			maxDecimals: 1,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "1568",
			result:      352053e6, // 352053(224.5236 * 1568=352053.0048) MDL
		},
		{
			maxDecimals: 1,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "0.15",
			result:      33e6 + 6e5, // 33.6 MDL
		},
		{
			maxDecimals: 1,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "2/3",
			result:      149e6 + 6e5, // 149.6 MDL
		},
		{
			maxDecimals: 2,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "200",
			result:      44904e6 + 7e5 + 2e4, // 44904.72 MDL
		},
		{
			maxDecimals: 2,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "1568",
			result:      352053e6, // 352053.00(224.5236 * 1568=352053.0048) MDL
		},
		{
			maxDecimals: 2,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "0.15",
			result:      33e6 + 6e5 + 7e4, // 33.67 MDL
		},
		{
			maxDecimals: 2,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "2/3",
			result:      149e6 + 6e5 + 8e4, // 149 MDL
		},
		{
			maxDecimals: 3,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "1568",
			result:      352053e6 + 4e3, // 352053.004(224.5236 * 1568=352053.0048) MDL
		},
		{
			maxDecimals: 3,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "0.15",
			result:      33e6 + 6e5 + 7e4 + 8e3, // 33.678 MDL
		},
		{
			maxDecimals: 3,
			wei:         big.NewInt(1).Mul(big.NewInt(2245236), big.NewInt(1e14)), // 224.5236 ETH
			rate:        "2/3",
			result:      149e6 + 6e5 + 8e4 + 2e3, // 149.682 MDL
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("wei=%d rate=%s maxDecimals=%d", tc.wei, tc.rate, tc.maxDecimals)
		t.Run(name, func(t *testing.T) {
			result, err := CalculateEthMDLValue(tc.wei, tc.rate, tc.maxDecimals)
			if tc.err == nil {
				require.NoError(t, err)
				require.Equal(t, tc.result, result, "%d != %d", tc.result, result)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.err, err)
				require.Equal(t, uint64(0), result, "%d != 0", result)
			}
		})
	}
}

func TestCalculateSkyMDLValue(t *testing.T) {
	cases := []struct {
		maxDecimals int
		droplets    int64
		rate        string
		result      uint64
		err         error
	}{
		{
			maxDecimals: 0,
			droplets:    -1,
			rate:        "1",
			err:         errors.New("droplets must be greater than or equal to 0"),
		},

		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "-1",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "0",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "invalidrate",
			err:         errors.New("can't convert invalidrate to decimal: exponent is not numeric"),
		},
		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "12k",
			err:         errors.New("can't convert 12k to decimal"),
		},
		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "1b",
			err:         errors.New("can't convert 1b to decimal"),
		},
		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "",
			err:         errors.New("can't convert  to decimal"),
		},

		{
			maxDecimals: 0,
			droplets:    0,
			rate:        "1",
			result:      0,
		},

		{
			maxDecimals: 0,
			droplets:    1e6,
			rate:        "1",
			result:      1e6,
		},

		{
			maxDecimals: 0,
			droplets:    1e6,
			rate:        "500",
			result:      500e6,
		},

		{
			maxDecimals: 0,
			droplets:    100e6,
			rate:        "500",
			result:      50000e6,
		},

		{
			maxDecimals: 0,
			droplets:    2e5,   // 0.002 BTC
			rate:        "500", // 500 MDL/BTC = 1 MDL / 0.002 BTC
			result:      1e8,   // 1 MDL
		},

		{
			maxDecimals: 0,
			droplets:    1e6, // 1 BTC
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 0,
			droplets:    12345e6, // 12345 BTC
			rate:        "1/2",
			result:      61725e5, // 61725 MDL
		},
		{
			maxDecimals: 0,
			droplets:    1e6,
			rate:        "0.0001",
			result:      100, // 0 MDL
		},
		{
			maxDecimals: 0,
			droplets:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      6320987136, // 63 MDL
		},
		{
			maxDecimals: 0,
			droplets:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      123456789e4, // 12345 MDL
		},
		{
			maxDecimals: 0,
			droplets:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      5843621489218, // 58436 MDL
		},

		{
			maxDecimals: 1,
			droplets:    1e6, // 1 BTC
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 1,
			droplets:    12345e8, // 12345 BTC
			rate:        "1/2",
			result:      61725e7, // 6172.5 MDL
		},
		{
			maxDecimals: 1,
			droplets:    1e3,
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 1,
			droplets:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      6320987136, // 63.2 MDL
		},
		{
			maxDecimals: 1,
			droplets:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      1234567890000, // 12345.6 MDL
		},
		{
			maxDecimals: 1,
			droplets:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      5843621489218, // 58436.2 MDL
		},

		{
			maxDecimals: 2,
			droplets:    1e8, // 1 BTC
			rate:        "1/2",
			result:      50000000, // 0.5 MDL
		},
		{
			maxDecimals: 2,
			droplets:    12345e8, // 12345 BTC
			rate:        "1/2",
			result:      617250000000, // 6172.5 MDL
		},
		{
			maxDecimals: 2,
			droplets:    1e8,
			rate:        "0.0001",
			result:      1e4, // 0 MDL
		},
		{
			maxDecimals: 2,
			droplets:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      6320987136, // 63.2 MDL
		},
		{
			maxDecimals: 2,
			droplets:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      1234567890000, // 12345.67 MDL
		},
		{
			maxDecimals: 2,
			droplets:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      5843621489218, // 58436.21 MDL
		},

		{
			maxDecimals: 3,
			droplets:    1e8, // 1 BTC
			rate:        "1/2",
			result:      50000000, // 0.5 MDL
		},
		{
			maxDecimals: 3,
			droplets:    12345e8, // 12345 BTC
			rate:        "1/2",
			result:      617250000000, // 6172.5 MDL
		},
		{
			maxDecimals: 3,
			droplets:    1e8,
			rate:        "0.0001",
			result:      10000, // 0 MDL
		},
		{
			maxDecimals: 3,
			droplets:    12345678, // 0.12345678 BTC
			rate:        "512",
			result:      6320987136, // 63.209 MDL
		},
		{
			maxDecimals: 3,
			droplets:    123456789, // 1.23456789 BTC
			rate:        "10000",
			result:      1234567890000, // 12345.678 MDL
		},
		{
			maxDecimals: 3,
			droplets:    876543219e4, // 87654.3219 BTC
			rate:        "2/3",
			result:      5843621489218, // 58436.214 MDL
		},

		{
			maxDecimals: 4,
			droplets:    1e8,
			rate:        "0.0001",
			result:      10000, // 0.0001 MDL
		},

		{
			maxDecimals: 3,
			droplets:    125e4,
			rate:        "1250",
			result:      1562500000, // 15.625 MDL
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("droplets=%d rate=%s maxDecimals=%d", tc.droplets, tc.rate, tc.maxDecimals)
		t.Run(name, func(t *testing.T) {
			result, err := CalculateSkyMDLValue(tc.droplets, tc.rate, tc.maxDecimals)
			if tc.err == nil {
				require.NoError(t, err)
				require.Equal(t, tc.result, result, "%d != %d", tc.result, result)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.err, err)
				require.Equal(t, uint64(0), result, "%d != 0", result)
			}
		})
	}
}

func TestCalculateWavesMDLValue(t *testing.T) {
	//The WAVES price is listed as 10^8. (1e8=1 WAVES)
	cases := []struct {
		maxDecimals int
		droplets    int64
		rate        string
		result      uint64
		err         error
	}{
		{
			maxDecimals: 0,
			droplets:    10000000, //0.1 WAVES
			rate:        "88",
			result:      8800000, //8.8 MDL
		},
		{
			maxDecimals: 0,
			droplets:    2e7, //0.2 WAVES
			rate:        "88",
			result:      176e5, //17.6 MDL
		},
		{
			maxDecimals: 0,
			droplets:    1297e7, // 129.7 WAVES
			rate:        "88",
			result:      114136e5, // 11413.6 MDL
		},
		{
			maxDecimals: 0,
			droplets:    -1,
			rate:        "1",
			err:         errors.New("droplets must be greater than or equal to 0"),
		},
		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "-1",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "0",
			err:         errors.New("rate must be greater than zero"),
		},

		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "invalidrate",
			err:         errors.New("can't convert invalidrate to decimal: exponent is not numeric"),
		},
		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "12k",
			err:         errors.New("can't convert 12k to decimal"),
		},
		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "1b",
			err:         errors.New("can't convert 1b to decimal"),
		},
		{
			maxDecimals: 0,
			droplets:    1,
			rate:        "",
			err:         errors.New("can't convert  to decimal"),
		},

		{
			maxDecimals: 0,
			droplets:    0,
			rate:        "1",
			result:      0,
		},
		{
			maxDecimals: 0,
			droplets:    1e8, // 1 WAVES
			rate:        "1",
			result:      1e6, // 1 MDL
		},

		{
			maxDecimals: 0,
			droplets:    1e8, // 1 WAVES
			rate:        "500",
			result:      500e6, // 500 MDL
		},

		{
			maxDecimals: 0,
			droplets:    100e6, // 1 WAVES
			rate:        "500",
			result:      500e6, // 500 MDL
		},
		{
			maxDecimals: 0,
			droplets:    2e5,   // 0.2 WAVES
			rate:        "500", // 500 MDL/WAVES = 1 MDL / 0.002 BTC
			result:      1e6,   // 100 MDL
		},

		{
			maxDecimals: 0,
			droplets:    1e8, // 1 WAVES
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 0,
			droplets:    1e12, // 10000 WAVES
			rate:        "1/2",
			result:      50000e5, // 5000 MDL
		},

		{
			maxDecimals: 0,
			droplets:    12345e6, // 123.45 WAVES
			rate:        "1/2",
			result:      617e5, // 61.7 MDL
		},
		{
			maxDecimals: 0,
			droplets:    1e6, // 0.01 MDL
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 0,
			droplets:    12345678, // 0.123456 WAVES
			rate:        "512",
			result:      632e5, // 63 MDL
		},
		{
			maxDecimals: 0,
			droplets:    123456789, // 1.234567 BTC
			rate:        "10000",
			result:      123456e5, // 12345.6 MDL
		},
		{
			maxDecimals: 0,
			droplets:    876543219e4, // 87654.3219 WAVES
			rate:        "2/3",
			result:      584362e5, // 58436.2 MDL
		},
		{
			maxDecimals: 1,
			droplets:    1e7, // 1 WAVES
			rate:        "1/2",
			result:      5e4, // 0.5 MDL
		},
		{
			maxDecimals: 1,
			droplets:    12345e8, // 12345 WAVES
			rate:        "1/2",
			result:      61725e5, // 6172.5 MDL
		},
		{
			maxDecimals: 1,
			droplets:    1e3, // 0.000010 WAVES
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 1,
			droplets:    12345678, // 0.123456 WAVES
			rate:        "512",
			result:      632e5, // 63.2 MDL
		},
		{
			maxDecimals: 1,
			droplets:    123456789, // 1.234567 WAVES
			rate:        "10000",
			result:      1234567e4, // 12345.67 MDL
		},
		{
			maxDecimals: 1,
			droplets:    876543219e4, // 87654.3219 WAVES
			rate:        "2/3",
			result:      5843621e4, // 58436.2 MDL
		},

		{
			maxDecimals: 2,
			droplets:    1e8, // 1 WAVES
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 2,
			droplets:    12345e8, // 12345 WAVES
			rate:        "1/2",
			result:      61725e5, // 6172.5 MDL
		},
		{
			maxDecimals: 2,
			droplets:    1e8, // 1 WAVES
			rate:        "0.0001",
			result:      0, // 0 MDL
		},
		{
			maxDecimals: 2,
			droplets:    12345678, // 0.123456 BTC
			rate:        "512",
			result:      63209e3, // 63.209 MDL
		},
		{
			maxDecimals: 2,
			droplets:    1234567e2, // 1.234567 WAVES
			rate:        "10000",
			result:      1234567e4, // 12345.67 MDL
		},
		{
			maxDecimals: 2,
			droplets:    876543219e4, // 87654.3219 WAVES
			rate:        "2/3",
			result:      58436214e3, // 58436.214 MDL
		},

		{
			maxDecimals: 3,
			droplets:    1e8, // 1 WAVES
			rate:        "1/2",
			result:      5e5, // 0.5 MDL
		},
		{
			maxDecimals: 3,
			droplets:    12345e8, // 12345 WAVES
			rate:        "1/2",
			result:      61725e5, // 6172.5 MDL
		},
		{
			maxDecimals: 3,
			droplets:    1e8, // 1 WAVES
			rate:        "0.0001",
			result:      1e2, // 0.0001 MDL
		},
		{
			maxDecimals: 3,
			droplets:    12345678, // 0.123456 WAVES
			rate:        "512",
			result:      632098e2, // 63.2098 MDL
		},
		{
			maxDecimals: 3,
			droplets:    123456789, // 1.234567 WAVES
			rate:        "10000",
			result:      123456789e2, // 12345.678 MDL
		},
		{
			maxDecimals: 3,
			droplets:    876543219e4, // 87654.3219 WAVES
			rate:        "2/3",
			result:      584362148e2, // 58436.2148 MDL
		},

		{
			maxDecimals: 4,
			droplets:    1e8, // 0.0001 WAVES
			rate:        "0.0001",
			result:      1e2, // 0.0001 MDL
		},

		{
			maxDecimals: 3,
			droplets:    125e4, // 0.0125 WAVES
			rate:        "1250",
			result:      15625e3, // 15.625 MDL
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("droplets=%d rate=%s maxDecimals=%d", tc.droplets, tc.rate, tc.maxDecimals)
		t.Run(name, func(t *testing.T) {
			result, err := CalculateWavesMDLValue(tc.droplets, tc.rate, tc.maxDecimals)
			if tc.err == nil {

				expectedAmtCoins, err := droplet.ToString(result)
				if err != nil {
					t.Error("droplet.ToString failed")
					return
				}
				expectedDecimal := decimal.New(int64(tc.result), -6)
				expectedStr := expectedDecimal.StringFixed(6)

				droptletsDecimal := decimal.New(int64(tc.droplets/100), -6)
				dropletsAmtCoins := droptletsDecimal.StringFixed(6)

				require.NoError(t, err)

				require.Equal(t, expectedStr, expectedAmtCoins, "expected(%s) != actual(%s), coins(%s), rate(%s)", expectedStr, expectedAmtCoins, dropletsAmtCoins, tc.rate)
				require.Equal(t, tc.result, result, "%d != %d")
			} else {
				require.Error(t, err)
				require.Equal(t, tc.err, err)
				require.Equal(t, uint64(0), result, "%d != 0", result)
			}
		})
	}
}


func TestCalculateWaves_MDLLIFE_MDLValue(t *testing.T) {
	//The WAVES price is listed as 10^8. (1e8=1 WAVES)
	cases := []struct {
		maxDecimals int
		droplets    int64
		rate        string
		result      uint64
		err         error
	}{
		{
			maxDecimals: 0,
			droplets:    10000000, //0.1 MDL.life //http://node.wavesbi.com:6869/blocks/seq/959412/959419 http://wavesgo.com/transactions/38dwB49fQ2bY33z6V36exD2u2JBgUoSoauYThyiV8Lfi
			rate:        "1",
			result:      1e5, //0.1 MDL
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("droplets=%d rate=%s maxDecimals=%d", tc.droplets, tc.rate, tc.maxDecimals)
		t.Run(name, func(t *testing.T) {
			result, err := CalculateWavesMDLValue(tc.droplets, tc.rate, tc.maxDecimals)
			if tc.err == nil {

				expectedAmtCoins, err := droplet.ToString(result)
				if err != nil {
					t.Error("droplet.ToString failed")
					return
				}
				expectedDecimal := decimal.New(int64(tc.result), -6)
				expectedStr := expectedDecimal.StringFixed(6)

				droptletsDecimal := decimal.New(int64(tc.droplets/100), -6)
				dropletsAmtCoins := droptletsDecimal.StringFixed(6)

				require.NoError(t, err)

				require.Equal(t, expectedStr, expectedAmtCoins, "expected(%s) != actual(%s), coins(%s), rate(%s)", expectedStr, expectedAmtCoins, dropletsAmtCoins, tc.rate)
				require.Equal(t, tc.result, result, "%d != %d")
			} else {
				require.Error(t, err)
				require.Equal(t, tc.err, err)
				require.Equal(t, uint64(0), result, "%d != 0", result)
			}
		})
	}
}
