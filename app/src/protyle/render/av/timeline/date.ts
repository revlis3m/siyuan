import {addCalendarDays, calendarDay, calendarDayDistance, getISOWeek} from "../calendar/date";

export type TTimelineZoom = "day" | "week" | "month" | "quarter";

export const TIMELINE_ZOOMS: TTimelineZoom[] = ["day", "week", "month", "quarter"];

// 每天占用的像素宽度，决定各刻度下条形的长度和横向滚动范围。
export const TIMELINE_DAY_WIDTH: Record<TTimelineZoom, number> = {day: 44, week: 22, month: 8, quarter: 3};

// 可见范围在条目区间和今天之外保留的天数，便于拖动到相邻日期。
const TIMELINE_PADDING_DAYS: Record<TTimelineZoom, number> = {day: 14, week: 28, month: 92, quarter: 183};

// 可见范围的最少天数，保证内容较少时时间轴仍能铺满视图。
const TIMELINE_MIN_DAYS: Record<TTimelineZoom, number> = {day: 60, week: 120, month: 400, quarter: 1100};

// 单个视图最多渲染的天数，避免极端日期生成超长的滚动区域。
export const TIMELINE_MAX_DAYS = 365 * 60;

export const getTimelineZoom = (zoom?: string): TTimelineZoom =>
    TIMELINE_ZOOMS.includes(zoom as TTimelineZoom) ? zoom as TTimelineZoom : "week";

export interface ITimelineRange {
    /** 第一天的本地零点，包含该日。 */
    start: number;
    /** 最后一天之后的本地零点，不包含该日。 */
    end: number;
}

export interface ITimelineSpan {
    /** 第一天的本地零点。 */
    startDay: number;
    /** 最后一天的本地零点，包含该日。 */
    endDay: number;
}

const localDate = (year: number, month: number, day: number) => {
    const date = new Date(0);
    date.setFullYear(year, month, day);
    date.setHours(0, 0, 0, 0);
    return date.getTime();
};

// 按刻度对齐到所在单位的第一天：周按周一，月和季度按当月或当季第一天。
export const getTimelineUnitStart = (value: number, zoom: TTimelineZoom) => {
    const day = calendarDay(value);
    const date = new Date(day);
    switch (zoom) {
        case "week":
            return addCalendarDays(day, -((date.getDay() + 6) % 7));
        case "month":
            return localDate(date.getFullYear(), date.getMonth(), 1);
        case "quarter":
            return localDate(date.getFullYear(), date.getMonth() - date.getMonth() % 3, 1);
        default:
            return day;
    }
};

export const getTimelineNextUnit = (unitStart: number, zoom: TTimelineZoom) => {
    const date = new Date(unitStart);
    switch (zoom) {
        case "week":
            return addCalendarDays(unitStart, 7);
        case "month":
            return localDate(date.getFullYear(), date.getMonth() + 1, 1);
        case "quarter":
            return localDate(date.getFullYear(), date.getMonth() + 3, 1);
        default:
            return addCalendarDays(unitStart, 1);
    }
};

// 全天日期的结束日期包含当天；带时间的结束端点恰为零点时属于前一天。
export const getTimelineSpan = (item: Pick<IAVTimelineItem, "start" | "end" | "isNotTime">): ITimelineSpan => {
    const startDay = calendarDay(item.start);
    let endDay = calendarDay(Math.max(item.start, item.end));
    if (!item.isNotTime && item.end > item.start && endDay === item.end) {
        endDay = addCalendarDays(endDay, -1);
    }
    return {startDay, endDay: Math.max(startDay, endDay)};
};

// 可见范围覆盖今天、全部条目和两侧留白，并按刻度单位对齐。
export const getTimelineRange = (spans: ITimelineSpan[], zoom: TTimelineZoom, today = calendarDay(Date.now())): ITimelineRange => {
    let first = today;
    let last = today;
    spans.forEach(span => {
        first = Math.min(first, span.startDay);
        last = Math.max(last, span.endDay);
    });
    if (calendarDayDistance(first, last) > TIMELINE_MAX_DAYS) {
        // 条目跨度过大时以今天为中心截取，范围外的条形从边缘延伸进入视图。
        first = Math.max(first, addCalendarDays(today, -TIMELINE_MAX_DAYS / 2));
        last = Math.min(last, addCalendarDays(first, TIMELINE_MAX_DAYS));
    }
    const padding = TIMELINE_PADDING_DAYS[zoom];
    let start = getTimelineUnitStart(addCalendarDays(first, -padding), zoom);
    let end = getTimelineNextUnit(getTimelineUnitStart(addCalendarDays(last, padding), zoom), zoom);
    const missing = TIMELINE_MIN_DAYS[zoom] - calendarDayDistance(start, end);
    if (missing > 0) {
        start = getTimelineUnitStart(addCalendarDays(start, -Math.floor(missing / 2)), zoom);
        end = getTimelineNextUnit(getTimelineUnitStart(addCalendarDays(end, Math.ceil(missing / 2)), zoom), zoom);
    }
    return {start, end};
};

export const getTimelineDayCount = (range: ITimelineRange) => calendarDayDistance(range.start, range.end);

export const getTimelineWidth = (range: ITimelineRange, zoom: TTimelineZoom) =>
    getTimelineDayCount(range) * TIMELINE_DAY_WIDTH[zoom];

export const getTimelineX = (day: number, range: ITimelineRange, zoom: TTimelineZoom) =>
    calendarDayDistance(range.start, calendarDay(day)) * TIMELINE_DAY_WIDTH[zoom];

// 将时间轴内的横坐标换算为所在日期的本地零点，超出范围时取最近的端点日期。
export const getTimelineDayAt = (x: number, range: ITimelineRange, zoom: TTimelineZoom) => {
    const index = Math.floor(x / TIMELINE_DAY_WIDTH[zoom]);
    return addCalendarDays(range.start, Math.max(0, Math.min(getTimelineDayCount(range) - 1, index)));
};

export interface ITimelineBar {
    left: number;
    width: number;
    /** 开始日期在可见范围之前。 */
    clippedStart: boolean;
    /** 结束日期在可见范围之后。 */
    clippedEnd: boolean;
}

export const getTimelineBar = (span: ITimelineSpan, range: ITimelineRange, zoom: TTimelineZoom): ITimelineBar | undefined => {
    const lastDay = addCalendarDays(range.end, -1);
    if (span.endDay < range.start || span.startDay > lastDay) {
        return;
    }
    const startDay = Math.max(span.startDay, range.start);
    const endDay = Math.min(span.endDay, lastDay);
    return {
        left: getTimelineX(startDay, range, zoom),
        width: (calendarDayDistance(startDay, endDay) + 1) * TIMELINE_DAY_WIDTH[zoom],
        clippedStart: span.startDay < range.start,
        clippedEnd: span.endDay > lastDay,
    };
};

export interface ITimelineHeaderCell {
    start: number;
    days: number;
    label: string;
    title: string;
}

const getHeaderCells = (range: ITimelineRange, unit: TTimelineZoom | "year",
                        label: (start: number) => {label: string; title: string}) => {
    const cells: ITimelineHeaderCell[] = [];
    let current = range.start;
    while (current < range.end) {
        let next: number;
        if (unit === "year") {
            next = localDate(new Date(current).getFullYear() + 1, 0, 1);
        } else {
            next = getTimelineNextUnit(getTimelineUnitStart(current, unit), unit);
        }
        next = Math.min(next, range.end);
        cells.push({start: current, days: calendarDayDistance(current, next), ...label(current)});
        current = next;
    }
    return cells;
};

// 两行表头：日和周刻度上行为月份，月和季度刻度上行为年份；下行为当前刻度的单位。
export const getTimelineHeader = (range: ITimelineRange, zoom: TTimelineZoom, locale: string) => {
    const format = (value: number, options: Intl.DateTimeFormatOptions) =>
        new Date(value).toLocaleDateString(locale, options);
    const full = (value: number) => new Date(value).toLocaleDateString(locale);
    const top = zoom === "day" || zoom === "week" ?
        getHeaderCells(range, "month", start => ({label: format(start, {year: "numeric", month: "long"}), title: ""})) :
        getHeaderCells(range, "year", start => ({label: format(start, {year: "numeric"}), title: ""}));
    const bottom = getHeaderCells(range, zoom, start => {
        const date = new Date(start);
        switch (zoom) {
            case "day":
                return {label: `${date.getDate()}`, title: format(start, {weekday: "long", year: "numeric", month: "long", day: "numeric"})};
            case "week": {
                const week = getISOWeek(start);
                return {label: format(start, {month: "short", day: "numeric"}), title: `${week.year}-W${String(week.week).padStart(2, "0")} · ${full(start)}`};
            }
            case "month":
                return {label: format(start, {month: "short"}), title: format(start, {year: "numeric", month: "long"})};
            default:
                return {label: `Q${Math.floor(date.getMonth() / 3) + 1}`, title: `${date.getFullYear()} Q${Math.floor(date.getMonth() / 3) + 1}`};
        }
    });
    return {top, bottom};
};

// 将时间平移到指定日期，保留当天的时刻。
export const shiftTimelineDate = (value: number, day: number) => addCalendarDays(value, calendarDayDistance(value, day));

/**
 * 条目写回日期时使用的字段值：start 为开始字段的日期值；end 为独立结束字段的日期值，未绑定结束字段时省略。
 * 未绑定结束字段时，区间结束日期保存在开始字段的 content2。
 */
export interface ITimelineDates {
    start: IAVCellDateValue;
    end?: IAVCellDateValue;
}

const getEndDay = (dates: ITimelineDates) => {
    if (dates.end?.isNotEmpty) {
        return calendarDay(dates.end.content);
    }
    if (dates.start.hasEndDate && dates.start.isNotEmpty2) {
        return calendarDay(dates.start.content2);
    }
    return calendarDay(dates.start.content);
};

const moveDate = (date: IAVCellDateValue, days: number): IAVCellDateValue => ({
    ...date,
    content: date.isNotEmpty ? addCalendarDays(date.content, days) : date.content,
    content2: date.hasEndDate && date.isNotEmpty2 ? addCalendarDays(date.content2, days) : date.content2,
});

// 整体移动保留时长：开始、开始字段自带的结束日期和独立结束字段同时平移。
export const moveTimelineDates = (dates: ITimelineDates, days: number): ITimelineDates => ({
    start: moveDate(dates.start, days),
    end: dates.end ? (dates.end.isNotEmpty ? moveDate(dates.end, days) : dates.end) : undefined,
});

// 拖动端点只修改该端点的日期并保留时刻；结束早于开始时返回 undefined。
export const resizeTimelineDates = (dates: ITimelineDates, endpoint: "start" | "end", day: number,
                                    useEndField: boolean): ITimelineDates | undefined => {
    const start = dates.start;
    if (!start.isNotEmpty || !Number.isFinite(start.content)) {
        return;
    }
    day = calendarDay(day);
    if (endpoint === "start") {
        if (day > getEndDay(dates)) {
            return;
        }
        return {start: {...start, content: shiftTimelineDate(start.content, day)}, end: dates.end};
    }
    if (day < calendarDay(start.content)) {
        return;
    }
    if (useEndField) {
        const end = dates.end || {content: 0, isNotEmpty: false};
        const base = end.isNotEmpty ? end.content : start.content;
        return {start, end: {...end, content: shiftTimelineDate(base, day), isNotEmpty: true,
            isNotTime: end.isNotEmpty ? end.isNotTime : start.isNotTime}};
    }
    const base = start.hasEndDate && start.isNotEmpty2 ? start.content2 : start.content;
    const content2 = shiftTimelineDate(base, day);
    if (start.isNotTime && day === calendarDay(start.content)) {
        return {start: {...start, hasEndDate: false, isNotEmpty2: false, content2}, end: dates.end};
    }
    if (content2 < start.content) {
        return;
    }
    return {start: {...start, hasEndDate: true, isNotEmpty2: true, content2}, end: dates.end};
};

// 将待安排条目放到指定日期：写入全天开始日期，原有的结束日期设置保持不变。
export const scheduleTimelineDate = (date: IAVCellDateValue | undefined, day: number): IAVCellDateValue => ({
    ...(date || {}),
    content: calendarDay(day),
    isNotEmpty: true,
    isNotTime: true,
    hasEndDate: false,
    isNotEmpty2: false,
});

// 拖动创建的日期区间，起止顺序与拖动方向无关。
export const getTimelineCreateSpan = (from: number, to: number): ITimelineSpan => ({
    startDay: Math.min(calendarDay(from), calendarDay(to)),
    endDay: Math.max(calendarDay(from), calendarDay(to)),
});
