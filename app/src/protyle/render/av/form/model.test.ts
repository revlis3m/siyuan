import assert from "node:assert/strict";
import test from "node:test";
import {
    buildFormValues,
    formCellValue,
    getFormDefault,
    getFormSettings,
    isFormFieldShown,
    missingRequiredFields,
    parseFormNumber,
    updateFormField,
} from "./model";

const columns: IAVColumn[] = [
    {id: "name", type: "block", name: "Description"},
    {id: "kind", type: "select", name: "Kind", options: [{name: "Expense", color: "1"}, {name: "Transfer", color: "7"}]},
    {id: "amount", type: "number", name: "Amount"},
    {id: "date", type: "date", name: "Date"},
    {id: "to", type: "relation", name: "To account"},
    {id: "done", type: "checkbox", name: "Done"},
];

const settings = (fields: Partial<IAVFormField>[]): IAVFormSettings =>
    getFormSettings({form: {title: "", description: "", submitLabel: "", fields: fields as IAVFormField[]}});

test("numbers typed with separators", () => {
    assert.equal(parseFormNumber("1 200 000"), 1200000);
    assert.equal(parseFormNumber("35,000"), 35000);
    assert.equal(parseFormNumber("1,5"), 1.5);
    assert.equal(parseFormNumber("1.234,56"), 1234.56);
    assert.equal(parseFormNumber("1,234.56"), 1234.56);
    assert.equal(parseFormNumber("1.200.000"), 1200000);
    assert.equal(parseFormNumber("-20"), -20);
    assert.equal(parseFormNumber("abc"), null);
    assert.equal(parseFormNumber(""), null);
});

test("defaults: fixed value, today and last used", () => {
    const s = settings([
        {keyID: "kind", default: "value", value: "Expense"},
        {keyID: "date", default: "today"},
        {keyID: "to", default: "last"},
        {keyID: "done", default: "value", value: "true"},
    ]);
    const today = new Date(2026, 10, 7, 9);
    const last = {to: {keyID: "to", type: "relation" as TAVCol, relation: {blockIDs: ["acc1"]}}};
    assert.equal(getFormDefault(columns[1], s, last, today), "Expense");
    assert.equal(getFormDefault(columns[3], s, last, today), "2026-11-07");
    assert.deepEqual(getFormDefault(columns[4], s, last, today), ["acc1"]);
    assert.equal(getFormDefault(columns[5], s, last, today), true);
    assert.equal(getFormDefault(columns[2], s, last, today), "");
    assert.deepEqual(getFormDefault(columns[4], s, {}, today), []);
});

test("conditional fields and required checks", () => {
    const s = settings([
        {keyID: "amount", required: true},
        {keyID: "to", required: true, showWhenKeyID: "kind", showWhenValues: ["Transfer"]},
    ]);
    const ids = columns.map(column => column.id);
    assert.equal(isFormFieldShown(s, "to", {kind: "Expense"}, ids), false);
    assert.equal(isFormFieldShown(s, "to", {kind: "Transfer"}, ids), true);
    assert.deepEqual(missingRequiredFields(columns, s, {kind: "Expense", amount: ""}), ["amount"]);
    assert.deepEqual(missingRequiredFields(columns, s, {kind: "Transfer", amount: "5", to: []}), ["to"]);
    assert.deepEqual(missingRequiredFields(columns, s, {kind: "Expense", amount: "5"}), []);
    // A condition on a field that is not in the form does not hide anything.
    assert.equal(isFormFieldShown(s, "to", {}, ["to"]), true);
});

test("values of the new item", () => {
    const s = settings([{keyID: "to", showWhenKeyID: "kind", showWhenValues: ["Transfer"]}]);
    const values = buildFormValues(columns, s, {
        name: " Lunch ", kind: "Expense", amount: "2 500", date: "2026-11-07", to: ["acc2"], done: false,
    });
    assert.deepEqual(values.map(value => value.keyID), ["name", "kind", "amount", "date"]);
    assert.deepEqual(values[0].block, {content: "Lunch"});
    assert.deepEqual(values[1].mSelect, [{content: "Expense", color: "1"}]);
    assert.deepEqual(values[2].number, {content: 2500, isNotEmpty: true});
    assert.equal(new Date(values[3].date.content).getHours(), 12);
    assert.equal(formCellValue(columns[5], true).checkbox.checked, true);
    assert.equal(formCellValue(columns[2], "x"), null);
});

test("field settings back to plain defaults are dropped", () => {
    let s = settings([]);
    s = updateFormField(s, "amount", {required: true});
    assert.equal(s.fields.length, 1);
    s = updateFormField(s, "amount", {required: false});
    assert.equal(s.fields.length, 0);
});
