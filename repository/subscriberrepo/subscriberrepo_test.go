package subscriberrepo_test

import (
	"log/slog"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/datalayer"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
)

func TestCrudSubscriber(t *testing.T) {
	var mail = "hello@hello.ch"
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	repo := subscriberrepo.NewSubscriberRepository(slog.New(slog.DiscardHandler), db)

	subscriber, err := repo.InsertSubscriber(mail)

	assert.NotEqual(t, "", subscriber.Id)
	assert.Equal(t, nil, err)

	subscribers := repo.GetValidatedSubscribers()
	assert.Equal(t, 0, len(subscribers))

	repo.ValidateEmail(subscriber.ValidationCode)

	subscribers = repo.GetValidatedSubscribers()
	assert.Equal(t, 1, len(subscribers))

	assert.Equal(t, subscriber.Id, subscribers[0].Id)
	assert.NotEqual(t, "", subscribers[0].CreatedAt)
	assert.Equal(t, mail, subscribers[0].Email)

	repo.DeleteSubscriber(subscriber.Id)
	subscribers = repo.GetValidatedSubscribers()

	assert.Equal(t, 0, len(subscribers))
}

func TestDuplicateInsertion(t *testing.T) {
	var mail = "hello@hello.ch"
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	repo := subscriberrepo.NewSubscriberRepository(slog.New(slog.DiscardHandler), db)

	subscriber, _ := repo.InsertSubscriber(mail)
	subscriber, err := repo.InsertSubscriber(mail)

	assert.Equal(t, "", subscriber.Id)
	assert.NotEqual(t, nil, err)
}

func TestValidateEmail(t *testing.T) {
	var mail = "hello@hello.ch"
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	repo := subscriberrepo.NewSubscriberRepository(slog.New(slog.DiscardHandler), db)

	subscriber, _ := repo.InsertSubscriber(mail)

	assert.Equal(t, false, subscriber.EmailValidated)

	validationSuccess := repo.ValidateEmail(subscriber.ValidationCode)

	assert.Equal(t, true, validationSuccess)

	subscriber = repo.GetSubscriber(subscriber.Id)

	assert.Equal(t, true, subscriber.EmailValidated)
}

func TestUnsubscribe(t *testing.T) {
	var mail = "hello@hello.ch"
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})

	defer db.Close()
	repo := subscriberrepo.NewSubscriberRepository(slog.New(slog.DiscardHandler), db)

	subscriber, _ := repo.InsertSubscriber(mail)

	assert.NotEqual(t, "", subscriber.UnsubscribeCode)

	unsubscribeSuccess := repo.Unsubscribe(subscriber.UnsubscribeCode)

	assert.Equal(t, true, unsubscribeSuccess)

	subscribers := repo.GetValidatedSubscribers()

	assert.Equal(t, 0, len(subscribers))
}

func TestGetSubscriberByEmail(t *testing.T) {
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	repo := subscriberrepo.NewSubscriberRepository(slog.New(slog.DiscardHandler), db)

	_, found := repo.GetSubscriberByEmail("hello@hello.ch")
	assert.Equal(t, false, found)

	inserted, _ := repo.InsertSubscriber("hello@hello.ch")
	subscriber, found := repo.GetSubscriberByEmail("hello@hello.ch")

	assert.Equal(t, true, found)
	assert.Equal(t, inserted.Id, subscriber.Id)
}

func TestInvalidCodesAreRejected(t *testing.T) {
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	repo := subscriberrepo.NewSubscriberRepository(slog.New(slog.DiscardHandler), db)

	repo.InsertSubscriber("hello@hello.ch")

	assert.Equal(t, false, repo.ValidateEmail(""))
	assert.Equal(t, false, repo.ValidateEmail("unknown"))
	assert.Equal(t, false, repo.Unsubscribe(""))
	assert.Equal(t, false, repo.Unsubscribe("unknown"))
}
