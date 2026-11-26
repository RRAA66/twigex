// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/twigex/twigex/model"
)

// A calculation is never stored: the browser works it out from the row with
// evaluateFormulaForRow in frontend/src/utils/projects/formulas.js. The
// submissions export works it out here the same way, with JavaScript's number,
// text and date rules, so the file shows what the submissions page shows.
// A change to either side belongs in both.

// formCalculation works formula out from a row read by calculationRow, in the
// user's timezone at now, and shows the result as the page does.
func formCalculation(formula *model.FormulaSpec, row map[string]any, loc *time.Location, now time.Time) string {
	if formula == nil || formula.Name == "" {
		return ""
	}

	calc := formulaCalc{row: row, loc: loc, now: now}

	result := calc.evaluate(formula)
	if result == nil || result == "" {
		return ""
	}

	return jsTrim(calc.format(result))
}

type formulaCalc struct {
	row map[string]any
	loc *time.Location
	now time.Time
}

// calculationRow reads a row as the page gets it, through JSON, with a picked
// row or option counted by its name.
func calculationRow(row map[string]any) map[string]any {
	var decoded map[string]any
	if b, err := json.Marshal(row); err != nil || json.Unmarshal(b, &decoded) != nil {
		return map[string]any{}
	}

	for name, value := range decoded {
		switch v := value.(type) {
		case []any:
			names := make([]string, len(v))
			for i, item := range v {
				if m, ok := item.(map[string]any); ok && m["name"] != nil {
					names[i] = jsString(m["name"])
				}
			}
			decoded[name] = strings.Join(names, ", ")
		case map[string]any:
			decoded[name] = v["name"]
		}
	}

	return decoded
}

func readsFields(spec *model.FormulaSpec) bool {
	for _, a := range spec.Args {
		if a.Type == "field" && jsTrim(stripFormulaQuotes(a.Value)) != "" {
			return true
		}
	}

	return false
}

func (c formulaCalc) resolve(arg model.FormulaArg) any {
	switch arg.Type {
	case "field":
		return c.row[stripFormulaQuotes(arg.Value)]
	case "const":
		return stripFormulaQuotes(arg.Value)
	}

	return nil
}

func (c formulaCalc) evaluate(spec *model.FormulaSpec) any {
	args := make([]any, len(spec.Args))
	for i, a := range spec.Args {
		args[i] = c.resolve(a)
	}

	arg := func(i int) any {
		if i < len(args) {
			return args[i]
		}

		return nil
	}

	// parseInt(arg ?? fallback, 10) || fallback, as the page reads a count.
	intArg := func(i int, fallback float64) float64 {
		v := arg(i)
		if v == nil {
			v = fallback
		}

		if n := jsParseInt(v); !math.IsNaN(n) && n != 0 {
			return n
		}

		return fallback
	}

	text := func(i int) string {
		if arg(i) == nil {
			return ""
		}

		return jsString(arg(i))
	}

	fn := strings.ToUpper(spec.Name)

	switch fn {
	case "SUM", "AVERAGE", "MIN", "MAX", "PRODUCT":
		// An aggregate over no fields shows the cell's stored value, which the
		// page reads under no name.
		if !readsFields(spec) {
			return ""
		}

		var nums []float64
		for _, a := range args {
			if n := jsToNumber(a); !math.IsNaN(n) && !math.IsInf(n, 0) {
				nums = append(nums, n)
			}
		}

		return aggregate(fn, nums)
	case "ROUND", "ROUNDUP", "ROUNDDOWN":
		n := jsToNumber(arg(0))
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return ""
		}

		p := math.Pow(10, math.Max(0, intArg(1, 0)))
		switch fn {
		case "ROUNDUP":
			return math.Ceil(n*p) / p
		case "ROUNDDOWN":
			return math.Floor(n*p) / p
		}

		return jsRound(float64(n*p)) / p
	case "ABS":
		n := jsToNumber(arg(0))
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return ""
		}

		return math.Abs(n)
	case "POWER":
		base, exp := jsToNumber(arg(0)), jsToNumber(arg(1))
		if math.IsNaN(base) || math.IsInf(base, 0) || math.IsNaN(exp) || math.IsInf(exp, 0) {
			return ""
		}

		return math.Pow(base, exp)
	case "IF":
		if jsTruthy(arg(0)) {
			return arg(1)
		}

		return arg(2)
	case "AND":
		for _, a := range args {
			if !jsTruthy(a) {
				return false
			}
		}

		return true
	case "OR":
		for _, a := range args {
			if jsTruthy(a) {
				return true
			}
		}

		return false
	case "NOT":
		return !jsTruthy(arg(0))
	case "CONCATENATE":
		var b strings.Builder
		for i := range args {
			b.WriteString(text(i))
		}

		return b.String()
	case "LEFT":
		s := utf16.Encode([]rune(text(0)))
		n := int(math.Min(math.Max(0, intArg(1, 1)), float64(len(s))))

		return string(utf16.Decode(s[:n]))
	case "RIGHT":
		s := utf16.Encode([]rune(text(0)))
		n := int(math.Min(math.Max(0, intArg(1, 1)), float64(len(s))))
		if n == 0 {
			// slice(-0) is the whole text.
			n = len(s)
		}

		return string(utf16.Decode(s[len(s)-n:]))
	case "MID":
		s := utf16.Encode([]rune(text(0)))
		start := math.Max(0, intArg(1, 1)-1)
		end := start + math.Max(0, intArg(2, 1))
		size := float64(len(s))

		return string(utf16.Decode(s[int(math.Min(start, size)):int(math.Min(end, size))]))
	case "LEN":
		return float64(len(utf16.Encode([]rune(text(0)))))
	case "TRIM":
		return strings.Join(strings.Fields(text(0)), " ")
	case "UPPER":
		return strings.ToUpper(text(0))
	case "LOWER":
		return strings.ToLower(text(0))
	case "TODAY":
		y, m, d := c.now.In(c.loc).Date()

		return time.Date(y, m, d, 0, 0, 0, 0, c.loc)
	case "NOW":
		return c.now.In(c.loc)
	case "DATE":
		y := jsParseIntOr(arg(0), 0)
		m := jsParseIntOr(arg(1), 1) - 1
		d := jsParseIntOr(arg(2), 1)
		if y >= 0 && y <= 99 {
			y += 1900
		}

		if math.Abs(y) > 275000 || math.Abs(m) > 3e6 || math.Abs(d) > 1e8 {
			return ""
		}

		return time.Date(int(y), time.Month(int(m)+1), int(d), 0, 0, 0, 0, c.loc)
	case "YEAR", "MONTH", "DAY":
		t, ok := c.date(arg(0))
		if !ok {
			return ""
		}

		switch fn {
		case "YEAR":
			return float64(t.Year())
		case "MONTH":
			return float64(t.Month())
		}

		return float64(t.Day())
	case "DATEDIF":
		unit := "D"
		if arg(2) != nil {
			unit = jsString(arg(2))
		}

		unit = stripFormulaQuotes(unit)
		if unit == "" {
			unit = "D"
		}

		if out, ok := c.datedif(arg(0), arg(1), strings.ToUpper(unit)); ok {
			return out
		}

		return ""
	case "WEEKDAY":
		t, ok := c.date(arg(0))
		if !ok {
			return ""
		}

		day := float64(t.Weekday())
		if kind := intArg(1, 1); kind == 2 || kind == 3 {
			if day == 0 {
				return float64(7)
			}

			return day
		}

		return day + 1
	}

	return ""
}

func aggregate(fn string, nums []float64) any {
	if len(nums) == 0 {
		if fn == "SUM" || fn == "AVERAGE" {
			return float64(0)
		}

		return ""
	}

	out := nums[0]
	for _, n := range nums[1:] {
		switch fn {
		case "SUM", "AVERAGE":
			out += n
		case "PRODUCT":
			out *= n
		case "MIN":
			out = math.Min(out, n)
		case "MAX":
			out = math.Max(out, n)
		}
	}

	if fn == "AVERAGE" {
		return out / float64(len(nums))
	}

	return out
}

func (c formulaCalc) datedif(start, end any, unit string) (float64, bool) {
	s, ok := c.date(start)
	if !ok {
		return 0, false
	}

	e, ok := c.date(end)
	if !ok {
		return 0, false
	}

	switch unit {
	case "D":
		return math.Floor(float64(e.UnixMilli()-s.UnixMilli()) / 86400000), true
	case "M":
		months := (e.Year()-s.Year())*12 + int(e.Month()) - int(s.Month())
		if e.Day() < s.Day() {
			months--
		}

		return float64(months), true
	case "Y":
		years := e.Year() - s.Year()
		if e.Month() < s.Month() || (e.Month() == s.Month() && e.Day() < s.Day()) {
			years--
		}

		return float64(years), true
	}

	return 0, false
}

var (
	calcNumberText = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	calcDayText    = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})(?:T00:00:00(?:\.0+)?Z)?$`)
	calcLocalTimes = []string{"2006-01-02T15:04:05.999999999", "2006-01-02T15:04", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04"}
)

// date reads a value as the page's coerceDate does: a number as seconds, or
// milliseconds from 1e12, and a day as that day at midnight where the user is.
func (c formulaCalc) date(v any) (time.Time, bool) {
	if s, ok := v.(string); ok {
		v = stripFormulaQuotes(jsTrim(s))
	}

	switch t := v.(type) {
	case nil:
		return time.Time{}, false
	case time.Time:
		return t.In(c.loc), true
	case bool:
		if t {
			return time.UnixMilli(1).In(c.loc), true
		}

		return time.UnixMilli(0).In(c.loc), true
	case float64:
		return c.fromNumber(t)
	case string:
		if t == "" {
			return time.Time{}, false
		}

		if calcNumberText.MatchString(t) {
			n, _ := strconv.ParseFloat(t, 64)
			return c.fromNumber(n)
		}

		if m := calcDayText.FindStringSubmatch(t); m != nil {
			y, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[3])

			return time.Date(y, time.Month(mo), d, 0, 0, 0, 0, c.loc), true
		}

		if d, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return d.In(c.loc), true
		}

		for _, layout := range calcLocalTimes {
			if d, err := time.ParseInLocation(layout, t, c.loc); err == nil {
				return d, true
			}
		}
	}

	return time.Time{}, false
}

func (c formulaCalc) fromNumber(n float64) (time.Time, bool) {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return time.Time{}, false
	}

	ms := n
	if n < 1e12 {
		ms = n * 1000
	}

	if math.Abs(ms) > 8.64e15 {
		return time.Time{}, false
	}

	return time.UnixMilli(int64(ms)).In(c.loc), true
}

// format shows a result as the page's formatCellValue does without a field.
func (c formulaCalc) format(v any) string {
	switch t := v.(type) {
	case time.Time:
		day := t.Format("02.01.2006")
		if t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond()/1e6 != 0 {
			return day + t.Format(" 15:04")
		}

		return day
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return jsString(t)
		}

		return jsString(jsRound(float64(t*100)) / 100)
	case nil:
		return ""
	}

	return jsString(v)
}

// jsRound rounds as Math.round: to the nearest whole number, a half up. Its
// callers convert a product to float64 first, which keeps the compiler from
// fusing it into the subtraction here and so rounding it otherwise than
// JavaScript does.
func jsRound(x float64) float64 {
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}

	return r
}

func jsTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case float64:
		return t != 0 && !math.IsNaN(t)
	case string:
		return t != ""
	}

	return true
}

// jsString writes a value as JavaScript's String does.
func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return jsNumber(t)
	}

	return fmt.Sprint(v)
}

func jsNumber(n float64) string {
	switch {
	case math.IsNaN(n):
		return "NaN"
	case math.IsInf(n, 1):
		return "Infinity"
	case math.IsInf(n, -1):
		return "-Infinity"
	case n == 0:
		return "0"
	}

	exp := strconv.FormatFloat(n, 'e', -1, 64)
	at := strings.IndexByte(exp, 'e')
	power, _ := strconv.Atoi(exp[at+1:])
	if power >= -6 && power <= 20 {
		return strconv.FormatFloat(n, 'f', -1, 64)
	}

	sign := "+"
	if power < 0 {
		sign = "-"
		power = -power
	}

	return exp[:at] + "e" + sign + strconv.Itoa(power)
}

var calcNumberStart = regexp.MustCompile(`^-?(\d+\.?\d*|\.\d+)`)

// jsToNumber reads a value as the page's toNumber does: the first comma as
// a decimal point, then the leading number of its digits, points and minus
// signs.
func jsToNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		if math.IsInf(t, 0) {
			return math.NaN()
		}

		return t
	case string:
		cleaned := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' || r == '.' || r == '-' {
				return r
			}

			return -1
		}, strings.Replace(t, ",", ".", 1))

		if n, err := strconv.ParseFloat(calcNumberStart.FindString(cleaned), 64); err == nil {
			return n
		}
	}

	return math.NaN()
}

var calcIntStart = regexp.MustCompile(`^[+-]?\d+`)

// jsParseInt reads a value as parseInt(value, 10) does.
func jsParseInt(v any) float64 {
	n, err := strconv.ParseFloat(calcIntStart.FindString(jsTrim(jsString(v))), 64)
	if err != nil {
		return math.NaN()
	}

	return n
}

func jsParseIntOr(v any, fallback float64) float64 {
	if n := jsParseInt(v); !math.IsNaN(n) && n != 0 {
		return n
	}

	return fallback
}

func jsTrim(s string) string {
	return strings.TrimSpace(s)
}
