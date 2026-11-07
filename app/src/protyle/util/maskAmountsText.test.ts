import assert from "node:assert/strict";
import test from "node:test";
import {findAmountRanges} from "./maskAmountsText";

const masked = (text: string) => findAmountRanges(text).map(([start, end]) => text.slice(start, end));

test("amounts with separators and decimals are found whole", () => {
    assert.deepEqual(masked("Paid 1 200 000 FCFA, then 35,000.50 and 12.5%"), ["1 200 000", "35,000.50", "12.5"]);
    assert.deepEqual(masked("Rent: 250 000 XOF"), ["250 000"]);
    assert.deepEqual(masked("items 12, 13."), ["12", "13"]);
});

test("dates and times stay visible", () => {
    assert.deepEqual(masked("2026-11-07 at 10:30, due 25/11/2026"), []);
    assert.deepEqual(masked("Week 2026-W45 cost 3000"), ["3000"]);
    assert.deepEqual(masked("No numbers here"), []);
});
