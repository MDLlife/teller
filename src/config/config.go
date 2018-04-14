// Package config is used to records the service configurations
package config

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/MDLlife/MDL/src/visor"
	"github.com/MDLlife/MDL/src/wallet"
	"github.com/MDLlife/teller/src/util/mathutil"
)

const (
	// BuyMethodDirect is used when buying directly from the local hot wallet
	BuyMethodDirect = "direct"
	// BuyMethodPassthrough is used when coins are first bought from an exchange before sending from the local hot wallet
	BuyMethodPassthrough = "passthrough"
)

var (
	// ErrInvalidBuyMethod is returned if BindAddress is called with an invalid buy method
	ErrInvalidBuyMethod = errors.New("Invalid buy method")
)

// ValidateBuyMethod returns an error if a buy method string is invalid
func ValidateBuyMethod(m string) error {
	switch m {
	case BuyMethodDirect, BuyMethodPassthrough:
		return nil
	default:
		return ErrInvalidBuyMethod
	}
}

// Config represents the configuration root
type Config struct {
	// Enable debug logging
	Debug bool `mapstructure:"debug"`
	// Run with gops profiler
	Profile bool `mapstructure:"profile"`
	// Where log is saved
	LogFilename string `mapstructure:"logfile"`
	// Where database is saved, inside the ~/.teller-mdl data directory
	DBFilename string `mapstructure:"dbfile"`

	// Path of BTC addresses JSON file
	BtcAddresses string `mapstructure:"btc_addresses"`
	// Path of ETH addresses JSON file
	EthAddresses string `mapstructure:"eth_addresses"`
	// Path of SKY addresses JSON file
	SkyAddresses string `mapstructure:"sky_addresses"`
	// Path of Waves addresses JSON file
	WavesAddresses string `mapstructure:"waves_addresses"`
	// Path of Waves MDL addresses JSON file
	WavesMDLAddresses string `mapstructure:"waves_mdl_addresses"`

	Teller Teller `mapstructure:"teller"`

	MDLRPC      MDLRPC   `mapstructure:"mdl_rpc"`
	BtcRPC      BtcRPC   `mapstructure:"btc_rpc"`
	EthRPC      EthRPC   `mapstructure:"eth_rpc"`
	SkyRPC      SkyRPC   `mapstructure:"sky_rpc"`
	WavesRPC    WavesRPC `mapstructure:"waves_rpc"`
	WavesMDLRPC WavesRPC `mapstructure:"waves_mdl_rpc"`

	BtcScanner      BtcScanner   `mapstructure:"btc_scanner"`
	EthScanner      EthScanner   `mapstructure:"eth_scanner"`
	SkyScanner      SkyScanner   `mapstructure:"sky_scanner"`
	WavesScanner    WavesScanner `mapstructure:"waves_scanner"`
	WavesMDLScanner WavesScanner `mapstructure:"waves_mdl_scanner"`

	MDLExchanger MDLExchanger `mapstructure:"mdl_exchanger"`

	Web Web `mapstructure:"web"`

	AdminPanel AdminPanel `mapstructure:"admin_panel"`

	Dummy Dummy `mapstructure:"dummy"`
}

// SupportedCrypto is used in the UI to build a list of supported Cryptos
type SupportedCrypto struct {
	Name            string `json:"name"`
	Label           string `json:"label"` // i18n label for translation
	Enabled         bool   `json:"enabled"`
	ExchangeRateUSD string `json:"exchange_rate_usd"`
	ExchangeRate    string `json:"exchange_rate"`
}

// Teller config for teller
type Teller struct {
	// Max number of btc addresses a mdl address can bind
	MaxBoundAddresses int `mapstructure:"max_bound_addrs"`
	// Allow address binding
	BindEnabled bool `mapstructure:"bind_enabled"`
	// Currently supported purchase methods
}

// MDLRPC config for MDL daemon node RPC
type MDLRPC struct {
	Address string `mapstructure:"address"`
}

// BtcRPC config for btcrpc
type BtcRPC struct {
	Server  string `mapstructure:"server"`
	User    string `mapstructure:"user"`
	Pass    string `mapstructure:"pass"`
	Cert    string `mapstructure:"cert"`
	Enabled bool   `mapstructure:"enabled"`
}

// EthRPC config for ethrpc
type EthRPC struct {
	Server  string `mapstructure:"server"`
	Port    string `mapstructure:"port"`
	Enabled bool   `mapstructure:"enabled"`
}

// SkyRPC config for skyrpc
type SkyRPC struct {
	Server  string `mapstructure:"server"`
	Port    string `mapstructure:"port"`
	Enabled bool   `mapstructure:"enabled"`
}

// WavesRPC config for wavesrpc
type WavesRPC struct {
	Server   string `mapstructure:"server"`
	Port     string `mapstructure:"port"`
	Enabled  bool   `mapstructure:"enabled"`
	Protocol string `mapstructure:"protocol"`
}

// WavesMDLRPC config for wavesmdlrpc
type WavesMDLRPC struct {
	Server  string `mapstructure:"server"`
	Port    string `mapstructure:"port"`
	Enabled bool   `mapstructure:"enabled"`
}

// BtcScanner config for BTC scanner
type BtcScanner struct {
	// How often to try to scan for blocks
	ScanPeriod            time.Duration `mapstructure:"scan_period"`
	InitialScanHeight     int64         `mapstructure:"initial_scan_height"`
	ConfirmationsRequired int64         `mapstructure:"confirmations_required"`
}

// EthScanner config for ETH scanner
type EthScanner struct {
	// How often to try to scan for blocks
	ScanPeriod            time.Duration `mapstructure:"scan_period"`
	InitialScanHeight     int64         `mapstructure:"initial_scan_height"`
	ConfirmationsRequired int64         `mapstructure:"confirmations_required"`
}

// SkyScanner config for SKY scanner
type SkyScanner struct {
	// How often to try to scan for blocks
	ScanPeriod            time.Duration `mapstructure:"scan_period"`
	InitialScanHeight     int64         `mapstructure:"initial_scan_height"`
	ConfirmationsRequired int64         `mapstructure:"confirmations_required"`
}

// WavesScanner config for WAVES scanner
type WavesScanner struct {
	// How often to try to scan for blocks
	ScanPeriod            time.Duration `mapstructure:"scan_period"`
	InitialScanHeight     int64         `mapstructure:"initial_scan_height"`
	ConfirmationsRequired int64         `mapstructure:"confirmations_required"`
}

// WavesMDLScanner config for WAVES MDL scanner
type WavesMDLScanner struct {
	// How often to try to scan for blocks
	ScanPeriod            time.Duration `mapstructure:"scan_period"`
	InitialScanHeight     int64         `mapstructure:"initial_scan_height"`
	ConfirmationsRequired int64         `mapstructure:"confirmations_required"`
}

// MDLExchanger config for mdl sender, disabling enabling coin on exchange, message and rates
type MDLExchanger struct {
	// exchange rate. Can be an int, float or rational fraction string
	MDLBtcExchangeName    string `mapstructure:"mdl_btc_exchange_name"`
	MDLBtcExchangeRate    string `mapstructure:"mdl_btc_exchange_rate"`
	MDLBtcExchangeRateUSD string `mapstructure:"mdl_btc_exchange_rate_usd"`
	MDLBtcExchangeLabel   string `mapstructure:"mdl_btc_exchange_label"`
	MDLBtcExchangeEnabled bool   `mapstructure:"mdl_btc_exchange_enabled"`

	MDLEthExchangeName    string `mapstructure:"mdl_eth_exchange_name"`
	MDLEthExchangeRate    string `mapstructure:"mdl_eth_exchange_rate"`
	MDLEthExchangeRateUSD string `mapstructure:"mdl_eth_exchange_rate_usd"`
	MDLEthExchangeLabel   string `mapstructure:"mdl_eth_exchange_label"`
	MDLEthExchangeEnabled bool   `mapstructure:"mdl_eth_exchange_enabled"`

	MDLSkyExchangeName    string `mapstructure:"mdl_sky_exchange_name"`
	MDLSkyExchangeRate    string `mapstructure:"mdl_sky_exchange_rate"`
	MDLSkyExchangeRateUSD string `mapstructure:"mdl_sky_exchange_rate_usd"`
	MDLSkyExchangeLabel   string `mapstructure:"mdl_sky_exchange_label"`
	MDLSkyExchangeEnabled bool   `mapstructure:"mdl_sky_exchange_enabled"`

	MDLWavesExchangeName    string `mapstructure:"mdl_waves_exchange_name"`
	MDLWavesExchangeRate    string `mapstructure:"mdl_waves_exchange_rate"`
	MDLWavesExchangeRateUSD string `mapstructure:"mdl_waves_exchange_rate_usd"`
	MDLWavesExchangeLabel   string `mapstructure:"mdl_waves_exchange_label"`
	MDLWavesExchangeEnabled bool   `mapstructure:"mdl_waves_exchange_enabled"`

	MDLWavesMDLExchangeName    string `mapstructure:"mdl_waves_mdl_exchange_name"`
	MDLWavesMDLExchangeRate    string `mapstructure:"mdl_waves_mdl_exchange_rate"`
	MDLWavesMDLExchangeRateUSD string `mapstructure:"mdl_waves_mdl_exchange_rate_usd"`
	MDLWavesMDLExchangeLabel   string `mapstructure:"mdl_waves_mdl_exchange_label"`
	MDLWavesMDLExchangeEnabled bool   `mapstructure:"mdl_waves_mdl_exchange_enabled"`

	// Number of decimal places to truncate MDL to
	MaxDecimals int `mapstructure:"max_decimals"`
	// How long to wait before rechecking transaction confirmations
	TxConfirmationCheckWait time.Duration `mapstructure:"tx_confirmation_check_wait"`
	// Path of hot MDL wallet file on disk
	Wallet string `mapstructure:"wallet"`
	// Allow sending of coins (deposits will still be received and recorded)
	SendEnabled bool `mapstructure:"send_enabled"`
	// Method of purchasing coins ("direct buy" or "passthrough"
	BuyMethod string `mapstructure:"buy_method"`
}

// Validate validates the MDLExchanger config
func (c MDLExchanger) Validate() error {
	if errs := c.validate(); len(errs) != 0 {
		return errs[0]
	}

	if errs := c.validateWallet(); len(errs) != 0 {
		return errs[0]
	}

	return nil
}

func (c MDLExchanger) validate() []error {
	var errs []error

	if _, err := mathutil.ParseRate(c.MDLBtcExchangeRate); err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.mdl_btc_exchange_rate invalid: %v", err))
	}

	if _, err := mathutil.ParseRate(c.MDLEthExchangeRate); err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.mdl_eth_exchange_rate invalid: %v", err))
	}

	if _, err := mathutil.ParseRate(c.MDLSkyExchangeRate); err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.mdl_sky_exchange_rate invalid: %v", err))
	}

	if _, err := mathutil.ParseRate(c.MDLWavesExchangeRate); err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.mdl_waves_exchange_rate invalid: %v", err))
	}

	if _, err := mathutil.ParseRate(c.MDLWavesMDLExchangeRate); err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.mdl_waves_mdl_exchange_rate invalid: %v", err))
	}

	if c.MaxDecimals < 0 {
		errs = append(errs, errors.New("mdl_exchanger.max_decimals can't be negative"))
	}

	if uint64(c.MaxDecimals) > visor.MaxDropletPrecision {
		errs = append(errs, fmt.Errorf("mdl_exchanger.max_decimals is larger than visor.MaxDropletPrecision=%d", visor.MaxDropletPrecision))
	}

	if err := ValidateBuyMethod(c.BuyMethod); err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.buy_method must be \"%s\" or \"%s\"", BuyMethodDirect, BuyMethodPassthrough))
	}

	return errs
}

func (c MDLExchanger) validateWallet() []error {
	var errs []error

	if c.Wallet == "" {
		errs = append(errs, errors.New("mdl_exchanger.wallet missing"))
	}

	if _, err := os.Stat(c.Wallet); os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("mdl_exchanger.wallet file %s does not exist", c.Wallet))
	}

	w, err := wallet.Load(c.Wallet)
	if err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.wallet file %s failed to load: %v", c.Wallet, err))
	} else if err := w.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("mdl_exchanger.wallet file %s is invalid: %v", c.Wallet, err))
	}

	return errs
}

// Web config for the teller HTTP interface
type Web struct {
	HTTPAddr         string        `mapstructure:"http_addr"`
	HTTPSAddr        string        `mapstructure:"https_addr"`
	StaticDir        string        `mapstructure:"static_dir"`
	AutoTLSHost      string        `mapstructure:"auto_tls_host"`
	TLSCert          string        `mapstructure:"tls_cert"`
	TLSKey           string        `mapstructure:"tls_key"`
	ThrottleMax      int64         `mapstructure:"throttle_max"` // Maximum number of requests per duration
	ThrottleDuration time.Duration `mapstructure:"throttle_duration"`
	BehindProxy      bool          `mapstructure:"behind_proxy"`
}

// Validate validates Web config
func (c Web) Validate() error {
	if c.HTTPAddr == "" && c.HTTPSAddr == "" {
		return errors.New("at least one of web.http_addr, web.https_addr must be set")
	}

	if c.HTTPSAddr != "" && c.AutoTLSHost == "" && (c.TLSCert == "" || c.TLSKey == "") {
		return errors.New("when using web.https_addr, either web.auto_tls_host or both web.tls_cert and web.tls_key must be set")
	}

	if (c.TLSCert == "" && c.TLSKey != "") || (c.TLSCert != "" && c.TLSKey == "") {
		return errors.New("web.tls_cert and web.tls_key must be set or unset together")
	}

	if c.AutoTLSHost != "" && (c.TLSKey != "" || c.TLSCert != "") {
		return errors.New("either use web.auto_tls_host or both web.tls_key and web.tls_cert")
	}

	if c.HTTPSAddr == "" && (c.AutoTLSHost != "" || c.TLSKey != "" || c.TLSCert != "") {
		return errors.New("web.auto_tls_host or web.tls_key or web.tls_cert is set but web.https_addr is not enabled")
	}

	return nil
}

// AdminPanel config for the admin panel AdminPanel
type AdminPanel struct {
	Host             string `mapstructure:"host"`
	FixBtcValue      int64  `mapstructure:"fix_btc_value"`
	FixEthValue      int64  `mapstructure:"fix_eth_value"`
	FixSkyValue      int64  `mapstructure:"fix_sky_value"`
	FixWavesValue    int64  `mapstructure:"fix_waves_value"`
	FixWavesMDLValue int64  `mapstructure:"fix_waves_mdl_value"`
	FixMdlValue      int64  `mapstructure:"fix_mdl_value"`
	FixUsdValue      string `mapstructure:"fix_usd_value"`
	FixTxValue       int64  `mapstructure:"fix_tx_value"`
}

// Dummy config for the fake sender and scanner
type Dummy struct {
	Scanner  bool   `mapstructure:"scanner"`
	Sender   bool   `mapstructure:"sender"`
	HTTPAddr string `mapstructure:"http_addr"`
}

// Redacted returns a copy of the config with sensitive information redacted
func (c Config) Redacted() Config {
	if c.BtcRPC.User != "" {
		c.BtcRPC.User = "<redacted>"
	}

	if c.BtcRPC.Pass != "" {
		c.BtcRPC.Pass = "<redacted>"
	}

	return c
}

// Validate validates the config
func (c Config) Validate() error {
	var errs []string
	oops := func(err string) {
		errs = append(errs, err)
	}

	if c.BtcAddresses == "" {
		oops("btc_addresses missing")
	}
	if _, err := os.Stat(c.BtcAddresses); os.IsNotExist(err) {
		oops("btc_addresses file does not exist")
	}
	if c.EthAddresses == "" {
		oops("eth_addresses missing")
	}
	if _, err := os.Stat(c.EthAddresses); os.IsNotExist(err) {
		oops("eth_addresses file does not exist")
	}
	if c.SkyAddresses == "" {
		oops("sky_addresses missing")
	}
	if _, err := os.Stat(c.SkyAddresses); os.IsNotExist(err) {
		oops("sky_addresses file does not exist")
	}
	if c.WavesAddresses == "" {
		oops("waves_addresses missing")
	}
	if _, err := os.Stat(c.WavesAddresses); os.IsNotExist(err) {
		oops("waves_addresses file does not exist")
	}
	if c.WavesMDLAddresses == "" {
		oops("waves_mdl_addresses missing")
	}
	if _, err := os.Stat(c.WavesMDLAddresses); os.IsNotExist(err) {
		oops("waves_mdl_addresses file does not exist")
	}

	if !c.Dummy.Sender {
		if c.MDLRPC.Address == "" {
			oops("mdl_rpc.address missing")
		}

		// test if mdl node rpc service is reachable
		conn, err := net.Dial("tcp", c.MDLRPC.Address)
		if err != nil {
			oops(fmt.Sprintf("mdl_rpc.address connect failed: %v", err))
		} else {
			if err := conn.Close(); err != nil {
				log.Printf("Failed to close test connection to mdl_rpc.address: %v", err)
			}
		}
	}

	if !c.Dummy.Scanner {
		if c.BtcRPC.Enabled {
			if c.BtcRPC.Server == "" {
				oops("btc_rpc.server missing")
			}

			if c.BtcRPC.User == "" {
				oops("btc_rpc.user missing")
			}
			if c.BtcRPC.Pass == "" {
				oops("btc_rpc.pass missing")
			}
			if c.BtcRPC.Cert == "" {
				oops("btc_rpc.cert missing")
			}

			if _, err := os.Stat(c.BtcRPC.Cert); os.IsNotExist(err) {
				oops("btc_rpc.cert file does not exist")
			}
		}
		if c.EthRPC.Enabled {
			if c.EthRPC.Server == "" {
				oops("eth_rpc.server missing")
			}
			if c.EthRPC.Port == "" {
				oops("eth_rpc.port missing")
			}
		}

		if c.SkyRPC.Enabled {
			if c.SkyRPC.Server == "" {
				oops("sky_rpc.server missing")
			}
			if c.SkyRPC.Port == "" {
				oops("sky_rpc.port missing")
			}
		}

		if c.WavesRPC.Enabled {
			if c.WavesRPC.Server == "" {
				oops("waves_rpc.server missing")
			}
			if c.WavesRPC.Port == "" {
				oops("waves_rpc.port missing")
			}
			if c.WavesRPC.Protocol == "" {
				oops("waves_rpc.protocol missing")
			}
		}

		if c.WavesMDLRPC.Enabled {
			if c.WavesMDLRPC.Server == "" {
				oops("waves_mdl_rpc.server missing")
			}
			if c.WavesMDLRPC.Port == "" {
				oops("waves_mdl_rpc.port missing")
			}
			if c.WavesMDLRPC.Protocol == "" {
				oops("waves_mdl_rpc.protocol missing")
			}
		}

	}

	if c.BtcScanner.ConfirmationsRequired < 0 {
		oops("btc_scanner.confirmations_required must be >= 0")
	}
	if c.BtcScanner.InitialScanHeight < 0 {
		oops("btc_scanner.initial_scan_height must be >= 0")
	}
	if c.EthScanner.ConfirmationsRequired < 0 {
		oops("eth_scanner.confirmations_required must be >= 0")
	}
	if c.EthScanner.InitialScanHeight < 0 {
		oops("eth_scanner.initial_scan_height must be >= 0")
	}

	if c.SkyScanner.ConfirmationsRequired < 0 {
		oops("sky_scanner.confirmations_required must be >= 0")
	}
	if c.SkyScanner.InitialScanHeight < 0 {
		oops("sky_scanner.initial_scan_height must be >= 0")
	}

	if c.WavesScanner.ConfirmationsRequired < 0 {
		oops("waves_scanner.confirmations_required must be >= 0")
	}
	if c.WavesScanner.InitialScanHeight < 0 {
		oops("waves_scanner.initial_scan_height must be >= 0")
	}

	if c.WavesMDLScanner.ConfirmationsRequired < 0 {
		oops("waves_mdl_scanner.confirmations_required must be >= 0")
	}
	if c.WavesMDLScanner.InitialScanHeight < 0 {
		oops("waves_mdl_scanner.initial_scan_height must be >= 0")
	}

	exchangeErrs := c.MDLExchanger.validate()
	for _, err := range exchangeErrs {
		oops(err.Error())
	}

	if !c.Dummy.Sender {
		exchangeErrs := c.MDLExchanger.validateWallet()
		for _, err := range exchangeErrs {
			oops(err.Error())
		}
	}

	if err := c.Web.Validate(); err != nil {
		oops(err.Error())
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.New(strings.Join(errs, "\n"))
}

func setDefaults() {
	// Top-level args
	viper.SetDefault("profile", false)
	viper.SetDefault("debug", true)
	viper.SetDefault("logfile", "./teller.log")
	viper.SetDefault("dbfile", "teller.db")

	// Teller
	viper.SetDefault("teller.max_bound_btc_addrs", 2)

	// MDLRPC
	viper.SetDefault("mdl_rpc.address", "127.0.0.1:6430")

	// BtcRPC
	viper.SetDefault("btc_rpc.server", "127.0.0.1:8334")
	viper.SetDefault("btc_rpc.enabled", true)

	// EthRPC
	viper.SetDefault("eth_rpc.enabled", false)

	// SkyRPC
	viper.SetDefault("sky_rpc.enabled", false)

	// WavesRPC
	viper.SetDefault("waves_rpc.enabled", false)

	// WavesMDLRPC
	viper.SetDefault("waves_mdl_rpc.enabled", false)

	// BtcScanner
	viper.SetDefault("btc_scanner.scan_period", time.Second*20)
	viper.SetDefault("btc_scanner.initial_scan_height", int64(492478))
	viper.SetDefault("btc_scanner.confirmations_required", int64(1))

	// MDLExchanger
	viper.SetDefault("mdl_exchanger.tx_confirmation_check_wait", time.Second*5)
	viper.SetDefault("mdl_exchanger.max_decimals", 3)
	viper.SetDefault("mdl_exchanger.buy_method", BuyMethodDirect)

	// MDLExchanger BTC
	viper.SetDefault("mdl_exchanger.mdl_btc_exchange_enabled", false)

	// MDLExchanger ETH
	viper.SetDefault("mdl_exchanger.mdl_eth_exchange_enabled", false)

	// MDLExchanger SKY
	viper.SetDefault("mdl_exchanger.mdl_sky_exchange_enabled", false)

	// MDLExchanger WAVES
	viper.SetDefault("mdl_exchanger.mdl_waves_exchange_enabled", false)

	// MDLExchanger WAVES MDL
	viper.SetDefault("mdl_exchanger.mdl_waves_mdl_exchange_enabled", false)

	// Web
	viper.SetDefault("web.bind_enabled", true)
	viper.SetDefault("web.send_enabled", true)
	viper.SetDefault("web.http_addr", "127.0.0.1:7071")
	viper.SetDefault("web.static_dir", "./web/build")
	viper.SetDefault("web.throttle_max", int64(60))
	viper.SetDefault("web.throttle_duration", time.Minute)

	// AdminPanel
	viper.SetDefault("admin_panel.host", "127.0.0.1:7711")
	viper.SetDefault("admin_panel.fix_btc_value", 0)
	viper.SetDefault("admin_panel.fix_eth_value", 0)
	viper.SetDefault("admin_panel.fix_sky_value", 0)
	viper.SetDefault("admin_panel.fix_waves_value", 0)
	viper.SetDefault("admin_panel.fix_mdl_value", 0)
	viper.SetDefault("admin_panel.fix_usd_value", "0")
	viper.SetDefault("admin_panel.fix_tx_value", 0)

	// DummySender
	viper.SetDefault("dummy.http_addr", "127.0.0.1:4121")
	viper.SetDefault("dummy.scanner", false)
	viper.SetDefault("dummy.sender", false)
}

// Load loads the configuration from "./$configName.*" where "*" is a
// JSON, toml or yaml file (toml preferred).
func Load(configName, appDir string) (Config, error) {
	if strings.HasSuffix(configName, ".toml") {
		configName = configName[:len(configName)-len(".toml")]
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath(appDir)
	viper.AddConfigPath(".")

	setDefaults()

	cfg := Config{}

	if err := viper.ReadInConfig(); err != nil {
		return cfg, err
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}
