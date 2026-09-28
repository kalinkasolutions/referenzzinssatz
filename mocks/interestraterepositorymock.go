package mocks

import (
	"github.com/google/uuid"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
)

type InterestRateRepositoryMock struct {
	inserted map[string]interestraterepo.InterestRate
}

func NewInterestRateRepositoryMock() *InterestRateRepositoryMock {
	return &InterestRateRepositoryMock{
		inserted: make(map[string]interestraterepo.InterestRate),
	}
}

func (r *InterestRateRepositoryMock) Insert(rir interestraterepo.InterestRate) string {
	id := uuid.New().String()
	rir.Id = id
	r.inserted[id] = rir
	return id
}

func (r *InterestRateRepositoryMock) GetAll() []interestraterepo.InterestRate {
	values := make([]interestraterepo.InterestRate, 0, len(r.inserted))
	for _, v := range r.inserted {
		values = append(values, v)
	}
	return values
}

func (r *InterestRateRepositoryMock) Delete(id string) {
	delete(r.inserted, id)
}

func (r *InterestRateRepositoryMock) ReferenceInterestRateExists(rir interestraterepo.InterestRate) bool {
	for _, v := range r.inserted {
		if v.ReferenceInterestRate == rir.ReferenceInterestRate && v.ValidFrom.Equal(rir.ValidFrom) && v.UnderlyingAvgInterestRate == rir.UnderlyingAvgInterestRate && v.ReferenceDateOfSurvey.Equal(rir.ReferenceDateOfSurvey) {
			return true
		}
	}
	return false
}

func (r *InterestRateRepositoryMock) GetNewest() (interestraterepo.InterestRate, bool) {
	var newest interestraterepo.InterestRate
	found := false
	for _, v := range r.inserted {
		if !found || v.ValidFrom.After(newest.ValidFrom) {
			newest = v
			found = true
		}
	}
	return newest, found
}
