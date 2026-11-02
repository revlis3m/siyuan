import * as assert from "node:assert/strict";
import {test} from "node:test";
import {
    formatTimelineSpan,
    getTimelineCell,
    getTimelineDateOperations,
    getTimelineDates,
    getTimelineEntries,
    getTimelineFields,
    getTimelineSettings,
    getTimelineSideColumns,
} from "./model";

const day = (date: string) => new Date(`${date}T00:00:00`).getTime();

const columns: IAVColumn[] = [
    {id: "name", name: "Name", type: "block"},
    {id: "start", name: "Start", type: "date", hidden: true},
    {id: "end", name: "End", type: "date", hidden: true},
    {id: "status", name: "Status", type: "status"},
    {id: "created", name: "Created", type: "created", hidden: true},
    {id: "note", name: "Note", type: "text", hidden: true},
];

const row = (id: string, start?: IAVCellDateValue, end?: IAVCellDateValue): IAVRow => ({id, cells: [
    {id: `${id}-name`, value: {id: `${id}-name`, keyID: "name", type: "block", isDetached: true, block: {content: id}}},
    {id: `${id}-start`, value: {id: `${id}-start`, keyID: "start", type: "date", date: start}},
    {id: `${id}-end`, value: {id: `${id}-end`, keyID: "end", type: "date", date: end}},
    {id: `${id}-status`, value: {id: `${id}-status`, keyID: "status", type: "status", mSelect: []}},
    {id: `${id}-created`, value: {id: `${id}-created`, keyID: "created", type: "created", created: {content: 1, isNotEmpty: true}}},
    {id: `${id}-note`, value: {id: `${id}-note`, keyID: "note", type: "text", text: {content: ""}}},
]});

const view = (timeline: Partial<IAVTimelineSettings>, rows: IAVRow[] = [], items: IAVTimelineItem[] = []) => ({
    columns, rows, timelineItems: items,
    timeline: {startKeyID: "", endKeyID: "", zoom: "week", colorBy: "", ...timeline},
}) as unknown as IAVTable;

test("timeline settings fill defaults for views rendered before their settings arrive", () => {
    assert.deepEqual(getTimelineSettings({columns: []} as unknown as IAVTable), {startKeyID: "", endKeyID: "", zoom: "week", colorBy: ""});
    assert.equal(getTimelineSettings(view({zoom: "" as IAVTimelineSettings["zoom"]})).zoom, "week");
    assert.equal(getTimelineSettings(view({zoom: "quarter"})).zoom, "quarter");
});

test("timeline entries pair kernel intervals with rows in the returned order", () => {
    const rows = [row("a"), row("b"), row("c")];
    const items = [
        {id: "c", start: day("2026-09-03"), end: day("2026-09-05"), isNotTime: true, invalid: false, title: "c", blockID: ""},
        {id: "missing", start: 0, end: 0, isNotTime: true, invalid: false, title: "", blockID: ""},
        {id: "a", start: day("2026-09-01"), end: day("2026-09-01"), isNotTime: true, invalid: false, title: "a", blockID: ""},
    ];
    const entries = getTimelineEntries(view({startKeyID: "start"}, rows, items));
    assert.deepEqual(entries.map(entry => [entry.row.id, entry.rowIndex]), [["c", 2], ["a", 0]]);
    assert.deepEqual(entries[0].span, {startDay: day("2026-09-03"), endDay: day("2026-09-05")});
    assert.deepEqual(getTimelineEntries({columns} as unknown as IAVTable), []);
});

test("timeline side column shows the primary field first and only visible properties", () => {
    const shuffled = view({}, []);
    shuffled.columns = [columns[3], columns[0], columns[5]];
    assert.deepEqual(getTimelineSideColumns(shuffled).map(column => column.id), ["name", "status"]);
    assert.deepEqual(getTimelineSideColumns(shuffled, true).map(column => column.id), ["name"]);
});

test("timeline fields decide which dates can be dragged", () => {
    assert.deepEqual(getTimelineFields(view({startKeyID: "start"})), {start: columns[1], end: undefined, editable: true, useEndField: false});
    assert.equal(getTimelineFields(view({startKeyID: "start", endKeyID: "end"})).useEndField, true);
    assert.equal(getTimelineFields(view({startKeyID: "start", endKeyID: "start"})).useEndField, false);
    assert.equal(getTimelineFields(view({startKeyID: "created"})).editable, false);
    assert.equal(getTimelineFields(view({startKeyID: "start", endKeyID: "created"})).editable, false);
    assert.equal(getTimelineFields(view({startKeyID: "note"})).start, undefined);
    const computed = view({startKeyID: "start"});
    computed.columns = columns.map(column => column.id === "start" ? {...column, renderTemplate: "2026-09-01"} : column);
    assert.equal(getTimelineFields(computed).editable, false);
});

test("timeline date writes update only changed fields with undo values", () => {
    const start = {content: day("2026-09-01"), isNotEmpty: true, isNotTime: true};
    const item = row("a", start, {isNotEmpty: false});
    const current = view({startKeyID: "start", endKeyID: "end"}, [item]);
    assert.equal(getTimelineCell(current, item, columns[2]).id, "a-end");
    assert.deepEqual(getTimelineDates(current, item), {start, end: {isNotEmpty: false}});
    const next = {start: {...start, content: day("2026-09-02")}, end: {isNotEmpty: true, content: day("2026-09-04"), isNotTime: true}};
    const operations = getTimelineDateOperations({view: current, row: item, avID: "av", blockID: "block", next});
    assert.deepEqual(operations.doOperations.map(operation => operation.keyID), ["start", "end"]);
    assert.deepEqual(operations.doOperations[0], {action: "updateAttrViewCell", avID: "av", blockID: "block", id: "a-start",
        keyID: "start", rowID: "a", data: {type: "date", date: next.start}});
    assert.deepEqual(operations.undoOperations[1].data, {type: "date", date: {isNotEmpty: false}});
    const unchanged = getTimelineDateOperations({view: current, row: item, avID: "av", blockID: "block",
        next: {start, end: next.end}});
    assert.deepEqual(unchanged.doOperations.map(operation => operation.keyID), ["end"]);
    const withoutEnd = getTimelineDateOperations({view: view({startKeyID: "start"}, [item]), row: item, avID: "av", blockID: "block", next});
    assert.deepEqual(withoutEnd.doOperations.map(operation => operation.keyID), ["start"]);
    assert.equal(getTimelineDates(view({startKeyID: "missing"}), item), undefined);
});

test("timeline span labels show one or two dates", () => {
    const single = formatTimelineSpan({startDay: day("2026-09-01"), endDay: day("2026-09-01")}, "en-US");
    assert.equal(single, new Date(day("2026-09-01")).toLocaleDateString("en-US"));
    assert.equal(formatTimelineSpan({startDay: day("2026-09-01"), endDay: day("2026-09-03")}, "en-US"),
        `${single} - ${new Date(day("2026-09-03")).toLocaleDateString("en-US")}`);
});
