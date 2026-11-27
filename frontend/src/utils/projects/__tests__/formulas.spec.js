// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, it } from "vitest";
import {
    calculateRowValueForRow,
    formatCellValue,
    columnTotal,
    formulaArgsFor,
    isComputedCalculations,
    roundDecimals,
} from "../formulas";

const calc = (formula, extra = {}) => ({
    name: "total",
    header_usage: "calculations",
    formula,
    ...extra,
});
const fields = (...names) => names.map((value) => ({ type: "field", value }));

describe("calculateRowValueForRow", () => {
    it("adds the fields a SUM names", () => {
        const header = calc({ name: "SUM", args: fields("hours", "extra") });

        expect(calculateRowValueForRow(header, { hours: "2.5", extra: 4 })).toBe(6.5);
    });

    it("skips a field that holds no number", () => {
        const header = calc({ name: "SUM", args: fields("hours", "extra") });

        expect(calculateRowValueForRow(header, { hours: "3", extra: "" })).toBe(3);
    });

    it("averages the fields an AVERAGE names", () => {
        const header = calc({ name: "AVERAGE", args: fields("a", "b") });

        expect(calculateRowValueForRow(header, { a: 2, b: 4 })).toBe(3);
    });

    it("reads its own stored value for an aggregate over no fields", () => {
        const header = calc({ name: "SUM", args: [] });

        expect(calculateRowValueForRow(header, { total: "12" })).toBe(12);
    });

    it("works out a function of constants", () => {
        const header = calc({
            name: "ROUND",
            args: [
                { type: "const", value: "2.345" },
                { type: "const", value: "2" },
            ],
        });

        expect(calculateRowValueForRow(header, {})).toBe(2.35);
    });

    it("is empty for a PRODUCT with a field left unanswered", () => {
        const header = calc({ name: "PRODUCT", args: fields("qty", "price") });

        expect(calculateRowValueForRow(header, { qty: 1, price: "" })).toBe("");
        expect(calculateRowValueForRow(header, { qty: 1, price: null })).toBe("");
        expect(calculateRowValueForRow(header, { qty: 1 })).toBe("");
        expect(calculateRowValueForRow(header, { qty: 2, price: "3.5" })).toBe(7);
    });

    it("multiplies by a constant beside a field", () => {
        const header = calc({
            name: "PRODUCT",
            args: [...fields("net"), { type: "const", value: "1.21" }],
        });

        expect(formatCellValue(calculateRowValueForRow(header, { net: "3.50" }))).toBe("4.24");
        expect(calculateRowValueForRow(header, { net: "" })).toBe("");
    });

    it("rounds a half away from zero by its written digits", () => {
        const round = (value, digits) =>
            calculateRowValueForRow(
                calc({
                    name: "ROUND",
                    args: [
                        { type: "const", value },
                        { type: "const", value: digits },
                    ],
                }),
                {},
            );

        expect(round("1.005", "2")).toBe(1.01);
        expect(round("2.675", "2")).toBe(2.68);
        expect(round("1.255", "2")).toBe(1.26);
        expect(round("-1.005", "2")).toBe(-1.01);
        expect(round("0.5", "0")).toBe(1);
        expect(round("-2.5", "0")).toBe(-3);
    });

    it("rounds up away from zero and down toward it without float noise", () => {
        const fn = (name, value, digits) =>
            calculateRowValueForRow(
                calc({
                    name,
                    args: [
                        { type: "const", value },
                        { type: "const", value: digits },
                    ],
                }),
                {},
            );

        expect(fn("ROUNDUP", "1.1", "1")).toBe(1.1);
        expect(fn("ROUNDUP", "0.07", "2")).toBe(0.07);
        expect(fn("ROUNDUP", "1.11", "1")).toBe(1.2);
        expect(fn("ROUNDUP", "-1.11", "1")).toBe(-1.2);
        expect(fn("ROUNDDOWN", "1.19", "1")).toBe(1.1);
        expect(fn("ROUNDDOWN", "-1.19", "1")).toBe(-1.1);
    });

    it("is empty for a field that is not a calculation", () => {
        expect(calculateRowValueForRow({ name: "x", header_usage: "text" }, { x: 1 })).toBe("");
    });
});

describe("isComputedCalculations", () => {
    it("is true for an aggregate over fields", () => {
        expect(isComputedCalculations(calc({ name: "SUM", args: fields("a") }))).toBe(true);
    });

    it("is false for an aggregate over no fields, which is typed in", () => {
        expect(isComputedCalculations(calc({ name: "SUM", args: [] }))).toBe(false);
    });

    it("is true for any other function", () => {
        expect(isComputedCalculations(calc({ name: "TODAY", args: [] }))).toBe(true);
    });

    it("is false for a field that is not a calculation", () => {
        expect(isComputedCalculations({ header_usage: "text" })).toBe(false);
    });
});

describe("formatCellValue", () => {
    it("shows up to two decimals", () => {
        expect(formatCellValue(2.5)).toBe("2.5");
        expect(formatCellValue(1 / 3)).toBe("0.33");
    });

    it("shows the decimals a field names", () => {
        expect(formatCellValue(2.5, { decimals: 2 })).toBe("2.50");
    });

    it("rounds a half away from zero by its written digits", () => {
        expect(formatCellValue(3.5 * 1.21)).toBe("4.24");
        expect(formatCellValue(1.005)).toBe("1.01");
        expect(formatCellValue(2.675)).toBe("2.68");
        expect(formatCellValue(1.255)).toBe("1.26");
        expect(formatCellValue(-4.235)).toBe("-4.24");
        expect(formatCellValue(-1.005, { decimals: 2 })).toBe("-1.01");
        expect(formatCellValue(1.005, { decimals: 2 })).toBe("1.01");
        expect(formatCellValue(0.5, { decimals: 0 })).toBe("1");
        expect(formatCellValue(-0.5, { decimals: 0 })).toBe("-1");
        expect(formatCellValue(-0.001)).toBe("0");
    });

    it("shows nothing for an empty value", () => {
        expect(formatCellValue("").trim()).toBe("");
        expect(formatCellValue(null).trim()).toBe("");
    });
});

describe("a field slot left empty", () => {
    const header = calc({ name: "SUM", args: fields("") });

    it("counts as no field, so the value is typed in", () => {
        expect(isComputedCalculations(header)).toBe(false);
    });

    it("reads the stored value", () => {
        expect(calculateRowValueForRow(header, { total: "7" })).toBe(7);
    });
});

describe("columnTotal", () => {
    const rows = [
        { a: 1, b: 2, total: "4" },
        { a: 3, b: 4, total: "6" },
        { a: "", b: "x", total: "" },
    ];

    it("adds a SUM over fields across the rows", () => {
        expect(columnTotal(calc({ name: "SUM", args: fields("a", "b") }), rows)).toBe(10);
    });

    it("averages each row's AVERAGE, counting a row with no numbers as 0", () => {
        expect(columnTotal(calc({ name: "AVERAGE", args: fields("a", "b") }), rows)).toBeCloseTo(
            5 / 3,
        );
    });

    it("takes the smallest of a MIN over fields, skipping a row with no numbers", () => {
        expect(columnTotal(calc({ name: "MIN", args: fields("a", "b") }), rows)).toBe(1);
    });

    it("adds each row's PRODUCT across the rows, not multiplies them", () => {
        expect(columnTotal(calc({ name: "PRODUCT", args: fields("a", "b") }), rows)).toBe(14);
    });

    it("adds the typed values of a PRODUCT over no fields", () => {
        expect(columnTotal(calc({ name: "PRODUCT", args: [] }), rows)).toBe(10);
    });

    it("combines the typed values of an aggregate over no fields", () => {
        expect(columnTotal(calc({ name: "MAX", args: [] }), rows)).toBe(6);
    });

    it("adds the row results as they are shown", () => {
        const vat = calc({
            name: "PRODUCT",
            args: [...fields("net"), { type: "const", value: "0.21" }],
        });
        const nets = [{ net: "0.05" }, { net: "0.05" }, { net: "0.05" }];

        expect(formatCellValue(calculateRowValueForRow(vat, nets[0]))).toBe("0.01");
        expect(formatCellValue(columnTotal(vat, nets))).toBe("0.03");
    });

    it("adds the row results to the decimals the column names", () => {
        const whole = calc({ name: "PRODUCT", args: fields("a", "b") }, { decimals: 0 });
        const halves = [
            { a: 0.5, b: 1 },
            { a: 0.5, b: 1 },
        ];

        expect(columnTotal(whole, halves)).toBe(2);
    });

    it("leaves out a row whose PRODUCT has an unanswered field", () => {
        const total = calc({ name: "PRODUCT", args: fields("qty", "price") });

        expect(
            columnTotal(total, [
                { qty: 2, price: "3" },
                { qty: 1, price: "" },
            ]),
        ).toBe(6);
        expect(columnTotal(total, [{ qty: 1, price: "" }])).toBe("");
    });

    it("is empty for a column that is not a calculation, or without rows", () => {
        expect(columnTotal({ name: "a", header_usage: "number" }, rows)).toBe("");
        expect(columnTotal(calc({ name: "SUM", args: [] }), null)).toBe("");
    });
});

describe("roundDecimals", () => {
    it("rounds the classic float halves up", () => {
        expect(roundDecimals(4.235, 2)).toBe(4.24);
        expect(roundDecimals(3.5 * 1.21, 2)).toBe(4.24);
        expect(roundDecimals(1.005, 2)).toBe(1.01);
        expect(roundDecimals(2.675, 2)).toBe(2.68);
        expect(roundDecimals(1.255, 2)).toBe(1.26);
    });

    it("rounds a negative half away from zero, without a negative zero", () => {
        expect(roundDecimals(-4.235, 2)).toBe(-4.24);
        expect(roundDecimals(-0.5, 0)).toBe(-1);
        expect(Object.is(roundDecimals(-0.001, 2), 0)).toBe(true);
    });

    it("keeps a value already at its decimals and handles large or tiny ones", () => {
        expect(roundDecimals(4.24, 2)).toBe(4.24);
        expect(roundDecimals(1234567.89 * 1.21, 2)).toBe(1493827.15);
        expect(roundDecimals(1e21, 2)).toBe(1e21);
        expect(roundDecimals(1e-7, 2)).toBe(0);
    });
});

describe("formulaArgsFor", () => {
    const headers = [{ name: "c1" }, { header_name: "c2", name: "x" }];

    it("gives a variadic formula's field arguments their headers", () => {
        const spec = {
            args: [
                { type: "field", value: "c1" },
                { type: "constant", value: 3 },
                { type: "field", value: "gone" },
            ],
        };

        expect(formulaArgsFor({ arity: "variadic" }, spec, headers)).toEqual([headers[0], 3, null]);
    });

    it("reads a fixed formula's field arguments from its catalog entry", () => {
        const formula = { args: [{ kind: "field" }, { kind: "number" }] };
        const spec = { args: [{ value: "c2" }, { value: undefined }] };

        expect(formulaArgsFor(formula, spec, headers)).toEqual([headers[1], ""]);
    });
});
