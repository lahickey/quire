package model

import "strings"

type Tags map[string][]string

func tagKey(tag string) string { return strings.ToLower(strings.TrimSpace(tag)) }

func (t Tags) kinds() map[string]string {
	out := map[string]string{}
	for kind, tags := range t {
		for _, tag := range tags {
			out[tagKey(tag)] = kind
		}
	}
	return out
}
