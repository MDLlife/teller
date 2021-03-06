// MDL teller, which provides service of monitoring deposits in different crypto
// and sending mdl coins
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
	"github.com/google/gops/agent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/MDLlife/teller/src/addrs"
	"github.com/MDLlife/teller/src/config"
	"github.com/MDLlife/teller/src/exchange"
	"github.com/MDLlife/teller/src/monitor"
	"github.com/MDLlife/teller/src/scanner"
	"github.com/MDLlife/teller/src/sender"
	"github.com/MDLlife/teller/src/teller"
	"github.com/MDLlife/teller/src/util"
	"github.com/MDLlife/teller/src/util/logger"
	"github.com/MDLlife/teller/src/util/mathutil"
	"github.com/shopspring/decimal"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createBtcScanner(log logrus.FieldLogger, cfg config.Config, scanStore *scanner.Store) (*scanner.BTCScanner, error) {
	// create btc rpc client
	certs, err := ioutil.ReadFile(cfg.BtcRPC.Cert)
	if err != nil {
		return nil, fmt.Errorf("Failed to read cfg.BtcRPC.Cert %s: %v", cfg.BtcRPC.Cert, err)
	}

	log.Info("Connecting to btcd")

	btcrpc, err := btcrpcclient.New(&btcrpcclient.ConnConfig{
		Endpoint:     "ws",
		Host:         cfg.BtcRPC.Server,
		User:         cfg.BtcRPC.User,
		Pass:         cfg.BtcRPC.Pass,
		Certificates: certs,
	}, nil)
	if err != nil {
		log.WithError(err).Error("Connect btcd failed")
		return nil, err
	}

	log.Info("Connect to btcd succeeded")

	err = scanStore.AddSupportedCoin(scanner.CoinTypeBTC)
	if err != nil {
		log.WithError(err).Error("scanStore.AddSupportedCoin(scanner.CoinTypeBTC) failed")
		return nil, err
	}

	btcScanner, err := scanner.NewBTCScanner(log, scanStore, btcrpc, scanner.Config{
		ScanPeriod:            cfg.BtcScanner.ScanPeriod,
		ConfirmationsRequired: cfg.BtcScanner.ConfirmationsRequired,
		InitialScanHeight:     cfg.BtcScanner.InitialScanHeight,
	})
	if err != nil {
		log.WithError(err).Error("Open btcScanner service failed")
		return nil, err
	}
	return btcScanner, nil
}

func createEthScanner(log logrus.FieldLogger, cfg config.Config, scanStore *scanner.Store) (*scanner.ETHScanner, error) {
	ethrpc, err := scanner.NewEthClient(cfg.EthRPC.Server, cfg.EthRPC.Port)
	if err != nil {
		log.WithError(err).Error("Connect geth failed")
		return nil, err
	}

	err = scanStore.AddSupportedCoin(scanner.CoinTypeETH)
	if err != nil {
		log.WithError(err).Error("scanStore.AddSupportedCoin(scanner.CoinTypeETH) failed")
		return nil, err
	}

	ethScanner, err := scanner.NewETHScanner(log, scanStore, ethrpc, scanner.Config{
		ScanPeriod:            cfg.EthScanner.ScanPeriod,
		ConfirmationsRequired: cfg.EthScanner.ConfirmationsRequired,
		InitialScanHeight:     cfg.EthScanner.InitialScanHeight,
	})
	if err != nil {
		log.WithError(err).Error("Open ethScanner service failed")
		return nil, err
	}
	return ethScanner, nil
}

func createSkyScanner(log logrus.FieldLogger, cfg config.Config, scanStore *scanner.Store) (*scanner.SKYScanner, error) {
	skyrpc := scanner.NewSkyClient(cfg.SkyRPC.Server, cfg.SkyRPC.Port)

	err := scanStore.AddSupportedCoin(scanner.CoinTypeSKY)
	if err != nil {
		log.WithError(err).Error("scanStore.AddSupportedCoin(scanner.CoinTypeSKY) failed")
		return nil, err
	}

	skyScanner, err := scanner.NewSkycoinScanner(log, scanStore, skyrpc, scanner.Config{
		ScanPeriod:            cfg.SkyScanner.ScanPeriod,
		ConfirmationsRequired: cfg.SkyScanner.ConfirmationsRequired,
		InitialScanHeight:     cfg.SkyScanner.InitialScanHeight,
	})
	if err != nil {
		log.WithError(err).Error("Open skyScanner service failed")
		return nil, err
	}
	return skyScanner, nil
}

func createWAVESScanner(log logrus.FieldLogger, cfg config.Config, scanStore *scanner.Store) (*scanner.WAVESScanner, error) {
	url := fmt.Sprintf("%s://%s:%s", cfg.WavesRPC.Protocol, cfg.WavesRPC.Server, cfg.WavesRPC.Port)
	log.Debug("createWAVESScanner URL, ", url)
	wavesrpc := scanner.NewWavesClient(url)

	err := scanStore.AddSupportedCoin(scanner.CoinTypeWAVES)
	if err != nil {
		log.WithError(err).Error("scanStore.AddSupportedCoin(scanner.CoinTypeWAVES) failed")
		return nil, err
	}

	wavesScanner, err := scanner.NewWavescoinScanner(log, scanStore, wavesrpc, scanner.Config{
		ScanPeriod:            cfg.WavesScanner.ScanPeriod,
		ConfirmationsRequired: cfg.WavesScanner.ConfirmationsRequired,
		InitialScanHeight:     cfg.WavesScanner.InitialScanHeight,
	})
	if err != nil {
		log.WithError(err).Error("Open wavesScanner service failed")
		return nil, err
	}
	return wavesScanner, nil
}

func createWAVESMDLScanner(log logrus.FieldLogger, cfg config.Config, scanStore *scanner.Store) (*scanner.WAVESMDLScanner, error) {
	url := fmt.Sprintf("%s://%s:%s", cfg.WavesMDLRPC.Protocol, cfg.WavesMDLRPC.Server, cfg.WavesMDLRPC.Port)
	log.Debug("createWAVESMDLScanner URL, ", url)
	wavesrpc := scanner.NewWavesClient(url)

	err := scanStore.AddSupportedCoin(scanner.CoinTypeWAVESMDL)
	if err != nil {
		log.WithError(err).Error("scanStore.AddSupportedCoin(scanner.CoinTypeWAVESMDL) failed")
		return nil, err
	}

	wavesMDLScanner, err := scanner.NewWavesMDLcoinScanner(log, scanStore, wavesrpc, scanner.Config{
		ScanPeriod:            cfg.WavesMDLScanner.ScanPeriod,
		ConfirmationsRequired: cfg.WavesMDLScanner.ConfirmationsRequired,
		InitialScanHeight:     cfg.WavesMDLScanner.InitialScanHeight,
	})
	if err != nil {
		log.WithError(err).Error("Open wavesMDLScanner service failed")
		return nil, err
	}
	return wavesMDLScanner, nil
}

func run() error {
	cur, err := user.Current()
	if err != nil {
		fmt.Println("Failed to get user's home directory:", err)
		return err
	}
	defaultAppDir := filepath.Join(cur.HomeDir, ".teller-mdl")

	appDirOpt := pflag.StringP("dir", "d", defaultAppDir, "application data directory")
	configNameOpt := pflag.StringP("config", "c", "config", "name of configuration file")
	pflag.Parse()

	if err := createFolderIfNotExist(*appDirOpt); err != nil {
		fmt.Println("Create application data directory failed:", err)
		return err
	}

	cfg, err := config.Load(*configNameOpt, *appDirOpt)
	if err != nil {
		return fmt.Errorf("Config error:\n%v", err)
	}

	// Init logger
	rusloggger, err := logger.NewLogger(cfg.LogFilename, cfg.Debug)
	if err != nil {
		fmt.Println("Failed to create Logrus logger:", err)
		return err
	}

	log := rusloggger.WithField("prefix", "teller")

	log.WithField("config", cfg.Redacted()).Info("Loaded teller config")

	if cfg.Profile {
		// Start gops agent, for profiling
		if err := agent.Listen(&agent.Options{
			NoShutdownCleanup: true,
		}); err != nil {
			log.WithError(err).Error("Start profile agent failed")
			return err
		}
	}

	quit := make(chan struct{})
	go catchInterrupt(quit)

	// Open db
	dbPath := filepath.Join(*appDirOpt, cfg.DBFilename)
	db, err := bolt.Open(dbPath, 0700, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		log.WithError(err).Error("Open db failed")
		return err
	}

	errC := make(chan error, 20)
	var wg sync.WaitGroup

	background := func(name string, errC chan<- error, f func() error) {
		log.Infof("Backgrounding task %s", name)
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := f()
			if err != nil {
				log.WithError(err).Errorf("Backgrounded task %s failed", name)
				errC <- fmt.Errorf("Backgrounded task %s failed: %v", name, err)
			} else {
				log.Infof("Backgrounded task %s shutdown", name)
			}
		}()
	}

	var btcScanner *scanner.BTCScanner
	var ethScanner *scanner.ETHScanner
	var skyScanner *scanner.SKYScanner
	var wavesScanner *scanner.WAVESScanner
	var wavesMDLScanner *scanner.WAVESMDLScanner

	var scanService scanner.Scanner
	var scanEthService scanner.Scanner
	var scanSkyService scanner.Scanner
	var scanWavesService scanner.Scanner
	var scanWavesMDLService scanner.Scanner

	var sendService *sender.SendService
	var sendRPC sender.Sender

	var btcAddrMgr *addrs.Addrs
	var ethAddrMgr *addrs.Addrs
	var skyAddrMgr *addrs.Addrs
	var wavesAddrMgr *addrs.Addrs
	var wavesMDLAddrMgr *addrs.Addrs

	// create multiplexer to manage scanner
	multiplexer := scanner.NewMultiplexer(log)

	dummyMux := http.NewServeMux()

	// create scan storer
	scanStore, err := scanner.NewStore(log, db)
	if err != nil {
		log.WithError(err).Error("scanner.NewStore failed")
		return err
	}

	if cfg.Dummy.Scanner {
		log.Info("btcd disabled, running dummy scanner")
		scanService = scanner.NewDummyScanner(log)
		scanService.(*scanner.DummyScanner).RegisterCoinType(scanner.CoinTypeBTC)
		// TODO -- refactor dummy scanning to support multiple coin types
		// scanEthService = scanner.NewDummyScanner(log)
		scanService.(*scanner.DummyScanner).BindHandlers(dummyMux)
	} else {
		// enable btc scanner
		if cfg.BtcRPC.Enabled {
			btcScanner, err = createBtcScanner(rusloggger, cfg, scanStore)
			if err != nil {
				log.WithError(err).Error("create btc scanner failed")
				return err
			}
			background("btcScanner.Run", errC, btcScanner.Run)

			scanService = btcScanner

			if err := multiplexer.AddScanner(scanService, scanner.CoinTypeBTC); err != nil {
				log.WithError(err).Errorf("multiplexer.AddScanner of %s failed", scanner.CoinTypeBTC)
				return err
			}
		}

		// enable eth scanner
		if cfg.EthRPC.Enabled {
			ethScanner, err = createEthScanner(rusloggger, cfg, scanStore)
			if err != nil {
				log.WithError(err).Error("create eth scanner failed")
				return err
			}

			background("ethScanner.Run", errC, ethScanner.Run)

			scanEthService = ethScanner

			if err := multiplexer.AddScanner(scanEthService, scanner.CoinTypeETH); err != nil {
				log.WithError(err).Errorf("multiplexer.AddScanner of %s failed", scanner.CoinTypeETH)
				return err
			}
		}

		// enable sky scanner
		if cfg.SkyRPC.Enabled {
			skyScanner, err = createSkyScanner(rusloggger, cfg, scanStore)
			if err != nil {
				log.WithError(err).Error("create sky scanner failed")
				return err
			}

			background("skyScanner.Run", errC, skyScanner.Run)

			scanSkyService = skyScanner

			if err := multiplexer.AddScanner(scanSkyService, scanner.CoinTypeSKY); err != nil {
				log.WithError(err).Errorf("multiplexer.AddScanner of %s failed", scanner.CoinTypeSKY)
				return err
			}
		}

		// enable waves scanner
		if cfg.WavesRPC.Enabled {
			wavesScanner, err = createWAVESScanner(rusloggger, cfg, scanStore)
			if err != nil {
				log.WithError(err).Error("create waves scanner failed")
				return err
			}

			background("wavesScanner.Run", errC, wavesScanner.Run)

			scanWavesService = wavesScanner

			if err := multiplexer.AddScanner(scanWavesService, scanner.CoinTypeWAVES); err != nil {
				log.WithError(err).Errorf("multiplexer.AddScanner of %s failed", scanner.CoinTypeWAVES)
				return err
			}
		}

		// enable waves MDL scanner
		if cfg.WavesMDLRPC.Enabled {
			wavesMDLScanner, err = createWAVESMDLScanner(rusloggger, cfg, scanStore)
			if err != nil {
				log.WithError(err).Error("create wavesMDL scanner failed")
				return err
			}

			background("wavesMDLScanner.Run", errC, wavesMDLScanner.Run)

			scanWavesMDLService = wavesMDLScanner

			if err := multiplexer.AddScanner(scanWavesMDLService, scanner.CoinTypeWAVESMDL); err != nil {
				log.WithError(err).Errorf("multiplexer.AddScanner of %s failed", scanner.CoinTypeWAVESMDL)
				return err
			}
		}

	}

	background("multiplex.Run", errC, multiplexer.Multiplex)

	if cfg.Dummy.Sender {
		log.Info("mdld disabled, running dummy sender")
		sendRPC = sender.NewDummySender(log)
		sendRPC.(*sender.DummySender).BindHandlers(dummyMux)
	} else {
		mdlClient, err := sender.NewAPI(cfg.MDLExchanger.Wallet, cfg.MDLRPC.Address)
		if err != nil {
			log.WithError(err).Error("sender.NewAPI failed")
			return err
		}

		sendService = sender.NewService(log, mdlClient)

		background("sendService.Run", errC, sendService.Run)

		sendRPC = sender.NewRetrySender(sendService)
	}

	if cfg.Dummy.Scanner || cfg.Dummy.Sender {
		log.Infof("Starting dummy admin interface listener on http://%s", cfg.Dummy.HTTPAddr)
		go func() {
			if err := http.ListenAndServe(cfg.Dummy.HTTPAddr, dummyMux); err != nil {
				log.WithError(err).Error("Dummy ListenAndServe failed")
			}
		}()
	}

	// create exchange service
	exchangeStore, err := exchange.NewStore(log, db)
	if err != nil {
		log.WithError(err).Error("exchange.NewStore failed")
		return err
	}

	var exchangeClient *exchange.Exchange

	switch cfg.MDLExchanger.BuyMethod {
	case config.BuyMethodDirect:
		var err error
		exchangeClient, err = exchange.NewDirectExchange(log, cfg.MDLExchanger, exchangeStore, multiplexer, sendRPC)
		if err != nil {
			log.WithError(err).Error("exchange.NewDirectExchange failed")
			return err
		}
	case config.BuyMethodPassthrough:
		var err error
		exchangeClient, err = exchange.NewPassthroughExchange(log, cfg.MDLExchanger, exchangeStore, multiplexer, sendRPC)
		if err != nil {
			log.WithError(err).Error("exchange.NewPassthroughExchange failed")
			return err
		}
	default:
		log.WithError(config.ErrInvalidBuyMethod).Error()
		return config.ErrInvalidBuyMethod
	}

	background("exchangeClient.Run", errC, exchangeClient.Run)

	// create AddrManager
	addrManager := addrs.NewAddrManager()

	if cfg.BtcRPC.Enabled {
		// create bitcoin address manager
		r, err := util.LoadFileToReader(cfg.BtcAddresses)
		if err != nil {
			log.WithError(err).Error("Load deposit bitcoin address list failed")
			return err
		}

		btcAddrMgr, err = addrs.NewBTCAddrs(log, db, r)
		if err != nil {
			log.WithError(err).Error("Create bitcoin deposit address manager failed")
			return err
		}
		if err := addrManager.PushGenerator(btcAddrMgr, scanner.CoinTypeBTC); err != nil {
			log.WithError(err).Error("add btc address manager failed")
			return err
		}
	}

	if cfg.EthRPC.Enabled {
		// create ethcoin address manager
		r, err := util.LoadFileToReader(cfg.EthAddresses)
		if err != nil {
			log.WithError(err).Error("Load deposit ethcoin address list failed")
			return err
		}

		ethAddrMgr, err = addrs.NewETHAddrs(log, db, r)
		if err != nil {
			log.WithError(err).Error("Create ethcoin deposit address manager failed")
			return err
		}
		if err := addrManager.PushGenerator(ethAddrMgr, scanner.CoinTypeETH); err != nil {
			log.WithError(err).Error("add eth address manager failed")
			return err
		}
	}

	if cfg.SkyRPC.Enabled {
		// create skycoin address manager
		r, err := util.LoadFileToReader(cfg.SkyAddresses)
		if err != nil {
			log.WithError(err).Error("Load deposit skycoin address list failed")
			return err
		}

		skyAddrMgr, err = addrs.NewSKYAddrs(log, db, r)
		if err != nil {
			log.WithError(err).Error("Create skycoin deposit address manager failed")
			return err
		}
		if err := addrManager.PushGenerator(skyAddrMgr, scanner.CoinTypeSKY); err != nil {
			log.WithError(err).Error("add sky address manager failed")
			return err
		}
	}

	if cfg.WavesRPC.Enabled {
		// create skycoin address manager
		r, err := util.LoadFileToReader(cfg.WavesAddresses)
		if err != nil {
			log.WithError(err).Error("Load deposit waves address list failed")
			return err
		}

		wavesAddrMgr, err = addrs.NewWAVESAddrs(log, db, r)
		if err != nil {
			log.WithError(err).Error("Create waves deposit address manager failed")
			return err
		}
		if err := addrManager.PushGenerator(wavesAddrMgr, scanner.CoinTypeWAVES); err != nil {
			log.WithError(err).Error("add waves address manager failed")
			return err
		}
	}

	if cfg.WavesMDLRPC.Enabled {
		// create skycoin address manager
		r, err := util.LoadFileToReader(cfg.WavesMDLAddresses)
		if err != nil {
			log.WithError(err).Error("Load deposit wavesMDL address list failed")
			return err
		}

		wavesMDLAddrMgr, err = addrs.NewWAVESAddrs(log, db, r)
		if err != nil {
			log.WithError(err).Error("Create wavesMDL deposit address manager failed")
			return err
		}
		if err := addrManager.PushGenerator(wavesMDLAddrMgr, scanner.CoinTypeWAVESMDL); err != nil {
			log.WithError(err).Error("add wavesMDL address manager failed")
			return err
		}
	}

	tellerServer := teller.New(log, exchangeClient, addrManager, cfg)

	// Run the service
	background("tellerServer.Run", errC, tellerServer.Run)

	// start monitor service
	if cfg.AdminPanel.FixUsdValue == "" {
		cfg.AdminPanel.FixUsdValue = "0"
	}
	fixUsdValue, err := mathutil.DecimalFromString(cfg.AdminPanel.FixUsdValue)
	if err != nil {
		fixUsdValue = decimal.New(0, 0)
		log.Error("Can't convert fix_usd_value: '" + cfg.AdminPanel.FixUsdValue + "' to decimal")
	}

	monitorCfg := monitor.Config{
		Addr:             cfg.AdminPanel.Host,
		FixBtcValue:      cfg.AdminPanel.FixBtcValue,
		FixEthValue:      cfg.AdminPanel.FixEthValue,
		FixSkyValue:      cfg.AdminPanel.FixSkyValue,
		FixWavesValue:    cfg.AdminPanel.FixWavesValue,
		FixWavesMDLValue: cfg.AdminPanel.FixWavesMDLValue,
		FixMdlValue:      cfg.AdminPanel.FixMdlValue,
		FixUsdValue:      fixUsdValue,
		FixTxValue:       cfg.AdminPanel.FixTxValue,
	}
	monitorService := monitor.New(log, monitorCfg, btcAddrMgr, ethAddrMgr, skyAddrMgr, wavesAddrMgr, wavesMDLAddrMgr, exchangeClient, btcScanner)

	background("monitorService.Run", errC, monitorService.Run)

	var finalErr error
	select {
	case <-quit:
	case finalErr = <-errC:
		if finalErr != nil {
			log.WithError(finalErr).Error("Goroutine error")
		}
	}

	log.Info("Shutting down...")

	if monitorService != nil {
		log.Info("Shutting down monitorService")
		monitorService.Shutdown()
	}

	// close the teller service
	log.Info("Shutting down tellerServer")
	tellerServer.Shutdown()

	log.Info("Shutting down the multiplexer")
	multiplexer.Shutdown()

	// close the scan service
	if btcScanner != nil {
		log.Info("Shutting down btcScanner")
		btcScanner.Shutdown()
	}
	// close the scan service
	if ethScanner != nil {
		log.Info("Shutting down ethScanner")
		ethScanner.Shutdown()
	}

	// close the scan service
	if skyScanner != nil {
		log.Info("Shutting down skyScanner")
		skyScanner.Shutdown()
	}

	// close the scan service
	if wavesScanner != nil {
		log.Info("Shutting down wavesScanner")
		wavesScanner.Shutdown()
	}

	// close the scan service
	if wavesMDLScanner != nil {
		log.Info("Shutting down wavesMDLScanner")
		wavesMDLScanner.Shutdown()
	}

	// close exchange service
	log.Info("Shutting down exchangeClient")
	exchangeClient.Shutdown()

	// close the mdl send service
	if sendService != nil {
		log.Info("Shutting down MDL sendService")
		sendService.Shutdown()
	}

	log.Info("Waiting for goroutines to exit")

	wg.Wait()

	log.Info("Shutdown complete")

	return finalErr
}

func createFolderIfNotExist(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// create the dir
		if err := os.Mkdir(path, 0700); err != nil {
			return err
		}
	}
	return nil
}

func printProgramStatus() {
	p := pprof.Lookup("goroutine")
	if err := p.WriteTo(os.Stdout, 2); err != nil {
		fmt.Println("ERROR:", err)
		return
	}
}

func catchInterrupt(quit chan<- struct{}) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	signal.Stop(sigchan)
	close(quit)

	// If ctrl-c is called again, panic so that the program state can be examined.
	// Ctrl-c would be called again if program shutdown was stuck.
	go catchInterruptPanic()
}

// catchInterruptPanic catches os.Interrupt and panics
func catchInterruptPanic() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	signal.Stop(sigchan)
	printProgramStatus()
	panic("SIGINT")
}
