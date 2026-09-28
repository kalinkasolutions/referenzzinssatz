package interestraterepo

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/kalinkasolutions/referenzzinssatz/logger"
)

const isoDate = "2006-01-02"

type InterestRate struct {
	Id                        string
	CreatedAt                 string
	ReferenceInterestRate     float64
	ValidFrom                 time.Time
	UnderlyingAvgInterestRate float64
	ReferenceDateOfSurvey     time.Time
}

type IInterestRepository interface {
	Insert(rir InterestRate) string
	ReferenceInterestRateExists(rir InterestRate) bool
	GetAll() []InterestRate
	Delete(id string)
	GetNewest() (InterestRate, bool)
}

type InterestRateRepository struct {
	logger logger.ILogger
	db     *sql.DB
}

const selectColumns = `SELECT Id, CreatedAt, ReferenceInterestRate, ValidFrom, UnderlyingAvgInterestRate, ReferenceDateOfSurvey FROM InterestRates`

func NewInterestRepository(logger logger.ILogger, db *sql.DB) *InterestRateRepository {
	return &InterestRateRepository{
		logger: logger,
		db:     db,
	}
}

func (r *InterestRateRepository) Insert(rir InterestRate) string {
	id := uuid.New().String()
	_, err := r.db.Exec(`
	INSERT INTO InterestRates
		(Id, CreatedAt, ReferenceInterestRate, ValidFrom, UnderlyingAvgInterestRate, ReferenceDateOfSurvey)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, time.Now().Format(time.RFC3339), rir.ReferenceInterestRate, rir.ValidFrom.Format(isoDate), rir.UnderlyingAvgInterestRate, rir.ReferenceDateOfSurvey.Format(isoDate))
	if err != nil {
		r.logger.Error("Failed to insert reference interest rate: %v", err)
	}
	return id
}

func (r *InterestRateRepository) ReferenceInterestRateExists(rir InterestRate) bool {
	row := r.db.QueryRow(
		`SELECT EXISTS (
				SELECT 1 FROM InterestRates WHERE
				ReferenceInterestRate = ? AND
				ValidFrom = ? AND
				UnderlyingAvgInterestRate = ? AND
				ReferenceDateOfSurvey = ?
		)`, rir.ReferenceInterestRate, rir.ValidFrom.Format(isoDate), rir.UnderlyingAvgInterestRate, rir.ReferenceDateOfSurvey.Format(isoDate))
	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		r.logger.Error("Failed to check if reference interest rate exists: %v", err)
	}
	return exists
}

func (r *InterestRateRepository) GetAll() []InterestRate {
	var rirs []InterestRate

	rows, err := r.db.Query(selectColumns)
	if err != nil {
		r.logger.Error("Failed to get reference interest rates: %v", err)
		return rirs
	}
	defer rows.Close()

	for rows.Next() {
		rir, err := scanInterestRate(rows)
		if err != nil {
			r.logger.Error("Failed to read reference interest rate row: %v", err)
			continue
		}
		rirs = append(rirs, rir)
	}
	return rirs
}

func (r *InterestRateRepository) Delete(id string) {
	_, err := r.db.Exec(`DELETE FROM InterestRates WHERE Id = ?`, id)
	if err != nil {
		r.logger.Error("Failed to delete reference interest rate: %v, id: %s", err, id)
	}
}

// GetNewest returns the rate with the latest ValidFrom; a later correction of the same date wins.
func (r *InterestRateRepository) GetNewest() (InterestRate, bool) {
	row := r.db.QueryRow(selectColumns + ` ORDER BY ValidFrom DESC, CreatedAt DESC LIMIT 1`)
	rir, err := scanInterestRate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return InterestRate{}, false
	}
	if err != nil {
		r.logger.Error("Failed to get newest reference interest rate: %v", err)
		return InterestRate{}, false
	}
	return rir, true
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInterestRate(row rowScanner) (InterestRate, error) {
	var rir InterestRate
	var validFrom, referenceDateOfSurvey string
	err := row.Scan(&rir.Id, &rir.CreatedAt, &rir.ReferenceInterestRate, &validFrom, &rir.UnderlyingAvgInterestRate, &referenceDateOfSurvey)
	if err != nil {
		return InterestRate{}, err
	}

	if rir.ValidFrom, err = time.Parse(isoDate, validFrom); err != nil {
		return InterestRate{}, err
	}
	if rir.ReferenceDateOfSurvey, err = time.Parse(isoDate, referenceDateOfSurvey); err != nil {
		return InterestRate{}, err
	}
	return rir, nil
}
