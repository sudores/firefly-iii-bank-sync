package main

import (
	"os"
	"sync"

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
		log.Err(err).Msg("Failed to initialize logging")
	}

	// Setup logging
	loggingInit(cfg.LogLevel)
	log.Info().Msg("Logging setup success")

	wg := sync.WaitGroup{}
	ffi := firelfyiii.NewFireflyiiiConnection(cfg.FFIToken, cfg.FFIURL)
	go func() {
		wg.Add(1)
		log.Info().Msg("Firefly-iii starting serving")
		ffi.Serve()
		log.Fatal().Err(err).Msg("Firefly serve failed")
		wg.Done()
	}()

	mb := mono.NewMonoConnetion(cfg.MonobankAPIToken, cfg.FBSHost, cfg.ListenAddr)
	go func() {
		wg.Add(1)
		log.Info().Msg("Monobank starting serving")
		mb.Serve()
		wg.Done()
	}()
	go func() {

		for v := range mb.TransactionChan {
			log.Debug().Msg("main Creating transaction")
			ffi.FireflyiiiTransactionChan <- v
			log.Debug().Msg("main Created transaction")
		}

	}()
	wg.Wait()
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
