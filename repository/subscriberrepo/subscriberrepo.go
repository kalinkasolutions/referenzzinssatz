package subscriberrepo

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"log/slog"
)

type Subscriber struct {
	Id              string
	CreatedAt       string
	Email           string
	EmailValidated  bool
	ValidationCode  string
	UnsubscribeCode string
}

type ISubscriberRepository interface {
	InsertSubscriber(email string) (Subscriber, error)
	GetSubscriberByEmail(email string) (Subscriber, bool)
	ValidateEmail(validationCode string) bool
	Unsubscribe(unsubscribeCode string) bool
	GetSubscriber(id string) Subscriber
	GetValidatedSubscribers() []Subscriber
	DeleteSubscriber(id string)
}

type SubscriberRepository struct {
	logger *slog.Logger
	db     *sql.DB
}

const selectColumns = `SELECT Id, CreatedAt, Email, EmailValidated, ValidationCode, UnsubscribeCode FROM Subscribers`

func NewSubscriberRepository(logger *slog.Logger, db *sql.DB) *SubscriberRepository {
	return &SubscriberRepository{
		logger: logger,
		db:     db,
	}
}

// InsertSubscriber expects a normalized address; the unique index rejects duplicates.
func (r *SubscriberRepository) InsertSubscriber(email string) (Subscriber, error) {
	subscriber := Subscriber{
		Id:              uuid.New().String(),
		CreatedAt:       time.Now().Format(time.RFC3339),
		Email:           email,
		ValidationCode:  uuid.New().String(),
		UnsubscribeCode: uuid.New().String(),
	}
	_, err := r.db.Exec(`INSERT INTO Subscribers
							(Id, CreatedAt, Email, EmailValidated, ValidationCode, UnsubscribeCode)
							VALUES (?, ?, ?, ?, ?, ?)`,
		subscriber.Id, subscriber.CreatedAt, subscriber.Email, subscriber.EmailValidated, subscriber.ValidationCode, subscriber.UnsubscribeCode)
	if err != nil {
		return Subscriber{}, err
	}
	return subscriber, nil
}

func (r *SubscriberRepository) GetSubscriberByEmail(email string) (Subscriber, bool) {
	subscriber, err := scanSubscriber(r.db.QueryRow(selectColumns+" WHERE Email = ?", email))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			r.logger.Error("Failed to look up subscriber by email", "error", err)
		}
		return Subscriber{}, false
	}
	return subscriber, true
}

func (r *SubscriberRepository) ValidateEmail(validationCode string) bool {
	result, err := r.db.Exec(`UPDATE Subscribers SET EmailValidated = ? WHERE ValidationCode = ?`, true, validationCode)
	return r.affectedExactlyOneRow(result, err, "validate email address")
}

func (r *SubscriberRepository) Unsubscribe(unsubscribeCode string) bool {
	result, err := r.db.Exec(`DELETE FROM Subscribers WHERE UnsubscribeCode = ?`, unsubscribeCode)
	return r.affectedExactlyOneRow(result, err, "unsubscribe")
}

func (r *SubscriberRepository) affectedExactlyOneRow(result sql.Result, err error, action string) bool {
	if err != nil {
		r.logger.Error("Failed to update subscriber", "action", action, "error", err)
		return false
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get affected rows", "action", action, "error", err)
		return false
	}
	return rowsAffected == 1
}

func (r *SubscriberRepository) GetSubscriber(id string) Subscriber {
	subscriber, err := scanSubscriber(r.db.QueryRow(selectColumns+" WHERE Id = ?", id))
	if err != nil {
		r.logger.Error("Failed to read subscriber row", "error", err)
	}
	return subscriber
}

func (r *SubscriberRepository) GetValidatedSubscribers() []Subscriber {
	var subscribers []Subscriber

	rows, err := r.db.Query(selectColumns+" WHERE EmailValidated = ?", true)
	if err != nil {
		r.logger.Error("Failed to get subscribers", "error", err)
		return subscribers
	}
	defer rows.Close()

	for rows.Next() {
		subscriber, err := scanSubscriber(rows)
		if err != nil {
			r.logger.Error("Failed to read subscriber row", "error", err)
			continue
		}
		subscribers = append(subscribers, subscriber)
	}
	return subscribers
}

func (r *SubscriberRepository) DeleteSubscriber(id string) {
	_, err := r.db.Exec(`DELETE FROM Subscribers WHERE Id = ?`, id)
	if err != nil {
		r.logger.Error("Failed to delete subscriber", "id", id, "error", err)
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSubscriber(row rowScanner) (Subscriber, error) {
	var subscriber Subscriber
	err := row.Scan(&subscriber.Id, &subscriber.CreatedAt, &subscriber.Email, &subscriber.EmailValidated, &subscriber.ValidationCode, &subscriber.UnsubscribeCode)
	return subscriber, err
}
