// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/twigex/twigex/model"
)

const maxConditionValueLength = 255

var conditionOperators = []string{"equals", "not_equals", "contains", "is_empty", "is_not_empty"}

func conditionNeedsValue(operator string) bool {
	return operator != "is_empty" && operator != "is_not_empty"
}

// validateFormConditions also clears the value of a rule that does not use
// one, so a stale value is not stored with it.
func validateFormConditions(fields []model.WorkspaceFormField) *model.AppError {
	invalid := model.NewAppError("workspace.form_invalid_condition", http.StatusBadRequest)

	rules := make(map[string]*model.VisibilityRule, len(fields))
	for _, f := range fields {
		rules[f.HeaderName] = nil
	}

	for _, f := range fields {
		rule := f.VisibilityRule
		if rule == nil {
			continue
		}

		if _, ok := rules[rule.TriggerField]; !ok || rule.TriggerField == f.HeaderName {
			return invalid
		}

		if !slices.Contains(conditionOperators, rule.Operator) || utf8.RuneCountInString(rule.Value) > maxConditionValueLength {
			return invalid
		}

		if !conditionNeedsValue(rule.Operator) {
			rule.Value = ""
		} else if rule.Value == "" {
			return invalid
		}

		rules[f.HeaderName] = rule
	}

	// Each field watches at most one other, so a chain longer than the form
	// has fields must loop.
	for _, f := range fields {
		name := f.HeaderName
		for range len(fields) {
			if rules[name] == nil {
				break
			}

			name = rules[name].TriggerField
		}

		if rules[name] != nil {
			return invalid
		}
	}

	return nil
}

// formFieldShown tells whether a field is on the form given the submitted
// values. A field that is not shown has no value, so a rule watching it sees
// an empty one.
func formFieldShown(name string, fields map[string]model.WorkspaceFormField, values map[string]any) bool {
	rule := fields[name].VisibilityRule
	if rule == nil {
		return true
	}

	trigger, ok := fields[rule.TriggerField]

	var value any
	if ok && formFieldShown(rule.TriggerField, fields, values) {
		value = values[rule.TriggerField]
	}

	return conditionHolds(rule, conditionValues(trigger, value))
}

// conditionValues reads a submitted value as setFormValue does: a link as
// the ids of its rows, a checkbox as checked or not.
func conditionValues(field model.WorkspaceFormField, value any) []string {
	var text string
	switch v := value.(type) {
	case string:
		text = v
	case bool:
		text = strconv.FormatBool(v)
	case float64:
		text = strconv.FormatFloat(v, 'f', -1, 64)
	}

	switch {
	case field.LinkedID != "" && !field.SingleSelect:
		var ids []string
		for _, id := range strings.Split(text, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}

		return ids

	case field.LinkedID == "" && (field.FieldType == "bool" || field.FieldType == "checkbox"):
		return []string{strconv.FormatBool(text != "" && text != "false" && text != "0")}

	case text == "":
		return nil
	}

	return []string{text}
}

func conditionHolds(rule *model.VisibilityRule, values []string) bool {
	switch rule.Operator {
	case "is_empty":
		return len(values) == 0
	case "is_not_empty":
		return len(values) > 0
	case "equals":
		return slices.Contains(values, rule.Value)
	case "not_equals":
		return !slices.Contains(values, rule.Value)
	case "contains":
		return slices.ContainsFunc(values, func(v string) bool { return strings.Contains(v, rule.Value) })
	}

	return false
}
