import * as assert from "node:assert/strict";
import {test} from "node:test";
import {calendarDayDistance} from "../calendar/date";
import {
    getTimelineBar,
    getTimelineCreateSpan,
    getTimelineDayAt,
    getTimelineHeader,
    getTimelineRange,
    getTimelineSpan,
    getTimelineUnitStart,
    getTimelineWidth,
    getTimelineX,
    getTimelineZoom,
    moveTimelineDates,
    resizeTimelineDates,
    scheduleTimelineDate,
    TIMELINE_DAY_WIDTH,
    TIMELINE_ZOOMS,
} from "./date";

const day = (date: string) => new Date(`${date}T00:00:00`).getTime();
const at = (date: string, time: string) => new Date(`${date}T${time}`).getTime();

test("timeline zoom falls back to week and scales get coarser", () => {
    assert.equal(getTimelineZoom("quarter"), "quarter");
    assert.equal(getTimelineZoom(""), "week");
    assert.equal(getTimelineZoom("year"), "week");
    for (let i = 1; i < TIMELINE_ZOOMS.length; i++) {
        assert.ok(TIMELINE_DAY_WIDTH[TIMELINE_ZOOMS[i]] < TIMELINE_DAY_WIDTH[TIMELINE_ZOOMS[i - 1]]);
    }
});

test("timeline units align to Monday, month and quarter starts", () => {
    assert.equal(getTimelineUnitStart(at("2026-09-17", "15:30:00"), "day"), day("2026-09-17"));
    assert.equal(getTimelineUnitStart(day("2026-09-20"), "week"), day("2026-09-14"));
    assert.equal(getTimelineUnitStart(day("2026-09-14"), "week"), day("2026-09-14"));
    assert.equal(getTimelineUnitStart(day("2026-09-17"), "month"), day("2026-09-01"));
    assert.equal(getTimelineUnitStart(day("2026-09-17"), "quarter"), day("2026-07-01"));
    assert.equal(getTimelineUnitStart(day("2026-12-31"), "quarter"), day("2026-10-01"));
});

test("timeline spans include all-day end dates and exclude a timed midnight end", () => {
    assert.deepEqual(getTimelineSpan({start: day("2026-09-01"), end: day("2026-09-03"), isNotTime: true}),
        {startDay: day("2026-09-01"), endDay: day("2026-09-03")});
    assert.deepEqual(getTimelineSpan({start: at("2026-09-01", "09:00:00"), end: day("2026-09-03"), isNotTime: false}),
        {startDay: day("2026-09-01"), endDay: day("2026-09-02")});
    assert.deepEqual(getTimelineSpan({start: at("2026-09-01", "09:00:00"), end: at("2026-09-01", "18:00:00"), isNotTime: false}),
        {startDay: day("2026-09-01"), endDay: day("2026-09-01")});
    assert.deepEqual(getTimelineSpan({start: day("2026-09-05"), end: day("2026-09-01"), isNotTime: true}),
        {startDay: day("2026-09-05"), endDay: day("2026-09-05")});
});

test("timeline range covers today, every item and the zoom padding", () => {
    const today = day("2026-09-17");
    const spans = [{startDay: day("2026-06-02"), endDay: day("2026-06-10")}, {startDay: day("2027-01-05"), endDay: day("2027-02-01")}];
    for (const zoom of TIMELINE_ZOOMS) {
        const range = getTimelineRange(spans, zoom, today);
        assert.ok(range.start <= day("2026-06-02") && range.end > day("2027-02-01"), zoom);
        assert.equal(getTimelineUnitStart(range.start, zoom), range.start, zoom);
        assert.equal(getTimelineUnitStart(range.end, zoom), range.end, zoom);
        assert.equal(getTimelineWidth(range, zoom), calendarDayDistance(range.start, range.end) * TIMELINE_DAY_WIDTH[zoom]);
    }
    const empty = getTimelineRange([], "day", today);
    assert.ok(empty.start < today && empty.end > today);
    assert.ok(calendarDayDistance(empty.start, empty.end) >= 60);
    const quarter = getTimelineRange([], "quarter", today);
    assert.ok(calendarDayDistance(quarter.start, quarter.end) >= 1100);
});

test("timeline positions convert between days and pixels", () => {
    const range = {start: day("2026-09-14"), end: day("2026-10-12")};
    assert.equal(getTimelineX(day("2026-09-14"), range, "week"), 0);
    assert.equal(getTimelineX(at("2026-09-16", "13:00:00"), range, "week"), 2 * TIMELINE_DAY_WIDTH.week);
    assert.equal(getTimelineDayAt(2 * TIMELINE_DAY_WIDTH.week + 1, range, "week"), day("2026-09-16"));
    assert.equal(getTimelineDayAt(-50, range, "week"), day("2026-09-14"));
    assert.equal(getTimelineDayAt(1e9, range, "week"), day("2026-10-11"));
    const bar = getTimelineBar({startDay: day("2026-09-15"), endDay: day("2026-09-17")}, range, "day");
    assert.deepEqual(bar, {left: TIMELINE_DAY_WIDTH.day, width: 3 * TIMELINE_DAY_WIDTH.day, clippedStart: false, clippedEnd: false});
    const clipped = getTimelineBar({startDay: day("2026-09-01"), endDay: day("2026-12-01")}, range, "month");
    assert.deepEqual(clipped, {left: 0, width: 28 * TIMELINE_DAY_WIDTH.month, clippedStart: true, clippedEnd: true});
    assert.equal(getTimelineBar({startDay: day("2026-08-01"), endDay: day("2026-08-02")}, range, "week"), undefined);
});

test("timeline headers group days by month and months by year", () => {
    const range = {start: day("2026-09-28"), end: day("2026-10-05")};
    const days = getTimelineHeader(range, "day", "en");
    assert.deepEqual(days.top.map(cell => cell.days), [3, 4]);
    assert.deepEqual(days.bottom.map(cell => cell.label), ["28", "29", "30", "1", "2", "3", "4"]);
    const weeks = getTimelineHeader({start: day("2026-09-14"), end: day("2026-10-12")}, "week", "en");
    assert.deepEqual(weeks.bottom.map(cell => cell.days), [7, 7, 7, 7]);
    assert.match(weeks.bottom[0].title, /^2026-W38/);
    const quarters = getTimelineHeader({start: day("2026-10-01"), end: day("2027-07-01")}, "quarter", "en");
    assert.deepEqual(quarters.bottom.map(cell => cell.label), ["Q4", "Q1", "Q2"]);
    assert.deepEqual(quarters.top.map(cell => cell.label), ["2026", "2027"]);
    const months = getTimelineHeader({start: day("2026-11-01"), end: day("2027-02-01")}, "month", "en");
    assert.equal(months.bottom.length, 3);
    assert.equal(months.bottom.reduce((total, cell) => total + cell.days, 0), calendarDayDistance(day("2026-11-01"), day("2027-02-01")));
});

test("moving a bar keeps its duration and time of day", () => {
    const ranged = {start: {content: at("2026-09-01", "09:30:00"), isNotEmpty: true, isNotTime: false, hasEndDate: true,
        content2: at("2026-09-03", "17:00:00"), isNotEmpty2: true}};
    const moved = moveTimelineDates(ranged, 5);
    assert.equal(moved.start.content, at("2026-09-06", "09:30:00"));
    assert.equal(moved.start.content2, at("2026-09-08", "17:00:00"));
    assert.equal(moved.end, undefined);
    const withEnd = moveTimelineDates({start: {content: day("2026-09-01"), isNotEmpty: true, isNotTime: true},
        end: {content: day("2026-09-04"), isNotEmpty: true, isNotTime: true}}, -2);
    assert.equal(withEnd.start.content, day("2026-08-30"));
    assert.equal(withEnd.end.content, day("2026-09-02"));
    const emptyEnd = moveTimelineDates({start: {content: day("2026-09-01"), isNotEmpty: true}, end: {isNotEmpty: false}}, 1);
    assert.deepEqual(emptyEnd.end, {isNotEmpty: false});
    // 跨越夏令时切换时仍按本地日历日移动。
    const previous = process.env.TZ;
    process.env.TZ = "Europe/Berlin";
    try {
        const start = new Date(2026, 2, 28).getTime();
        const dst = moveTimelineDates({start: {content: start, isNotEmpty: true, isNotTime: true}}, 2);
        assert.equal(new Date(dst.start.content).getHours(), 0);
        assert.equal(new Date(dst.start.content).getDate(), 30);
    } finally {
        process.env.TZ = previous;
    }
});

test("resizing a bar edits one endpoint and rejects reversed ranges", () => {
    const single = {start: {content: day("2026-09-10"), isNotEmpty: true, isNotTime: true}};
    const extended = resizeTimelineDates(single, "end", day("2026-09-14"), false);
    assert.deepEqual(extended.start, {...single.start, hasEndDate: true, isNotEmpty2: true, content2: day("2026-09-14")});
    const earlier = resizeTimelineDates({start: extended.start}, "start", day("2026-09-08"), false);
    assert.equal(earlier.start.content, day("2026-09-08"));
    assert.equal(earlier.start.content2, day("2026-09-14"));
    assert.equal(resizeTimelineDates({start: extended.start}, "start", day("2026-09-15"), false), undefined);
    assert.equal(resizeTimelineDates(single, "end", day("2026-09-09"), false), undefined);
    const collapsed = resizeTimelineDates({start: extended.start}, "end", day("2026-09-10"), false);
    assert.equal(collapsed.start.hasEndDate, false);

    const timed = {start: {content: at("2026-09-10", "09:00:00"), isNotEmpty: true, isNotTime: false, hasEndDate: true,
        content2: at("2026-09-10", "18:00:00"), isNotEmpty2: true}};
    const timedEnd = resizeTimelineDates(timed, "end", day("2026-09-12"), false);
    assert.equal(timedEnd.start.content2, at("2026-09-12", "18:00:00"));

    const withEndField = {start: {content: day("2026-09-10"), isNotEmpty: true, isNotTime: true}, end: {isNotEmpty: false}};
    const endField = resizeTimelineDates(withEndField, "end", day("2026-09-20"), true);
    assert.deepEqual(endField.start, withEndField.start);
    assert.deepEqual(endField.end, {isNotEmpty: true, content: day("2026-09-20"), isNotTime: true});
    const startField = resizeTimelineDates(endField, "start", day("2026-09-20"), true);
    assert.equal(startField.start.content, day("2026-09-20"));
    assert.equal(resizeTimelineDates(endField, "start", day("2026-09-21"), true), undefined);
    assert.equal(resizeTimelineDates({start: {isNotEmpty: false}}, "end", day("2026-09-21"), false), undefined);
});

test("scheduling and drag creation produce all-day spans", () => {
    assert.deepEqual(scheduleTimelineDate(undefined, at("2026-09-10", "13:00:00")),
        {content: day("2026-09-10"), isNotEmpty: true, isNotTime: true, hasEndDate: false, isNotEmpty2: false});
    assert.equal(scheduleTimelineDate({content2: 5, formattedContent: "x"}, day("2026-09-10")).formattedContent, "x");
    assert.deepEqual(getTimelineCreateSpan(day("2026-09-12"), at("2026-09-09", "08:00:00")),
        {startDay: day("2026-09-09"), endDay: day("2026-09-12")});
});
