package teller

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gz-c/tollbooth"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/secure"
	"golang.org/x/crypto/acme/autocert"

	"github.com/MDLlife/MDL/src/cipher"
	"github.com/MDLlife/MDL/src/util/droplet"

	"github.com/MDLlife/teller/src/addrs"
	"github.com/MDLlife/teller/src/config"
	"github.com/MDLlife/teller/src/exchange"
	"github.com/MDLlife/teller/src/scanner"
	"github.com/MDLlife/teller/src/sender"
	"github.com/MDLlife/teller/src/util/httputil"
	"github.com/MDLlife/teller/src/util/logger"
)

const (
	shutdownTimeout = time.Second * 5

	// https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
	// The timeout configuration is necessary for public servers, or else
	// connections will be used up
	serverReadTimeout  = time.Second * 10
	serverWriteTimeout = time.Second * 60
	serverIdleTimeout  = time.Second * 120

	// Directory where cached SSL certs from Let's Encrypt are stored
	tlsAutoCertCache = "cert-cache"
)

var (
	errInternalServerError = errors.New("Internal Server Error")
)

// HTTPServer exposes the API endpoints and static website
type HTTPServer struct {
	cfg           config.Config
	exchanger     exchange.Exchanger
	log           logrus.FieldLogger
	service       *Service
	httpListener  *http.Server
	httpsListener *http.Server
	quit          chan struct{}
	done          chan struct{}
}

// NewHTTPServer creates an HTTPServer
func NewHTTPServer(log logrus.FieldLogger, cfg config.Config, service *Service, exchanger exchange.Exchanger) *HTTPServer {
	return &HTTPServer{
		cfg: cfg.Redacted(),
		log: log.WithFields(logrus.Fields{
			"prefix": "teller.http",
		}),
		service:   service,
		exchanger: exchanger,
		quit:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

// Run runs the HTTPServer
func (s *HTTPServer) Run() error {
	log := s.log
	log.WithField("config", s.cfg).Info("HTTP service start")
	defer log.Info("HTTP service closed")
	defer close(s.done)

	var mux http.Handler = s.setupMux()

	allowedHosts := []string{} // empty array means all hosts allowed
	var sslHost string
	if s.cfg.Web.AutoTLSHost == "" {
		// Note: if AutoTLSHost is not set, but HTTPSAddr is set, then
		// http will redirect to the HTTPSAddr listening IP, which would be
		// either 127.0.0.1 or 0.0.0.0
		// When running behind a DNS name, make sure to set AutoTLSHost
		sslHost = s.cfg.Web.HTTPSAddr
	} else {
		sslHost = s.cfg.Web.AutoTLSHost
		// When using -auto-tls-host,
		// which implies automatic Let's Encrypt SSL cert generation in production,
		// restrict allowed hosts to that host.
		allowedHosts = []string{s.cfg.Web.AutoTLSHost}
	}

	if len(allowedHosts) == 0 {
		log = log.WithField("allowedHosts", "all")
	} else {
		log = log.WithField("allowedHosts", allowedHosts)
	}

	log = log.WithField("sslHost", sslHost)

	log.Info("Configured")

	secureMiddleware := configureSecureMiddleware(sslHost, allowedHosts)
	mux = secureMiddleware.Handler(mux)

	if s.cfg.Web.HTTPAddr != "" {
		s.httpListener = setupHTTPListener(s.cfg.Web.HTTPAddr, mux)
	}

	handleListenErr := func(f func() error) error {
		if err := f(); err != nil {
			select {
			case <-s.quit:
				return nil
			default:
				log.WithError(err).Error("ListenAndServe or ListenAndServeTLS error")
				return fmt.Errorf("http serve failed: %v", err)
			}
		}
		return nil
	}

	if s.cfg.Web.HTTPAddr != "" {
		log.Info(fmt.Sprintf("HTTP server listening on http://%s", s.cfg.Web.HTTPAddr))
	}
	if s.cfg.Web.HTTPSAddr != "" {
		log.Info(fmt.Sprintf("HTTPS server listening on https://%s", s.cfg.Web.HTTPSAddr))
	}

	var tlsCert, tlsKey string
	if s.cfg.Web.HTTPSAddr != "" {
		log.Info("Using TLS")

		s.httpsListener = setupHTTPListener(s.cfg.Web.HTTPSAddr, mux)

		tlsCert = s.cfg.Web.TLSCert
		tlsKey = s.cfg.Web.TLSKey

		if s.cfg.Web.AutoTLSHost != "" {
			log.Info("Using Let's Encrypt autocert")
			// https://godoc.org/golang.org/x/crypto/acme/autocert
			// https://stackoverflow.com/a/40494806
			certManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(s.cfg.Web.AutoTLSHost),
				Cache:      autocert.DirCache(tlsAutoCertCache),
			}

			s.httpsListener.TLSConfig = &tls.Config{
				GetCertificate: certManager.GetCertificate,
			}

			// These will be autogenerated by the autocert middleware
			tlsCert = ""
			tlsKey = ""
		}

	}

	return handleListenErr(func() error {
		var wg sync.WaitGroup
		errC := make(chan error)

		if s.cfg.Web.HTTPAddr != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := s.httpListener.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.WithError(err).Println("ListenAndServe error")
					errC <- err
				}
			}()
		}

		if s.cfg.Web.HTTPSAddr != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := s.httpsListener.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
					log.WithError(err).Error("ListenAndServeTLS error")
					errC <- err
				}
			}()
		}

		done := make(chan struct{})

		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case err := <-errC:
			return err
		case <-s.quit:
			return nil
		case <-done:
			return nil
		}
	})
}

func configureSecureMiddleware(sslHost string, allowedHosts []string) *secure.Secure {
	sslRedirect := true
	if sslHost == "" {
		sslRedirect = false
	}

	return secure.New(secure.Options{
		AllowedHosts: allowedHosts,
		SSLRedirect:  sslRedirect,
		SSLHost:      sslHost,

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
		// FIXME: Web frontend code has inline styles, CSP doesn't work yet
		// ContentSecurityPolicy: "default-src 'self'",

		// Set HSTS to one year, for this domain only, do not add to chrome preload list
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
		STSSeconds:           31536000, // 1 year
		STSIncludeSubdomains: false,
		STSPreload:           false,

		// Deny use in iframes
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Frame-Options
		FrameDeny: true,

		// Disable MIME sniffing in browsers
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Content-Type-Options
		ContentTypeNosniff: true,

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-XSS-Protection
		BrowserXssFilter: true,

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
		// "same-origin" is invalid in chrome
		ReferrerPolicy: "no-referrer",
	})
}

func setupHTTPListener(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}
}

func (s *HTTPServer) setupMux() *http.ServeMux {
	mux := http.NewServeMux()

	ratelimit := func(h http.Handler) http.Handler {
		limiter := tollbooth.NewLimiter(s.cfg.Web.ThrottleMax, s.cfg.Web.ThrottleDuration, nil)
		if s.cfg.Web.BehindProxy {
			limiter.SetIPLookups([]string{"X-Forwarded-For", "RemoteAddr", "X-Real-IP"})
		}
		return tollbooth.LimitHandler(limiter, h)
	}

	handleAPI := func(path string, h http.Handler) {
		// Allow requests from a local mdl wallet
		h = cors.New(cors.Options{
			//AllowedOrigins: []string{"http://127.0.0.1:8320"},
			AllowedOrigins: []string{"*"},
		}).Handler(h)

		h = gziphandler.GzipHandler(h)

		mux.Handle(path, h)
	}

	// API Methods
	handleAPI("/api/bind", ratelimit(httputil.LogHandler(s.log, BindHandler(s))))
	handleAPI("/api/status", ratelimit(httputil.LogHandler(s.log, StatusHandler(s))))
	handleAPI("/api/config", httputil.LogHandler(s.log, ConfigHandler(s)))
	handleAPI("/api/exchange-status", httputil.LogHandler(s.log, ExchangeStatusHandler(s)))

	// Static files
	mux.Handle("/", gziphandler.GzipHandler(http.FileServer(http.Dir(s.cfg.Web.StaticDir))))

	return mux
}

// Shutdown stops the HTTPServer
func (s *HTTPServer) Shutdown() {
	s.log.Info("Shutting down HTTP server(s)")
	defer s.log.Info("Shutdown HTTP server(s)")
	close(s.quit)

	var wg sync.WaitGroup
	wg.Add(2)

	shutdown := func(proto string, ln *http.Server) {
		defer wg.Done()
		if ln == nil {
			return
		}
		log := s.log.WithFields(logrus.Fields{
			"proto":   proto,
			"timeout": shutdownTimeout,
		})

		defer log.Info("Shutdown server")
		log.Info("Shutting down server")

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := ln.Shutdown(ctx); err != nil {
			log.WithError(err).Error("HTTP server shutdown error")
		}
	}

	shutdown("HTTP", s.httpListener)
	shutdown("HTTPS", s.httpsListener)

	wg.Wait()

	<-s.done
}

// BindResponse http response for /api/bind
type BindResponse struct {
	DepositAddress string `json:"deposit_address,omitempty"`
	CoinType       string `json:"coin_type,omitempty"`
	BuyMethod      string `json:"buy_method"`
}

type bindRequest struct {
	MDLAddr  string `json:"mdladdr"`
	CoinType string `json:"coin_type"`
}

// BindHandler binds mdl address with another coin address
// Method: POST
// Accept: application/json
// URI: /api/bind
// Args:
//    {"mdladdr": "...", "coin_type": "BTC"}
func BindHandler(s *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		w.Header().Set("Accept", "application/json")

		if !validMethod(ctx, w, r, []string{http.MethodPost}) {
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
			errorResponse(ctx, w, http.StatusUnsupportedMediaType, errors.New("Invalid content type"))
			return
		}

		bindReq := &bindRequest{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&bindReq); err != nil {
			err = fmt.Errorf("Invalid json request body: %v", err)
			errorResponse(ctx, w, http.StatusBadRequest, err)
			return
		}
		defer func(log logrus.FieldLogger) {
			if err := r.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to closed request body")
			}
		}(log)

		// Remove extraneous whitespace
		bindReq.MDLAddr = strings.Trim(bindReq.MDLAddr, "\n\t ")

		log = log.WithField("bindReq", bindReq)
		ctx = logger.WithContext(ctx, log)

		if bindReq.MDLAddr == "" {
			errorResponse(ctx, w, http.StatusBadRequest, errors.New("Missing mdladdr"))
			return
		}

		switch bindReq.CoinType {
		case scanner.CoinTypeBTC:
			if !s.cfg.BtcRPC.Enabled {
				errorResponse(ctx, w, http.StatusBadRequest, fmt.Errorf("Oops, there seems to be an issue. The selected coin type %s is not enabled. We are working on a fix, please try again in a couple of hours", scanner.CoinTypeBTC))
				return
			}
		case scanner.CoinTypeETH:
			if !s.cfg.EthRPC.Enabled {
				errorResponse(ctx, w, http.StatusBadRequest, fmt.Errorf("Oops, there seems to be an issue. The selected coin type %s is not enabled. We are working on a fix, please try again in a couple of hours", scanner.CoinTypeETH))
				return
			}
		case scanner.CoinTypeSKY:
			if !s.cfg.SkyRPC.Enabled {
				errorResponse(ctx, w, http.StatusBadRequest, fmt.Errorf("Oops, there seems to be an issue. The selected coin type %s is not enabled. We are working on a fix, please try again in a couple of hours", scanner.CoinTypeSKY))
				return
			}
		case scanner.CoinTypeWAVES:
			if !s.cfg.WavesRPC.Enabled {
				errorResponse(ctx, w, http.StatusBadRequest, fmt.Errorf("Oops, there seems to be an issue. The selected coin type %s is not enabled. We are working on a fix, please try again in a couple of hours", scanner.CoinTypeWAVES))
				return
			}
		case "":
			errorResponse(ctx, w, http.StatusBadRequest, errors.New("Missing coin_type"))
			return
		default:
			errorResponse(ctx, w, http.StatusBadRequest, errors.New("Invalid coin_type"))
			return
		}

		log.Info()

		if !verifyMDLAddress(ctx, w, bindReq.MDLAddr) {
			return
		}

		log.Info("Calling service.BindAddress")

		boundAddr, err := s.service.BindAddress(bindReq.MDLAddr, bindReq.CoinType)
		if err != nil {
			log.WithError(err).Error("service.BindAddress failed")
			switch err {
			case ErrBindDisabled:
				errorResponse(ctx, w, http.StatusForbidden, err)
			default:
				switch err {
				case addrs.ErrDepositAddressEmpty, ErrMaxBoundAddresses:
				default:
					err = errInternalServerError
				}
				errorResponse(ctx, w, http.StatusInternalServerError, err)
			}
			return
		}

		log = log.WithField("boundAddr", boundAddr)
		log.Infof("Bound mdl and %s addresses", bindReq.CoinType)

		if err := httputil.JSONResponse(w, BindResponse{
			DepositAddress: boundAddr.Address,
			CoinType:       boundAddr.CoinType,
			BuyMethod:      boundAddr.BuyMethod,
		}); err != nil {
			log.WithError(err).Error(err)
		}
	}
}

// StatusResponse http response for /api/status
type StatusResponse struct {
	Statuses []exchange.DepositStatus `json:"statuses,omitempty"`
}

// StatusHandler returns the deposit status of specific mdl address
// Method: GET
// URI: /api/status
// Args:
//     mdladdr
func StatusHandler(s *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		if !validMethod(ctx, w, r, []string{http.MethodGet}) {
			return
		}

		mdlAddr := r.URL.Query().Get("mdladdr")

		// Remove extraneous whitespace
		mdlAddr = strings.Trim(mdlAddr, "\n\t ")

		if mdlAddr == "" {
			errorResponse(ctx, w, http.StatusBadRequest, errors.New("Missing mdladdr"))
			return
		}

		log = log.WithField("mdlAddr", mdlAddr)
		ctx = logger.WithContext(ctx, log)

		log.Info()

		if !verifyMDLAddress(ctx, w, mdlAddr) {
			return
		}

		log.Info("Sending StatusRequest to teller")

		depositStatuses, err := s.service.GetDepositStatuses(mdlAddr)
		if err != nil {
			log.WithError(err).Error("service.GetDepositStatuses failed")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}

		log = log.WithFields(logrus.Fields{
			"depositStatuses":    depositStatuses,
			"depositStatusesLen": len(depositStatuses),
		})
		log.Info("Got depositStatuses")

		if err := httputil.JSONResponse(w, StatusResponse{
			Statuses: depositStatuses,
		}); err != nil {
			log.WithError(err).Error(err)
		}
	}
}

// ConfigResponse http response for /api/config
type ConfigResponse struct {
	Enabled                  bool                     `json:"enabled"`
	Available                float64                  `json:"available"`
	BtcConfirmationsRequired int64                    `json:"btc_confirmations_required"`
	EthConfirmationsRequired int64                    `json:"eth_confirmations_required"`
	MaxBoundAddresses        int                      `json:"max_bound_addrs"`
	MDLBtcExchangeRate       string                   `json:"mdl_btc_exchange_rate"`
	MDLEthExchangeRate       string                   `json:"mdl_eth_exchange_rate"`
	MDLSkyExchangeRate       string                   `json:"mdl_sky_exchange_rate"`
	MDLWavesExchangeRate     string                   `json:"mdl_waves_exchange_rate"`
	MDLWavesMDLExchangeRate  string                   `json:"mdl_waves_mdl_exchange_rate"`
	MaxDecimals              int                      `json:"max_decimals"`
	Supported                []config.SupportedCrypto `json:"supported"`
}

// ConfigHandler returns the teller configuration
// Method: GET
// URI: /api/config
func ConfigHandler(s *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		if !validMethod(ctx, w, r, []string{http.MethodGet}) {
			return
		}

		// Convert the exchange rate to a mdl balance string
		rate := s.cfg.MDLExchanger.MDLBtcExchangeRate
		maxDecimals := s.cfg.MDLExchanger.MaxDecimals
		dropletsPerBTC, err := exchange.CalculateBtcMDLValue(exchange.SatoshisPerBTC, rate, maxDecimals)
		if err != nil {
			log.WithError(err).Error("exchange.CalculateBtcMDLValue failed")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}

		mdlPerBTC, err := droplet.ToString(dropletsPerBTC)
		if err != nil {
			log.WithError(err).Error("droplet.ToString failed dropletsPerBTC")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}
		rate = s.cfg.MDLExchanger.MDLEthExchangeRate
		dropletsPerETH, err := exchange.CalculateEthMDLValue(big.NewInt(exchange.WeiPerETH), rate, maxDecimals)
		if err != nil {
			log.WithError(err).Error("exchange.CalculateEthMDLValue failed")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}
		mdlPerETH, err := droplet.ToString(dropletsPerETH)
		if err != nil {
			log.WithError(err).Error("droplet.ToString failed dropletsPerETH")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}

		rate = s.cfg.MDLExchanger.MDLSkyExchangeRate
		dropletsPerSKY, err := exchange.CalculateSkyMDLValue(exchange.DropletsPerSKY, rate, maxDecimals)
		if err != nil {
			log.WithError(err).Error("exchange.CalculateSkyMDLValue failed")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}
		mdlPerSKY, err := droplet.ToString(dropletsPerSKY)
		if err != nil {
			log.WithError(err).Error("droplet.ToString failed dropletsPerSKY")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}

		rate = s.cfg.MDLExchanger.MDLWavesExchangeRate
		dropletsPerWAVES, err := exchange.CalculateWavesMDLValue(exchange.DropletsPerWAVES, rate, maxDecimals)
		if err != nil {
			log.WithError(err).Error("exchange.CalculateWavesMDLValue failed")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}
		mdlPerWAVES, err := droplet.ToString(dropletsPerWAVES)
		if err != nil {
			log.WithError(err).Error("droplet.ToString failed CalculateWavesMDLValue")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}

		rate = s.cfg.MDLExchanger.MDLWavesMDLExchangeRate
		dropletsPerWAVESMDL, err := exchange.CalculateWavesMDLValue(exchange.DropletsPerWAVES, rate, maxDecimals)
		if err != nil {
			log.WithError(err).Error("exchange.CalculateWavesMDLValue failed")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}
		mdlPerWAVESMDL, err := droplet.ToString(dropletsPerWAVESMDL)
		if err != nil {
			log.WithError(err).Error("droplet.ToString failed CalculateWavesMDLValue")
			errorResponse(ctx, w, http.StatusInternalServerError, errInternalServerError)
			return
		}

		supportedCrypto := []config.SupportedCrypto{
			{
				Name:            s.cfg.MDLExchanger.MDLBtcExchangeName,
				ExchangeRate:    s.cfg.MDLExchanger.MDLBtcExchangeRate,
				ExchangeRateUSD: s.cfg.MDLExchanger.MDLBtcExchangeRateUSD,
				Label:           s.cfg.MDLExchanger.MDLBtcExchangeLabel,
				Enabled:         s.cfg.MDLExchanger.MDLBtcExchangeEnabled,
			},
			{
				Name:            s.cfg.MDLExchanger.MDLEthExchangeName,
				ExchangeRate:    s.cfg.MDLExchanger.MDLEthExchangeRate,
				ExchangeRateUSD: s.cfg.MDLExchanger.MDLEthExchangeRateUSD,
				Label:           s.cfg.MDLExchanger.MDLEthExchangeLabel,
				Enabled:         s.cfg.MDLExchanger.MDLEthExchangeEnabled,
			},
			{
				Name:            s.cfg.MDLExchanger.MDLSkyExchangeName,
				ExchangeRate:    s.cfg.MDLExchanger.MDLSkyExchangeRate,
				ExchangeRateUSD: s.cfg.MDLExchanger.MDLSkyExchangeRateUSD,
				Label:           s.cfg.MDLExchanger.MDLSkyExchangeLabel,
				Enabled:         s.cfg.MDLExchanger.MDLSkyExchangeEnabled,
			},
			{
				Name:            s.cfg.MDLExchanger.MDLWavesExchangeName,
				ExchangeRate:    s.cfg.MDLExchanger.MDLWavesExchangeRate,
				ExchangeRateUSD: s.cfg.MDLExchanger.MDLWavesExchangeRateUSD,
				Label:           s.cfg.MDLExchanger.MDLWavesExchangeLabel,
				Enabled:         s.cfg.MDLExchanger.MDLWavesExchangeEnabled,
			},
			{
				Name:            s.cfg.MDLExchanger.MDLWavesMDLExchangeName,
				ExchangeRate:    s.cfg.MDLExchanger.MDLWavesMDLExchangeRate,
				ExchangeRateUSD: s.cfg.MDLExchanger.MDLWavesMDLExchangeRateUSD,
				Label:           s.cfg.MDLExchanger.MDLWavesMDLExchangeLabel,
				Enabled:         s.cfg.MDLExchanger.MDLWavesMDLExchangeEnabled,
			},
		}

		balance := 0.0
		if b, err := s.exchanger.Balance(); err == nil {
			if balance, err = strconv.ParseFloat(b.Coins, 3); err != nil {
				balance = 0.0
			}
		}
		if err := httputil.JSONResponse(w, ConfigResponse{
			Enabled:                  s.cfg.Teller.BindEnabled,
			Available:                balance,
			BtcConfirmationsRequired: s.cfg.BtcScanner.ConfirmationsRequired,
			EthConfirmationsRequired: s.cfg.EthScanner.ConfirmationsRequired,

			MDLBtcExchangeRate:      mdlPerBTC,
			MDLEthExchangeRate:      mdlPerETH,
			MDLSkyExchangeRate:      mdlPerSKY,
			MDLWavesExchangeRate:    mdlPerWAVES,
			MDLWavesMDLExchangeRate: mdlPerWAVESMDL,

			MaxDecimals:       maxDecimals,
			MaxBoundAddresses: s.cfg.Teller.MaxBoundAddresses,
			Supported:         supportedCrypto,
		}); err != nil {
			log.WithError(err).Error(err)
		}
	}
}

// ExchangeStatusResponse http response for /api/exchange-status
type ExchangeStatusResponse struct {
	Error   string                        `json:"error"`
	Balance ExchangeStatusResponseBalance `json:"balance"`
}

// ExchangeStatusResponseBalance is the balance field of ExchangeStatusResponse
type ExchangeStatusResponseBalance struct {
	Coins string `json:"coins"`
	Hours string `json:"hours"`
}

// ExchangeStatusHandler returns the status of the exchanger
// Method: GET
// URI: /api/exchange-status
func ExchangeStatusHandler(s *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		if !validMethod(ctx, w, r, []string{http.MethodGet}) {
			return
		}

		errorMsg := ""
		err := s.exchanger.Status()

		// If the status is an RPCError, the most likely cause is that the
		// wallet has an insufficient balance (other causes could be a temporary
		// application error, or a bug in the mdl node).
		// Errors that are not RPCErrors are transient and common, such as
		// exchange.ErrNotConfirmed, which will happen frequently and temporarily.
		switch err.(type) {
		case sender.RPCError:
			errorMsg = err.Error()
		default:
		}

		// Get the wallet balance, but ignore any error. If an error occurs,
		// return a balance of 0
		bal, err := s.exchanger.Balance()
		coins := "0.000000"
		hours := "0"
		if err != nil {
			log.WithError(err).Error("s.exchange.Balance failed")
		} else {
			coins = bal.Coins
			hours = bal.Hours
		}

		resp := ExchangeStatusResponse{
			Error: errorMsg,
			Balance: ExchangeStatusResponseBalance{
				Coins: coins,
				Hours: hours,
			},
		}

		log.WithField("resp", resp).Info()

		if err := httputil.JSONResponse(w, resp); err != nil {
			log.WithError(err).Error(err)
		}
	}
}

func validMethod(ctx context.Context, w http.ResponseWriter, r *http.Request, allowed []string) bool {
	for _, m := range allowed {
		if r.Method == m {
			return true
		}
	}

	w.Header().Set("Allow", strings.Join(allowed, ", "))

	status := http.StatusMethodNotAllowed
	errorResponse(ctx, w, status, errors.New("Invalid request method"))

	return false
}

func verifyMDLAddress(ctx context.Context, w http.ResponseWriter, mdlAddr string) bool {
	log := logger.FromContext(ctx)

	if _, err := cipher.DecodeBase58Address(mdlAddr); err != nil {
		msg := fmt.Sprintf("Invalid mdl address: %v", err)
		httputil.ErrResponse(w, http.StatusBadRequest, msg)
		log.WithFields(logrus.Fields{
			"status":  http.StatusBadRequest,
			"mdlAddr": mdlAddr,
		}).WithError(err).Info("Invalid mdl address")
		return false
	}

	return true
}

func errorResponse(ctx context.Context, w http.ResponseWriter, code int, err error) {
	log := logger.FromContext(ctx)
	log.WithFields(logrus.Fields{
		"status":    code,
		"statusMsg": http.StatusText(code),
	}).WithError(err).Info()

	if err != errInternalServerError {
		httputil.ErrResponse(w, code, err.Error())
	} else {
		httputil.ErrResponse(w, code)
	}
}
