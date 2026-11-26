// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"testing"
	"time"

	"github.com/twigex/twigex/model"
)

// Each want is what the submissions page shows for the same row, read from
// formulas.js in a browser in Riga.
func TestFormCalculationShowsWhatThePageShows(t *testing.T) {
	riga, err := time.LoadLocation("Europe/Riga")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}

	row := map[string]any{
		"net":    "1234.56",
		"price":  "3.50",
		"rate":   "1.21",
		"vat":    "0.21",
		"qty":    float64(3),
		"half":   "2.675",
		"first":  time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
		"last":   "2026-04-03",
		"client": []model.LinkedItem{{ID: "c1", Name: "Baltic Timber"}, {ID: "c2", Name: "Nord"}},
		"note":   "  Hello   there ",
		"empty":  "",
	}

	field := func(name string) model.FormulaArg { return model.FormulaArg{Type: "field", Value: name} }
	constant := func(value string) model.FormulaArg { return model.FormulaArg{Type: "const", Value: value} }

	cases := []struct {
		name string
		spec model.FormulaSpec
		want string
	}{
		{"a product to two decimals", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("net"), field("rate")}}, "1493.82"},
		{"a product as the page rounds it", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("price"), field("rate")}}, "4.23"},
		{"a half rounded up", model.FormulaSpec{Name: "SUM", Args: []model.FormulaArg{field("half")}}, "2.68"},
		{"a whole number without decimals", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("qty"), constant("2")}}, "6"},
		{"an empty answer skipped", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("vat"), field("empty")}}, "0.21"},
		{"days across the clock change", model.FormulaSpec{Name: "DATEDIF", Args: []model.FormulaArg{field("first"), field("last"), constant(`"D"`)}}, "6"},
		{"picked rows by name", model.FormulaSpec{Name: "CONCATENATE", Args: []model.FormulaArg{field("client"), constant(": "), field("qty")}}, "Baltic Timber, Nord: 3"},
		{"text trimmed", model.FormulaSpec{Name: "TRIM", Args: []model.FormulaArg{field("note")}}, "Hello there"},
		{"a day", model.FormulaSpec{Name: "DATE", Args: []model.FormulaArg{constant("2026"), constant("3"), constant("29")}}, "29.03.2026"},
		{"a test", model.FormulaSpec{Name: "IF", Args: []model.FormulaArg{field("empty"), constant("yes"), constant("no")}}, "no"},
		{"an aggregate over no fields", model.FormulaSpec{Name: "SUM", Args: []model.FormulaArg{constant("5")}}, ""},
		{"a formula the page does not know", model.FormulaSpec{Name: "FOO", Args: []model.FormulaArg{field("qty")}}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formCalculation(&tc.spec, calculationRow(row), riga, time.Now()); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
