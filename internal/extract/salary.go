package extract

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	amountRe  = regexp.MustCompile(`(\d{1,3}(?:[.,]\d{3})+|\d+)(?:[.,]\d{1,2})?\s?([kK])?\b`)
	monthRe   = regexp.MustCompile(`(?i)\bmes\b|/\s?mes|month|mensual|monthly|/\s?mo\b`)
	upToRe    = regexp.MustCompile(`(?i)up to|hasta|max(imum)?\.?\s|m[aá]ximo`)
	keywordRe = regexp.MustCompile(`(?i)salary|salario|rango salarial|banda salarial|retribuci[oó]n|compensation|pay range|sueldo|remuneraci[oó]n|brutos?/a[ñn]o|gross`)
)

// Salary reads an amount or range from a short salary text.
// It returns 0, 0, "unknown" when no plausible amount is found.
func Salary(text string) (lo, hi float64, period string) {
	monthly := monthRe.MatchString(text)
	var amts []float64
	for _, m := range amountRe.FindAllStringSubmatchIndex(text, -1) {
		digits := text[m[2]:m[3]]
		hasK := m[4] >= 0
		hasSep := strings.ContainsAny(digits, ".,")
		v, err := strconv.ParseFloat(strings.NewReplacer(".", "", ",", "").Replace(digits), 64)
		if err != nil {
			continue
		}
		if hasK {
			v *= 1000
		} else if !hasSep && v >= 1900 && v <= 2099 {
			continue // a year, not a salary
		}
		if monthly {
			if v < 500 || v > 20000 {
				continue
			}
		} else if v < 15000 || v > 400000 {
			continue
		}
		amts = append(amts, v)
	}
	if len(amts) == 0 {
		return 0, 0, "unknown"
	}
	period = "year"
	if monthly {
		period = "month"
	}
	if len(amts) == 1 {
		if upToRe.MatchString(text) {
			return 0, amts[0], period
		}
		return amts[0], amts[0], period
	}
	return amts[0], amts[1], period
}

// salaryFromDesc looks for a salary keyword and reads the 120 characters after it.
func salaryFromDesc(desc string) (lo, hi float64, period string) {
	for _, k := range keywordRe.FindAllStringIndex(desc, -1) {
		end := min(len(desc), k[0]+120)
		if lo, hi, p := Salary(desc[k[0]:end]); p != "unknown" {
			return lo, hi, p
		}
	}
	return 0, 0, "unknown"
}
