package mocks

type RecaptchaMock struct {
	Human bool
}

func (r *RecaptchaMock) CreateAssessment(token string, recaptchaAction string) bool {
	return r.Human
}
