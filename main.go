package main

import (
	"flag"
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/api"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/datalayer"
	"github.com/kalinkasolutions/referenzzinssatz/interestrateparser"
	"github.com/kalinkasolutions/referenzzinssatz/logger"
	"github.com/kalinkasolutions/referenzzinssatz/loggersink/consolelogsink"
	"github.com/kalinkasolutions/referenzzinssatz/loggersink/dblogsink"
	"github.com/kalinkasolutions/referenzzinssatz/recaptcha"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
	"github.com/kalinkasolutions/referenzzinssatz/sendmail"
)

const (
	checkInterval = 24 * time.Hour
	logRetention  = 90 * 24 * time.Hour
)

func main() {
	logger := logger.NewLogger(consolelogsink.NewConsoleSink())
	logger.Info("starting referenzzinssatz")

	var configPath string
	flag.StringVar(&configPath, "configPath", "/app/conf.json", "path to the config")
	flag.Parse()

	config := config.LoadConfig(configPath, logger)
	db := datalayer.NewDb(logger, config)

	logRepo := logrepo.NewLogRepository(db)
	logger.AddSink(dblogsink.NewDbSink(logRepo))

	subscriberRepo := subscriberrepo.NewSubscriberRepository(logger, db)
	interestRateRepo := interestraterepo.NewInterestRepository(logger, db)
	interestRateParser := interestrateparser.NewInterestRateParser(logger, config, interestRateRepo)
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
	logger logger.ILogger,
	parser interestrateparser.IInterestRateParser,
	interestRateRepo interestraterepo.IInterestRepository,
	subscriberRepo subscriberrepo.ISubscriberRepository,
	sendMail sendmail.ISendMail,
) {
	previous, hadPrevious := interestRateRepo.GetNewest()

	inserted, err := parser.ExtractInterestRate()
	if err != nil {
		logger.Error("Failed to extract reference interest rates: %v", err)
		return
	}
	logger.Info("Checked reference interest rate, %d new entries", len(inserted))

	current, _ := interestRateRepo.GetNewest()
	if !hadPrevious || current.Id == previous.Id {
		return
	}
	sendMail.SendReferenzZinssatzUpdate(current, previous, subscriberRepo.GetValidatedSubscribers())
}
