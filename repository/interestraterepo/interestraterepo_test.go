package interestraterepo_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/datalayer"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
)

func newTestRepository(t *testing.T) (*interestraterepo.InterestRateRepository, *sql.DB) {
	db := datalayer.NewDb(mocks.NewLoggerMock(), config.Config{
		DatabaseName: "file::memory:?cache=shared",
	})
	t.Cleanup(func() { db.Close() })
	return interestraterepo.NewInterestRepository(mocks.NewLoggerMock(), db), db
}

func date(value string) time.Time {
	parsed, _ := time.Parse("2006-01-02", value)
	return parsed
}

func TestCrudInterestRate(t *testing.T) {
	repo, _ := newTestRepository(t)
	rir := interestraterepo.InterestRate{
		ReferenceInterestRate:     1.25,
		ValidFrom:                 date("2025-09-02"),
		UnderlyingAvgInterestRate: 1.37,
		ReferenceDateOfSurvey:     date("2025-06-30"),
	}

	id := repo.Insert(rir)
	rirs := repo.GetAll()

	assert.Equal(t, 1, len(rirs))
	assert.Equal(t, id, rirs[0].Id)
	assert.NotEqual(t, "", rirs[0].CreatedAt)
	assert.Equal(t, rir.ReferenceInterestRate, rirs[0].ReferenceInterestRate)
	assert.Equal(t, rir.ValidFrom, rirs[0].ValidFrom)
	assert.Equal(t, rir.UnderlyingAvgInterestRate, rirs[0].UnderlyingAvgInterestRate)
	assert.Equal(t, rir.ReferenceDateOfSurvey, rirs[0].ReferenceDateOfSurvey)
	assert.Equal(t, true, repo.ReferenceInterestRateExists(rir))

	repo.Delete(id)

	assert.Equal(t, 0, len(repo.GetAll()))
	assert.Equal(t, false, repo.ReferenceInterestRateExists(rir))
}

func TestGetNewestOrdersByValidFrom(t *testing.T) {
	repo, _ := newTestRepository(t)

	_, found := repo.GetNewest()
	assert.Equal(t, false, found)

	repo.Insert(interestraterepo.InterestRate{ReferenceInterestRate: 1.5, ValidFrom: date("2025-06-03"), ReferenceDateOfSurvey: date("2025-03-31")})
	repo.Insert(interestraterepo.InterestRate{ReferenceInterestRate: 1.25, ValidFrom: date("2025-09-02"), ReferenceDateOfSurvey: date("2025-06-30")})
	repo.Insert(interestraterepo.InterestRate{ReferenceInterestRate: 1.75, ValidFrom: date("2024-12-02"), ReferenceDateOfSurvey: date("2024-09-30")})

	newest, found := repo.GetNewest()

	assert.Equal(t, true, found)
	assert.Equal(t, 1.25, newest.ReferenceInterestRate)
}

func TestMigrationNormalizesScrapedDates(t *testing.T) {
	_, db := newTestRepository(t)
	_, err := db.Exec(`INSERT INTO InterestRates VALUES ('legacy', '2024-01-01T00:00:00Z', 1.75, ' 02.12.2023 ', 1.69, '30.09.2023')`)
	assert.Equal(t, nil, err)

	// Re-run the date migration the way an old database would receive it.
	_, err = db.Exec(`PRAGMA user_version = 1`)
	assert.Equal(t, nil, err)
	datalayer.NewDb(mocks.NewLoggerMock(), config.Config{DatabaseName: "file::memory:?cache=shared"}).Close()

	var validFrom, surveyDate string
	err = db.QueryRow(`SELECT ValidFrom, ReferenceDateOfSurvey FROM InterestRates WHERE Id = 'legacy'`).Scan(&validFrom, &surveyDate)

	assert.Equal(t, nil, err)
	assert.Equal(t, "2023-12-02", validFrom)
	assert.Equal(t, "2023-09-30", surveyDate)
}
