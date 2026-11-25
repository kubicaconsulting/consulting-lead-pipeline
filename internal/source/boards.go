package source

import (
	"strings"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

// linkedin maps fantastic-jobs/advanced-linkedin-job-search-api items.
func linkedin(it map[string]any) model.Job {
	return model.Job{
		ID:           str(it, "id"),
		Title:        str(it, "title"),
		Company:      str(it, "organization"),
		City:         first(it, "cities_derived"),
		Location:     first(it, "locations_derived"),
		URL:          str(it, "url"),
		Lang:         str(it, "ai_job_language"),
		Desc:         clean(str(it, "description_text")),
		Posted:       date(str(it, "date_posted")),
		SalaryLo:     num(it, "ai_salary_min_value"),
		SalaryHi:     num(it, "ai_salary_max_value"),
		SalaryPeriod: period(str(it, "ai_salary_unit_text")),
		Headcount:    str(it, "org_linkedin_size"),
		Industry:     str(it, "org_linkedin_industry"),
		WorkMode:     str(it, "ai_work_arrangement"),
	}
}

// tecnoempleo maps blackfalcondata/tecnoempleo-scraper items.
func tecnoempleo(it map[string]any) model.Job {
	city := str(it, "province")
	if city == "" {
		city = str(it, "location")
	}
	return model.Job{
		Title:        str(it, "title"),
		Company:      str(it, "company"),
		City:         city,
		Location:     str(it, "location"),
		URL:          str(it, "url"),
		Desc:         clean(str(it, "descriptionText")),
		Posted:       date(str(it, "publishDateISO")),
		SalaryText:   str(it, "salaryText"),
		SalaryLo:     num(it, "salaryMin"),
		SalaryHi:     num(it, "salaryMax"),
		SalaryPeriod: period(str(it, "salaryPeriod")),
		WorkMode:     str(it, "workArrangement"),
	}
}

// indeedApify maps borderline/indeed-scraper items (flattened "a.b" keys).
func indeedApify(it map[string]any) model.Job {
	mode := ""
	if b, ok := it["isRemote"].(bool); ok && b {
		mode = "remote"
	}
	desc := clean(str(it, "descriptionText"))
	if attrs, ok := it["attributes"].([]any); ok && len(attrs) > 0 {
		parts := make([]string, 0, len(attrs))
		for _, a := range attrs {
			if s, ok := a.(string); ok {
				parts = append(parts, s)
			}
		}
		desc += "\n" + strings.Join(parts, ", ")
	}
	return model.Job{
		Title:        str(it, "title"),
		Company:      str(it, "companyName"),
		City:         str(it, "location.city"),
		URL:          str(it, "jobUrl"),
		Lang:         str(it, "language"),
		Desc:         desc,
		Posted:       date(str(it, "datePublished")),
		SalaryText:   str(it, "salary.salaryText"),
		SalaryPeriod: period(str(it, "salary.salaryType")),
		Headcount:    str(it, "companyNumEmployees"),
		Industry:     str(it, "companySector"),
		WorkMode:     mode,
	}
}
