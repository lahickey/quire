package model

import (
	"cmp"
	"fmt"
	"time"

	"go.yaml.in/yaml/v3"
)

var datesDayFirst = Dates{"2 January 2006", "2 Jan 2006", "Jan 2006"}

type Date struct{ time.Time }

type Dates struct {
	Long           string
	Short          string
	ShortMonthYear string
}

func (d Dates) orDefault() Dates {
	return Dates{
		Long:           cmp.Or(d.Long, datesDayFirst.Long),
		Short:          cmp.Or(d.Short, datesDayFirst.Short),
		ShortMonthYear: cmp.Or(d.ShortMonthYear, datesDayFirst.ShortMonthYear),
	}
}

func ParseDate(iso string) (Date, error) {
	t, err := time.Parse(time.DateOnly, iso)
	if err != nil {
		return Date{}, fmt.Errorf("date %q: want ISO YYYY-MM-DD", iso)
	}
	return Date{t}, nil
}

func (d *Date) UnmarshalYAML(n *yaml.Node) error {
	var s string
	if err := n.Decode(&s); err != nil {
		return fmt.Errorf("date %q: want ISO YYYY-MM-DD", n.Value)
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d Date) ISO() string {
	if d.IsZero() {
		return ""
	}
	return d.Format(time.DateOnly)
}

func (d Date) Formatted(layout string) string {
	if d.IsZero() {
		return ""
	}
	return d.Format(layout)
}
