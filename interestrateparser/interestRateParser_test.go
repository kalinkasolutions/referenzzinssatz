package interestrateparser

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
)

func TestParseInterestRates(t *testing.T) {
	rirs, err := ParseInterestRates(readFile(t, "../testresources/test.html"))

	assert.Equal(t, nil, err)
	assert.Equal(t, 69, len(rirs))
	assert.Equal(t, 1.25, rirs[0].ReferenceInterestRate)
	assert.Equal(t, time.Date(2025, 9, 2, 0, 0, 0, 0, time.UTC), rirs[0].ValidFrom)
	assert.Equal(t, 1.37, rirs[0].UnderlyingAvgInterestRate)
	assert.Equal(t, time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC), rirs[0].ReferenceDateOfSurvey)
}

func TestParseInterestRatesRejectsUnreadableRows(t *testing.T) {
	html := `<table class="table"><tbody>
		<tr><td>1,25 %</td><td>02.09.2025</td><td>1,37 %</td><td>30.06.2025</td></tr>
		<tr><td>n/a</td><td>03.06.2025</td><td>1,44 %</td><td>31.03.2025</td></tr>
	</tbody></table>`

	rirs, err := ParseInterestRates(html)

	assert.NotEqual(t, nil, err)
	assert.Equal(t, 0, len(rirs))
}

func TestParseInterestRatesRejectsMissingTable(t *testing.T) {
	_, err := ParseInterestRates("<html><body>Wartungsarbeiten</body></html>")

	assert.NotEqual(t, nil, err)
}

func TestParseDateToleratesWhitespaceAndFootnotes(t *testing.T) {
	date, err := parseDate("\n\t 2.6.2025*")

	assert.Equal(t, nil, err)
	assert.Equal(t, time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC), date)
}

func TestExtractInterestRateStoresOnlyNewRates(t *testing.T) {
	requireBrowser(t)
	interestRateRepoMock := mocks.NewInterestRateRepositoryMock()

	rirs, err := newParserFor(t, "../testresources/test.html", interestRateRepoMock).ExtractInterestRate()
	assert.Equal(t, nil, err)
	assert.Equal(t, 69, len(rirs))

	rirs, err = newParserFor(t, "../testresources/test.html", interestRateRepoMock).ExtractInterestRate()
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(rirs))

	rirs, err = newParserFor(t, "../testresources/test_updated.html", interestRateRepoMock).ExtractInterestRate()
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(rirs))
	assert.Equal(t, 1.0, rirs[0].ReferenceInterestRate)
	assert.Equal(t, 70, len(interestRateRepoMock.GetAll()))
}

func TestExtractInterestRateReportsUnreachablePage(t *testing.T) {
	requireBrowser(t)
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	parser := NewInterestRateParser(mocks.NewLoggerMock(), config.Config{ReferenzZinssatzUrl: server.URL}, mocks.NewInterestRateRepositoryMock())

	_, err := parser.ExtractInterestRate()

	assert.NotEqual(t, nil, err)
}

func newParserFor(t *testing.T, testfile string, repo *mocks.InterestRateRepositoryMock) *InterestRateParser {
	html := readFile(t, testfile)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	t.Cleanup(server.Close)
	return NewInterestRateParser(mocks.NewLoggerMock(), config.Config{ReferenzZinssatzUrl: server.URL}, repo)
}

func readFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load test file %v", err)
	}
	return string(content)
}

func requireBrowser(t *testing.T) {
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "headless-shell"} {
		if _, err := exec.LookPath(name); err == nil {
			return
		}
	}
	t.Skip("no Chrome or Chromium installed")
}
