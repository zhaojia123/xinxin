package webassets

import (
	"embed"
	"html/template"
	"io/fs"
	"strconv"
)

// FS contains all server-rendered pages and public assets in the executable.
//
//go:embed templates/*.html static/*
var FS embed.FS

func Templates(filingNumbers ...string) (*template.Template, error) {
	filingNumber := ""
	if len(filingNumbers) > 0 {
		filingNumber = filingNumbers[0]
	}
	return template.New("templates").Funcs(template.FuncMap{
		"filingNumber": func() string { return filingNumber },
		"add":          func(a, b int) int { return a + b },
		"sub":          func(a, b int) int { return a - b },
		"payBasisLabel": func(value string) string {
			switch value {
			case "hourly":
				return "按小时"
			case "daily":
				return "按天"
			case "half_daily":
				return "按半天"
			default:
				return "按月"
			}
		},
		"menuPageURL": func(page int, q, priceCategory, itemCategory, minPrice, maxPrice, sort string) string {
			value := "?page=" + strconv.Itoa(page)
			if q != "" {
				value += "&q=" + template.URLQueryEscaper(q)
			}
			if priceCategory != "" {
				value += "&price_category=" + template.URLQueryEscaper(priceCategory)
			}
			if itemCategory != "" {
				value += "&item_category=" + template.URLQueryEscaper(itemCategory)
			}
			if minPrice != "" {
				value += "&min_price=" + template.URLQueryEscaper(minPrice)
			}
			if maxPrice != "" {
				value += "&max_price=" + template.URLQueryEscaper(maxPrice)
			}
			if sort != "" {
				value += "&sort=" + template.URLQueryEscaper(sort)
			}
			return value
		},
	}).ParseFS(FS, "templates/*.html")
}

func Static() (fs.FS, error) {
	return fs.Sub(FS, "static")
}
