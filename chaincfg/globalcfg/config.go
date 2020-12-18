// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package globalcfg contains configuration which must be available
// anywhere in the project, do not import anything which is part of pktd.
package globalcfg

import (
	"time"
)

// ProofOfWork means the type of proof of work used on the chain
type ProofOfWork int

const (
	// PowSha256 is the original proof of work from satoshi.
	// This is the default value
	PowSha256 ProofOfWork = iota

	// PowPacketCrypt is the PoW used by chains such as pkt.cash
	PowPacketCrypt
)

type CoinAmount struct {
	Name       string
	ProperName string
	Units      int64
	Zeros      int
}

// Config is the global config which is accessible anywhere in the app
type Config struct {
	ProofOfWorkAlgorithm ProofOfWork
	HasNetworkSteward    bool
	MaxUnits             int64
	UnitsPerCoin         int64
	MaxTimeOffset        time.Duration
	MedianTimeBlocks     int
	Amounts              []CoinAmount
}

var (
	gConf      Config
	registered bool
)

// BitcoinDefaults creates a new config with the default values for bitcoin
func BitcoinDefaults() Config {
	return Config{
		ProofOfWorkAlgorithm: PowSha256,
		HasNetworkSteward:    false,
		MaxUnits:             21e6 * 1e8,
		UnitsPerCoin:         1e8,
		MaxTimeOffset:        2 * 60 * 60,
		MedianTimeBlocks:     11,
		Amounts: []CoinAmount{
			{Name: "BTC", Units: 1e8, Zeros: 8},
			{Name: "MBTC", Units: 1e14, Zeros: 14},
			{Name: "kBTC", Units: 1e11, Zeros: 11},
			{Name: "mBTC", Units: 1e5, Zeros: 5},
			{Name: "μBTC", Units: 1e2, Zeros: 2},
			{Name: "uBTC", Units: 1e2, Zeros: 2, ProperName: "μBTC"},
			{Name: "Satoshi", Units: 1, Zeros: 0},
			{Name: "satoshi", Units: 1, Zeros: 0, ProperName: "Satoshi"},
		},
	}
}

func PktDefaults() Config {
	return Config{
		ProofOfWorkAlgorithm: PowPacketCrypt,
		HasNetworkSteward:    true,
		UnitsPerCoin:         0x40000000,
		MaxUnits:             0x40000000 * 6000000000,
		Amounts: []CoinAmount{
			{Name: "PKT", Units: int64(1) << 30, Zeros: 9},
			{Name: "MPKT", Units: int64(1) << 30 * 1e6, Zeros: 12},
			{Name: "kPKT", Units: int64(1) << 30 * 1e3, Zeros: 15},

			{Name: "Pibit", Units: int64(1) << 50, Zeros: 15},
			{Name: "Tibit", Units: int64(1) << 40, Zeros: 12},
			{Name: "Gibit", Units: int64(1) << 30, Zeros: 9},
			{Name: "Mibit", Units: int64(1) << 20, Zeros: 6},
			{Name: "Kibit", Units: int64(1) << 10, Zeros: 3},
			{Name: "bit", Units: int64(1), Zeros: 0},
		},
		// Just over 10x bitcoin, but using an odd number because CalcPastMedianTime()
		// algorithm depends on the number being odd
		MedianTimeBlocks: 111,

		// 1/10th that of bitcoin, because blocks come at a 10x rate
		MaxTimeOffset: 60 * 12,
	}
}

// SelectConfig is used to register the blockchain parameters with globalcfg
func SelectConfig(conf Config) {
	if registered {
		panic("globalcfg already selected")
	}
	registered = true
	gConf = conf
}

func checkRegistered() {
	if !registered {
		panic("globalcfg requested but not yet registered")
	}
}

// GetMaxTimeOffset is the maximum number of seconds a block time
// is allowed to be ahead of the current time.
func GetMaxTimeOffset() time.Duration {
	checkRegistered()
	return gConf.MaxTimeOffset
}

// GetMedianTimeBlocks provides the number of previous blocks which should be
// used to calculate the median time used to validate block timestamps.
func GetMedianTimeBlocks() int {
	checkRegistered()
	return gConf.MedianTimeBlocks
}

// GetProofOfWorkAlgorithm tells whether the chain in use uses a custom proof
// of work algorithm or the normal sha256 proof of work.
func GetProofOfWorkAlgorithm() ProofOfWork {
	checkRegistered()
	return gConf.ProofOfWorkAlgorithm
}

// IsPacketCryptAllowedVersion tells whether the specified version of PacketCrypt proof is allowed.
func IsPacketCryptAllowedVersion(version int, blockHeight int32) bool {
	if version > 1 && blockHeight < 113949 {
		return false
	} else if version < 2 && blockHeight > 122621 {
		return false
	}
	return true
}

// HasNetworkSteward returns true for blockchains which require a network steward fee
func HasNetworkSteward() bool {
	checkRegistered()
	return gConf.HasNetworkSteward
}

// SatoshiPerBitcoin returns the number of atomic units per "coin"
func SatoshiPerBitcoin() int64 {
	checkRegistered()
	return gConf.UnitsPerCoin
}

// MaxUnitsU64 returns the maximum number of atomic units of currency
func MaxUnitsI64() int64 {
	checkRegistered()
	return gConf.MaxUnits
}

// UnitsPerCoinI64 returns the maximum number of atomic units per "coin"
func UnitsPerCoinI64() int64 {
	checkRegistered()
	return gConf.UnitsPerCoin
}

func AmountUnits() []CoinAmount {
	checkRegistered()
	return gConf.Amounts
}
