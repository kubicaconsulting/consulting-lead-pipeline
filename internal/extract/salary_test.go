package extract

import "testing"

func TestSalary(t *testing.T) {
	cases := []struct {
		in     string
		lo, hi float64
		period string
	}{
		{"Salary between 45.000€ and 55.000€ gross/year", 45000, 55000, "year"},
		{"Salary: up to 60k", 0, 60000, "year"},
		{"Compensation 140,000 185,000 USD", 140000, 185000, "year"},
		{"Salario 45.000-55.000€ brutos/año", 45000, 55000, "year"},
		{"Sueldo 3.500 €/mes", 3500, 3500, "month"},
		{"competitive salary", 0, 0, "unknown"},
		{"Founded since 2019, we have 2020 customers", 0, 0, "unknown"},
		{"144.340 € al año", 144340, 144340, "year"},
		{"Rango salarial 50k - 62k", 50000, 62000, "year"},
	}
	for _, c := range cases {
		lo, hi, p := Salary(c.in)
		if lo != c.lo || hi != c.hi || p != c.period {
			t.Errorf("Salary(%q) = %v, %v, %q; want %v, %v, %q", c.in, lo, hi, p, c.lo, c.hi, c.period)
		}
	}
}
