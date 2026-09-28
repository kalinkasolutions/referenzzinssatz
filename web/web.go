// Package web embeds the page and mail templates and the static assets,
// so the binary runs without the source tree next to it.
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"math"
	"strconv"
	"strings"
	"time"
)

//go:embed templates static
var files embed.FS

var germanMonths = [...]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}

var templateFuncs = template.FuncMap{
	"percent":          Percent,
	"percentagePoints": PercentagePoints,
	"dateLong":         DateLong,
	"dateShort":        DateShort,
}

func PageTemplates() (*template.Template, error) {
	return template.New("pages").Funcs(templateFuncs).ParseFS(files, "templates/*.html")
}

func MailTemplates() (*template.Template, error) {
	return template.New("mails").Funcs(templateFuncs).ParseFS(files, "templates/mail/*.html")
}

func Static() fs.FS {
	static, err := fs.Sub(files, "static")
	if err != nil {
		panic(err) // the directory is embedded at compile time, so this cannot happen
	}
	return static
}

// Percent formats a rate the way the BWO publishes it, e.g. 1.25 as "1,25 %".
func Percent(rate float64) string {
	return swissNumber(rate) + " %"
}

// PercentagePoints formats a change with its sign, e.g. -0.25 as "−0,25".
func PercentagePoints(change float64) string {
	rounded := math.Round(change*100) / 100
	switch {
	case rounded > 0:
		return "+" + swissNumber(rounded)
	case rounded < 0:
		return "−" + swissNumber(-rounded)
	default:
		return "±0"
	}
}

func DateLong(date time.Time) string {
	return strconv.Itoa(date.Day()) + ". " + germanMonths[date.Month()-1] + " " + strconv.Itoa(date.Year())
}

func DateShort(date time.Time) string {
	return date.Format("02.01.2006")
}

func swissNumber(value float64) string {
	return strings.Replace(strconv.FormatFloat(value, 'f', -1, 64), ".", ",", 1)
}
