package main

import (
	"errors"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
)

// parserStub stores the given rates on every run, like the real parser does.
type parserStub struct {
	repo  *mocks.InterestRateRepositoryMock
	rates []interestraterepo.InterestRate
	err   error
}

func (p parserStub) ExtractInterestRate() ([]interestraterepo.InterestRate, error) {
	if p.err != nil {
		return nil, p.err
	}
	var inserted []interestraterepo.InterestRate
	for _, rate := range p.rates {
		if !p.repo.ReferenceInterestRateExists(rate) {
			rate.Id = p.repo.Insert(rate)
			inserted = append(inserted, rate)
		}
	}
	return inserted, nil
}

func rate(value float64, validFrom string) interestraterepo.InterestRate {
	date, _ := time.Parse("2006-01-02", validFrom)
	return interestraterepo.InterestRate{ReferenceInterestRate: value, ValidFrom: date}
}

func TestNotifiesWhenNewestRateChanges(t *testing.T) {
	repo := mocks.NewInterestRateRepositoryMock()
	repo.Insert(rate(1.5, "2025-06-03"))
	subscribers := mocks.NewSubscriberRepositoryMock(subscriberrepo.Subscriber{Id: "1", Email: "a@example.ch", EmailValidated: true})
	mail := &mocks.SendMailMock{}

	notifyOnNewInterestRate(mocks.NewLoggerMock(), parserStub{repo: repo, rates: []interestraterepo.InterestRate{rate(1.25, "2025-09-02")}}, repo, subscribers, mail)

	assert.Equal(t, 1, len(mail.Updates))
	assert.Equal(t, 1.25, mail.Updates[0].Newest.ReferenceInterestRate)
	assert.Equal(t, 1.5, mail.Updates[0].Previous.ReferenceInterestRate)
	assert.Equal(t, 1, len(mail.Updates[0].Subscribers))
}

func TestDoesNotNotifyForOlderAdditions(t *testing.T) {
	repo := mocks.NewInterestRateRepositoryMock()
	repo.Insert(rate(1.25, "2025-09-02"))
	mail := &mocks.SendMailMock{}

	notifyOnNewInterestRate(mocks.NewLoggerMock(), parserStub{repo: repo, rates: []interestraterepo.InterestRate{rate(1.75, "2023-12-02")}}, repo, mocks.NewSubscriberRepositoryMock(), mail)

	assert.Equal(t, 0, len(mail.Updates))
}

func TestDoesNotNotifyOnFirstScrape(t *testing.T) {
	repo := mocks.NewInterestRateRepositoryMock()
	mail := &mocks.SendMailMock{}

	notifyOnNewInterestRate(mocks.NewLoggerMock(), parserStub{repo: repo, rates: []interestraterepo.InterestRate{rate(1.5, "2025-06-03"), rate(1.25, "2025-09-02")}}, repo, mocks.NewSubscriberRepositoryMock(), mail)

	assert.Equal(t, 2, len(repo.GetAll()))
	assert.Equal(t, 0, len(mail.Updates))
}

func TestDoesNotNotifyWhenScrapeFails(t *testing.T) {
	repo := mocks.NewInterestRateRepositoryMock()
	repo.Insert(rate(1.25, "2025-09-02"))
	mail := &mocks.SendMailMock{}

	notifyOnNewInterestRate(mocks.NewLoggerMock(), parserStub{repo: repo, err: errors.New("timeout")}, repo, mocks.NewSubscriberRepositoryMock(), mail)

	assert.Equal(t, 0, len(mail.Updates))
}
