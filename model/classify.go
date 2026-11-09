package model

import (
	"fmt"
	"slices"
	"strings"
)

type Rule struct {
	category string
	needles  []string
}

type Classifier struct {
	rules []Rule
}

func (c *Classifier) check(values ...string) error {
	seen := make(map[string]bool)
	var unknown []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		if _, ok := c.Classify(v); !ok {
			unknown = append(unknown, v)
			seen[v] = true
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unclassified: %s", strings.Join(unknown, ", "))
	}
	return nil
}

func Exact(category string, needles ...string) Rule {
	return Rule{category: category, needles: needles}
}

func NewClassifier(rules ...Rule) *Classifier {
	out := &Classifier{rules: make([]Rule, 0, len(rules))}
	for _, r := range rules {
		cr := Rule{category: r.category}
		for _, n := range r.needles {
			if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
				cr.needles = append(cr.needles, n)
			}
		}
		if len(cr.needles) > 0 {
			out.rules = append(out.rules, cr)
		}
	}
	return out
}

func (c *Classifier) Classify(s string) (string, bool) {
	h := strings.ToLower(strings.TrimSpace(s))
	if h == "" {
		return "", false
	}
	for _, r := range c.rules {
		if slices.Contains(r.needles, h) {
			return r.category, true
		}
	}
	return "", false
}

func (c *Classifier) Markers(tags []string) []Marker {
	out := make([]Marker, len(tags))
	for i, t := range tags {
		out[i].Text = t
		if c != nil {
			out[i].Kind, _ = c.Classify(t)
		}
	}
	return out
}
