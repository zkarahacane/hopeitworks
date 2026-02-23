package action

import "strings"

// renderTemplate replaces {key} placeholders in tpl with values from vars.
func renderTemplate(tpl string, vars map[string]string) string {
	result := tpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}
