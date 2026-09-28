package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/api"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/datalayer"
	"github.com/kalinkasolutions/referenzzinssatz/dblog"
	"github.com/kalinkasolutions/referenzzinssatz/interestrateparser"
	"github.com/kalinkasolutions/referenzzinssatz/loki"
	"github.com/kalinkasolutions/referenzzinssatz/recaptcha"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
	"github.com/kalinkasolutions/referenzzinssatz/sendmail"
)

const (
	checkInterval = 24 * time.Hour
	logRetention  = 90 * 24 * time.Hour
	// Info-level lines such as every HTTP request stay out of the database; Loki gets them all.
	storedLogLevel = slog.LevelWarn
	dateFormat     = "2006-01-02"
)

func main() {
	consoleLevel := new(slog.LevelVar)
	console := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: consoleLevel})
	logger := slog.New(console)
	logger.Info("starting referenzzinssatz")

	var configPath string
	flag.StringVar(&configPath, "configPath", "/app/conf.json", "path to the config")
	flag.Parse()

	config := config.LoadConfig(configPath, logger)
	if config.Debug {
		consoleLevel.Set(slog.LevelDebug)
	}

	// Loki joins as soon as the config is known, so it also sees the database setup.
	handlers := []slog.Handler{console}
	if config.Loki.Url != "" {
		lokiWriter := loki.NewWriter(config.Loki)
		handlers = append(handlers, slog.NewJSONHandler(lokiWriter, &slog.HandlerOptions{Level: consoleLevel}))
		go flushOnShutdown(lokiWriter)
		logger = slog.New(slog.NewMultiHandler(handlers...))
		logger.Info("Shipping logs to Loki", "url", loki.PushUrl(config.Loki.Url))
	}

	db := datalayer.NewDb(logger, config)
	logRepo := logrepo.NewLogRepository(db)
	handlers = append(handlers, dblog.NewHandler(logRepo, storedLogLevel))
	logger = slog.New(slog.NewMultiHandler(handlers...))
	slog.SetDefault(logger)

	subscriberRepo := subscriberrepo.NewSubscriberRepository(logger, db)
	interestRateRepo := interestraterepo.NewInterestRepository(logger, db)
	interestRateParser := interestrateparser.NewInterestRateParser(config, interestRateRepo)
	sendMail := sendmail.NewSendMail(logger, config)
	captcha := recaptcha.NewReCaptcha(logger, config)

	go runEvery(checkInterval, func() {
		notifyOnNewInterestRate(logger, interestRateParser, interestRateRepo, subscriberRepo, sendMail)
		logRepo.DeleteOlderThan(time.Now().Add(-logRetention))
	})

	api.NewApi(config, logger, subscriberRepo, interestRateRepo, sendMail, captcha).Load()
}

// flushOnShutdown pushes the last collected lines when the container is stopped.
func flushOnShutdown(lokiWriter *loki.Writer) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	slog.Info("Shutting down")
	lokiWriter.Flush()
	os.Exit(0)
}

// runEvery runs job right away and then after every interval, so a restart never delays a check.
func runEvery(interval time.Duration, job func()) {
	for {
		job()
		time.Sleep(interval)
	}
}

// Subscribers hear about a change only when the newest rate differs from the one before the
// scrape. On the very first scrape there is nothing to compare with, so nobody is mailed.
func notifyOnNewInterestRate(
	logger *slog.Logger,
	parser interestrateparser.IInterestRateParser,
	interestRateRepo interestraterepo.IInterestRepository,
	subscriberRepo subscriberrepo.ISubscriberRepository,
	sendMail sendmail.ISendMail,
) {
	logger.Info("Checking reference interest rate")
	previous, hadPrevious := interestRateRepo.GetNewest()

	inserted, err := parser.ExtractInterestRate()
	if err != nil {
		logger.Error("Failed to extract reference interest rates", "error", err)
		return
	}
	logger.Info("Checked reference interest rate", "newEntries", len(inserted))

	current, _ := interestRateRepo.GetNewest()
	switch {
	case !hadPrevious:
		logger.Info("First scrape, nothing to compare with, no mails sent", "rate", current.ReferenceInterestRate, "validFrom", current.ValidFrom.Format(dateFormat))
	case current.Id == previous.Id:
		logger.Info("Reference interest rate unchanged", "rate", current.ReferenceInterestRate, "validFrom", current.ValidFrom.Format(dateFormat))
	default:
		logger.Info("Reference interest rate changed, notifying subscribers",
			"previousRate", previous.ReferenceInterestRate,
			"rate", current.ReferenceInterestRate,
			"validFrom", current.ValidFrom.Format(dateFormat))
		sendMail.SendReferenzZinssatzUpdate(current, previous, subscriberRepo.GetValidatedSubscribers())
	}
}
