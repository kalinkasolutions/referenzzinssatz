package mocks

import (
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
)

type SendMailMock struct {
	Fail            bool
	ValidationMails []subscriberrepo.Subscriber
	Updates         []InterestRateUpdate
}

type InterestRateUpdate struct {
	Newest      interestraterepo.InterestRate
	Previous    interestraterepo.InterestRate
	Subscribers []subscriberrepo.Subscriber
}

func (s *SendMailMock) SendValidationMail(subscriber subscriberrepo.Subscriber) bool {
	s.ValidationMails = append(s.ValidationMails, subscriber)
	return !s.Fail
}

func (s *SendMailMock) SendReferenzZinssatzUpdate(newest interestraterepo.InterestRate, previous interestraterepo.InterestRate, subscribers []subscriberrepo.Subscriber) {
	s.Updates = append(s.Updates, InterestRateUpdate{Newest: newest, Previous: previous, Subscribers: subscribers})
}
