// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"math"
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
		"none":   nil,
		"small":  "0.05",
	}

	field := func(name string) model.FormulaArg { return model.FormulaArg{Type: "field", Value: name} }
	constant := func(value string) model.FormulaArg { return model.FormulaArg{Type: "const", Value: value} }

	cases := []struct {
		name string
		spec model.FormulaSpec
		want string
	}{
		{"a product to two decimals", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("net"), field("rate")}}, "1493.82"},
		{"a product rounded by its written digits", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("price"), field("rate")}}, "4.24"},
		{"a product by a constant", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("price"), constant("1.21")}}, "4.24"},
		{"a small product rounded", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("small"), constant("0.21")}}, "0.01"},
		{"a half rounded up", model.FormulaSpec{Name: "SUM", Args: []model.FormulaArg{field("half")}}, "2.68"},
		{"a whole number without decimals", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("qty"), constant("2")}}, "6"},
		{"a product of an empty answer", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("vat"), field("empty")}}, ""},
		{"a product of a missing answer", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("qty"), field("none")}}, ""},
		{"a product of an unknown field", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("qty"), field("gone")}}, ""},
		{"a product with a field slot left empty", model.FormulaSpec{Name: "PRODUCT", Args: []model.FormulaArg{field("qty"), field("")}}, "3"},
		{"a sum skips an empty answer", model.FormulaSpec{Name: "SUM", Args: []model.FormulaArg{field("vat"), field("empty")}}, "0.21"},
		{"a maximum skips an empty answer", model.FormulaSpec{Name: "MAX", Args: []model.FormulaArg{field("vat"), field("empty")}}, "0.21"},
		{"rounded half away from zero", model.FormulaSpec{Name: "ROUND", Args: []model.FormulaArg{constant("1.005"), constant("2")}}, "1.01"},
		{"a negative rounded half away from zero", model.FormulaSpec{Name: "ROUND", Args: []model.FormulaArg{constant("-2.5"), constant("0")}}, "-3"},
		{"a half to a whole number", model.FormulaSpec{Name: "ROUND", Args: []model.FormulaArg{constant("0.5"), constant("0")}}, "1"},
		{"rounded up without float noise", model.FormulaSpec{Name: "ROUNDUP", Args: []model.FormulaArg{constant("0.07"), constant("2")}}, "0.07"},
		{"a negative rounded up away from zero", model.FormulaSpec{Name: "ROUNDUP", Args: []model.FormulaArg{constant("-1.11"), constant("1")}}, "-1.2"},
		{"a negative rounded down toward zero", model.FormulaSpec{Name: "ROUNDDOWN", Args: []model.FormulaArg{constant("-1.19"), constant("1")}}, "-1.1"},
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

// Each want is what the page's roundDecimals gives for the same value.
func TestRoundDecimalsRoundsByTheWrittenDigits(t *testing.T) {
	// Variables, so the products are worked out in float64 as on the page,
	// not exactly as constants.
	price, rate, net := 3.5, 1.21, 1234567.89

	cases := []struct {
		n, places, want float64
	}{
		{4.235, 2, 4.24},
		{price * rate, 2, 4.24},
		{1.005, 2, 1.01},
		{2.675, 2, 2.68},
		{1.255, 2, 1.26},
		{-4.235, 2, -4.24},
		{-1.005, 2, -1.01},
		{0.5, 0, 1},
		{-0.5, 0, -1},
		{4.24, 2, 4.24},
		{net * rate, 2, 1493827.15},
		{1e21, 2, 1e21},
		{1e-7, 2, 0},
	}

	for _, tc := range cases {
		if got := roundDecimals(tc.n, tc.places, math.Round); got != tc.want {
			t.Errorf("roundDecimals(%v, %v) = %v, want %v", tc.n, tc.places, got, tc.want)
		}
	}
}
