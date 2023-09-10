package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sudores/firefly-iii-bank-sync/bank/mono"
	"github.com/sudores/firefly-iii-bank-sync/cnf"
	firelfyiii "github.com/sudores/firefly-iii-bank-sync/dest/fireflyiii"
)

func main() {

	// Getting configuration
	cfg, err := cnf.Parse()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logging")
	}

	// Setup logging
	loggingInit(cfg.LogLevel)
	log.Info().Msg("Logging setup success")

	exit := make(chan os.Signal)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)

	ffi := firelfyiii.NewFireflyiiiConnection(cfg.FFIToken, cfg.FFIURL)
	ffiCtx, ffiCancel := context.WithCancel(context.Background())
	defer ffiCancel()
	go func() {
		log.Info().Msg("Firefly-iii starting serving")
		ffi.Serve(ffiCtx)
		log.Fatal().Err(err).Msg("Firefly serve failed")
	}()

	mb := mono.NewMonoConnetion(cfg.MonobankAPIToken, cfg.FBSHost, cfg.ListenAddr)
	go func() {
		log.Info().Msg("Monobank starting serving")
		mb.Serve()
	}()
	go func() {

		for v := range mb.TransactionChan {
			log.Debug().Msg("main Creating transaction")
			ffi.FireflyiiiTransactionChan <- v
			log.Debug().Msg("main Created transaction")
		}

	}()

	osSig := <-exit
	log.Info().Msgf("%s received. Shutting down...", osSig.String())

	// TODO: Add contexts cancellation
	mbCtx, mbCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer mbCancel()
	if err := mb.Shutdown(mbCtx); err != nil {
		log.Fatal().Err(err).Msg("Monobank shutdown failed with error")
	}

	log.Info().Msg("Shutdown successful. Bye!!!")
}

// loggingInit setups the logging of whole bot
func loggingInit(logLevel string) {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Msgf(`Log level "%s" is unrecognized. Eligible log levels are: trace, debug, info, err, fatal, panic`, logLevel)
	}
	zerolog.SetGlobalLevel(level)
	log.Debug().Msg("Logger initialized")
}
