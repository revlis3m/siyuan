import {isAVDateType} from "../capabilities";
import {isCalendarDateEditable} from "../calendar/date";
import {getTimelineSpan, getTimelineZoom, ITimelineDates, ITimelineSpan} from "./date";

export const TIMELINE_COLOR_CONDITIONAL = "conditional";

export const isTimelineDateColumn = (column: IAVColumn) => isAVDateType(column?.type);

// 未返回设置时按默认的周刻度处理，保证切换布局后的首次渲染可用。
export const getTimelineSettings = (view: IAVTable): IAVTimelineSettings => ({
    startKeyID: view.timeline?.startKeyID || "",
    endKeyID: view.timeline?.endKeyID || "",
    zoom: getTimelineZoom(view.timeline?.zoom),
    colorBy: view.timeline?.colorBy || "",
});

export interface ITimelineEntry {
    item: IAVTimelineItem;
    row: IAVRow;
    rowIndex: number;
    span: ITimelineSpan;
}

// 按内核返回的顺序配对条目和区间，缺少区间的条目不显示在时间轴上。
export const getTimelineEntries = (view: IAVTable): ITimelineEntry[] => {
    const rows = new Map((view.rows || []).map((row, rowIndex) => [row.id, {row, rowIndex}]));
    const entries: ITimelineEntry[] = [];
    (view.timelineItems || []).forEach(item => {
        const source = rows.get(item?.id);
        if (source) {
            entries.push({item, row: source.row, rowIndex: source.rowIndex, span: getTimelineSpan(item)});
        }
    });
    return entries;
};

// 左侧列显示主键和可见字段，主键始终在最前；移动端只显示主键。
export const getTimelineSideColumns = (view: IAVTable, primaryOnly = false) => {
    const primary = view.columns.find(column => column.type === "block");
    const others = primaryOnly ? [] : view.columns.filter(column => column.type !== "block" && !column.hidden);
    return primary ? [primary, ...others] : others;
};

export interface ITimelineFields {
    start?: IAVColumn;
    end?: IAVColumn;
    /** 开始字段为普通日期字段，且绑定的结束字段也可编辑。 */
    editable: boolean;
    /** 区间结束写入独立结束字段，否则写入开始字段的结束日期。 */
    useEndField: boolean;
}

export const getTimelineFields = (view: IAVTable): ITimelineFields => {
    const settings = getTimelineSettings(view);
    const start = view.columns.find(column => column.id === settings.startKeyID && isTimelineDateColumn(column));
    const end = settings.endKeyID && settings.endKeyID !== settings.startKeyID ?
        view.columns.find(column => column.id === settings.endKeyID && isTimelineDateColumn(column)) : undefined;
    return {
        start, end,
        editable: !!start && isCalendarDateEditable(start) && (!end || isCalendarDateEditable(end)),
        useEndField: !!end,
    };
};

export const getTimelineCell = (view: IAVTable, row: IAVRow, column?: IAVColumn) => {
    if (!column) {
        return;
    }
    const index = view.columns.findIndex(item => item.id === column.id);
    return (index >= 0 ? row.cells[index] : undefined) || row.cells.find(cell => cell.value?.keyID === column.id);
};

export const getTimelineDates = (view: IAVTable, row: IAVRow): ITimelineDates | undefined => {
    const fields = getTimelineFields(view);
    const start = getTimelineCell(view, row, fields.start);
    if (!start) {
        return;
    }
    const dates: ITimelineDates = {start: {...(start.value?.date || {})}};
    if (fields.end) {
        dates.end = {...(getTimelineCell(view, row, fields.end)?.value?.date || {})};
    }
    return dates;
};

const isSameDate = (a: IAVCellDateValue | undefined, b: IAVCellDateValue | undefined) =>
    JSON.stringify(a || {}) === JSON.stringify(b || {});

// 生成写回日期的事务：只更新发生变化的字段，撤销恢复原值。
export const getTimelineDateOperations = (options: {
    view: IAVTable;
    row: IAVRow;
    avID: string;
    blockID: string;
    next: ITimelineDates;
}) => {
    const fields = getTimelineFields(options.view);
    const previous = getTimelineDates(options.view, options.row);
    const doOperations: IOperation[] = [];
    const undoOperations: IOperation[] = [];
    if (!previous) {
        return {doOperations, undoOperations};
    }
    const add = (column: IAVColumn | undefined, before: IAVCellDateValue | undefined, after: IAVCellDateValue | undefined) => {
        if (!column || !after || isSameDate(before, after)) {
            return;
        }
        const cell = getTimelineCell(options.view, options.row, column);
        const operation = {
            action: "updateAttrViewCell" as const, avID: options.avID, blockID: options.blockID,
            id: cell?.value?.id || cell?.id, keyID: column.id, rowID: options.row.id,
        };
        doOperations.push({...operation, data: {type: "date", date: after}});
        undoOperations.push({...operation, data: {type: "date", date: before || {}}});
    };
    add(fields.start, previous.start, options.next.start);
    add(fields.end, previous.end, options.next.end);
    return {doOperations, undoOperations};
};

export const formatTimelineSpan = (span: ITimelineSpan, locale: string) => {
    const start = new Date(span.startDay).toLocaleDateString(locale);
    return span.startDay === span.endDay ? start : `${start} - ${new Date(span.endDay).toLocaleDateString(locale)}`;
};
