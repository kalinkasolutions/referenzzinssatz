package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/api"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/datalayer"
	"github.com/kalinkasolutions/referenzzinssatz/dblog"
	"github.com/kalinkasolutions/referenzzinssatz/interestrateparser"
	"github.com/kalinkasolutions/referenzzinssatz/recaptcha"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
	"github.com/kalinkasolutions/referenzzinssatz/sendmail"
)

const (
	checkInterval = 24 * time.Hour
	logRetention  = 90 * 24 * time.Hour
	// Info-level lines such as every HTTP request stay in the console only.
	storedLogLevel = slog.LevelWarn
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
	db := datalayer.NewDb(logger, config)

	logRepo := logrepo.NewLogRepository(db)
	logger = slog.New(slog.NewMultiHandler(console, dblog.NewHandler(logRepo, storedLogLevel)))
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
	previous, hadPrevious := interestRateRepo.GetNewest()

	inserted, err := parser.ExtractInterestRate()
	if err != nil {
		logger.Error("Failed to extract reference interest rates", "error", err)
		return
	}
	logger.Info("Checked reference interest rate", "newEntries", len(inserted))

	current, _ := interestRateRepo.GetNewest()
	if !hadPrevious || current.Id == previous.Id {
		return
	}
	sendMail.SendReferenzZinssatzUpdate(current, previous, subscriberRepo.GetValidatedSubscribers())
}
