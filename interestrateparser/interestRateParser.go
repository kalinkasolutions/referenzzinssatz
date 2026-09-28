package interestrateparser

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
)

const rateTableRows = "table.table tbody tr"

var (
	percentPattern = regexp.MustCompile(`\d+(,\d+)?`)
	datePattern    = regexp.MustCompile(`\d{1,2}\.\d{1,2}\.\d{4}`)
)

type IInterestRateParser interface {
	ExtractInterestRate() ([]interestraterepo.InterestRate, error)
}

type InterestRateParser struct {
	config           config.Config
	interestRateRepo interestraterepo.IInterestRepository
}

func NewInterestRateParser(config config.Config, interestRateRepo interestraterepo.IInterestRepository) *InterestRateParser {
	return &InterestRateParser{
		config:           config,
		interestRateRepo: interestRateRepo,
	}
}

// ExtractInterestRate scrapes the published rates and stores the ones not seen before, returning those.
func (ip *InterestRateParser) ExtractInterestRate() ([]interestraterepo.InterestRate, error) {
	html, err := ip.fetchRenderedHtml()
	if err != nil {
		return nil, fmt.Errorf("could not fetch %s: %w", ip.config.ReferenzZinssatzUrl, err)
	}

	rirs, err := ParseInterestRates(html)
	if err != nil {
		return nil, err
	}

	var inserted []interestraterepo.InterestRate
	for _, rir := range rirs {
		if !ip.interestRateRepo.ReferenceInterestRateExists(rir) {
			rir.Id = ip.interestRateRepo.Insert(rir)
			inserted = append(inserted, rir)
		}
	}
	return inserted, nil
}

// The page renders its table with JavaScript, so it needs a real browser.
func (ip *InterestRateParser) fetchRenderedHtml() (string, error) {
	ctx, cancelAllocator := chromedp.NewExecAllocator(context.Background(),
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox)
	defer cancelAllocator()

	ctx, cancelCtx := chromedp.NewContext(ctx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(ip.config.ReferenzZinssatzUrl),
		chromedp.WaitReady(rateTableRows, chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)
	return html, err
}

// ParseInterestRates fails on any unreadable row: if the page layout changes,
// mailing a half-parsed table is worse than mailing nothing.
func ParseInterestRates(html string) ([]interestraterepo.InterestRate, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("could not parse html: %w", err)
	}

	var rirs []interestraterepo.InterestRate
	var rowErrors []error
	doc.Find("table.table tbody").First().Find("tr").Each(func(i int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() < 4 {
			return
		}
		rir, err := parseRow(tds)
		if err != nil {
			rowErrors = append(rowErrors, fmt.Errorf("row %d: %w", i+1, err))
			return
		}
		rirs = append(rirs, rir)
	})

	if len(rowErrors) > 0 {
		return nil, errors.Join(rowErrors...)
	}
	if len(rirs) == 0 {
		return nil, errors.New("no interest rates found, the page layout may have changed")
	}
	return rirs, nil
}

func parseRow(tds *goquery.Selection) (interestraterepo.InterestRate, error) {
	referenceInterestRate, err := parsePercent(tds.Eq(0).Text())
	if err != nil {
		return interestraterepo.InterestRate{}, err
	}
	validFrom, err := parseDate(tds.Eq(1).Text())
	if err != nil {
		return interestraterepo.InterestRate{}, err
	}
	underlyingAvgInterestRate, err := parsePercent(tds.Eq(2).Text())
	if err != nil {
		return interestraterepo.InterestRate{}, err
	}
	referenceDateOfSurvey, err := parseDate(tds.Eq(3).Text())
	if err != nil {
		return interestraterepo.InterestRate{}, err
	}

	return interestraterepo.InterestRate{
		ReferenceInterestRate:     referenceInterestRate,
		ValidFrom:                 validFrom,
		UnderlyingAvgInterestRate: underlyingAvgInterestRate,
		ReferenceDateOfSurvey:     referenceDateOfSurvey,
	}, nil
}

// parsePercent reads the Swiss notation used on the page, e.g. "1,25 %".
func parsePercent(text string) (float64, error) {
	number := percentPattern.FindString(text)
	if number == "" {
		return 0, fmt.Errorf("no percentage in %q", text)
	}
	return strconv.ParseFloat(strings.ReplaceAll(number, ",", "."), 64)
}

func parseDate(text string) (time.Time, error) {
	date := datePattern.FindString(text)
	if date == "" {
		return time.Time{}, fmt.Errorf("no date in %q", text)
	}
	return time.Parse("2.1.2006", date)
}
