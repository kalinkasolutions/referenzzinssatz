package mocks

import (
	"errors"

	"github.com/google/uuid"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
)

type SubscriberRepositoryMock struct {
	subscribers map[string]subscriberrepo.Subscriber
}

func NewSubscriberRepositoryMock(subscribers ...subscriberrepo.Subscriber) *SubscriberRepositoryMock {
	mock := &SubscriberRepositoryMock{subscribers: make(map[string]subscriberrepo.Subscriber)}
	for _, subscriber := range subscribers {
		mock.subscribers[subscriber.Id] = subscriber
	}
	return mock
}

func (r *SubscriberRepositoryMock) InsertSubscriber(email string) (subscriberrepo.Subscriber, error) {
	if _, found := r.GetSubscriberByEmail(email); found {
		return subscriberrepo.Subscriber{}, errors.New("subscriber already exists")
	}
	subscriber := subscriberrepo.Subscriber{
		Id:              uuid.New().String(),
		Email:           email,
		ValidationCode:  uuid.New().String(),
		UnsubscribeCode: uuid.New().String(),
	}
	r.subscribers[subscriber.Id] = subscriber
	return subscriber, nil
}

func (r *SubscriberRepositoryMock) GetSubscriberByEmail(email string) (subscriberrepo.Subscriber, bool) {
	for _, subscriber := range r.subscribers {
		if subscriber.Email == email {
			return subscriber, true
		}
	}
	return subscriberrepo.Subscriber{}, false
}

func (r *SubscriberRepositoryMock) ValidateEmail(validationCode string) bool {
	for id, subscriber := range r.subscribers {
		if validationCode != "" && subscriber.ValidationCode == validationCode {
			subscriber.EmailValidated = true
			r.subscribers[id] = subscriber
			return true
		}
	}
	return false
}

func (r *SubscriberRepositoryMock) Unsubscribe(unsubscribeCode string) bool {
	for id, subscriber := range r.subscribers {
		if unsubscribeCode != "" && subscriber.UnsubscribeCode == unsubscribeCode {
			delete(r.subscribers, id)
			return true
		}
	}
	return false
}

func (r *SubscriberRepositoryMock) GetSubscriber(id string) subscriberrepo.Subscriber {
	return r.subscribers[id]
}

func (r *SubscriberRepositoryMock) GetValidatedSubscribers() []subscriberrepo.Subscriber {
	var validated []subscriberrepo.Subscriber
	for _, subscriber := range r.subscribers {
		if subscriber.EmailValidated {
			validated = append(validated, subscriber)
		}
	}
	return validated
}

func (r *SubscriberRepositoryMock) DeleteSubscriber(id string) {
	delete(r.subscribers, id)
}
