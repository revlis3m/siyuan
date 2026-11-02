import {AV_DATE_TYPES} from "../capabilities";
import {Constants} from "../../../../constants";
import dayjs from "dayjs";
import {escapeAttr, escapeHtml} from "../../../../util/escape";
import {isMobile} from "../../../../util/functions";
import {transaction} from "../../../wysiwyg/transaction";
import {renderCell} from "../cell";
import {getConditionalBackground} from "../conditionalColor";
import {getColNameByType} from "../col";
import {finishAVLocate} from "../locate";
import {createAttributeViewItem} from "../newItemTemplate";
import {openDatabaseRowByData} from "../openDatabaseRow";
import {avRender, genTabHeaderHTML, updateSearch} from "../render";
import {replaceAVContainer} from "../container";
import {renderAVRichTextElements} from "../richText";
import {bindAvSearch, captureAvSearchSelection, deferAvSearchRender} from "../search";
import {getAVData, setAVData} from "../virtualScroll";
import {Menu} from "../../../../plugin/Menu";
import {toggleMenu} from "../../../../menus/menuToggle";
import {addCalendarDays, calendarDay} from "../calendar/date";
import {
    getTimelineBar,
    getTimelineCreateSpan,
    getTimelineDayAt,
    getTimelineHeader,
    getTimelineRange,
    getTimelineWidth,
    getTimelineX,
    getTimelineZoom,
    ITimelineDates,
    ITimelineRange,
    moveTimelineDates,
    resizeTimelineDates,
    scheduleTimelineDate,
    TIMELINE_DAY_WIDTH,
    TIMELINE_ZOOMS,
    TTimelineZoom,
} from "./date";
import {
    formatTimelineSpan,
    getTimelineDateOperations,
    getTimelineDates,
    getTimelineEntries,
    getTimelineFields,
    getTimelineSettings,
    getTimelineSideColumns,
    ITimelineEntry,
} from "./model";
import {bindTimelineSettings, canEditTimeline, getTimelineSettingsHTML, getTimelineZoomLabel, setTimelineSettings} from "./settings";

const TIMELINE_NAME_WIDTH = 220;
const TIMELINE_FIELD_WIDTH = 140;
const TIMELINE_MOBILE_NAME_WIDTH = 128;

interface ITimelineState {
    /** 视图左侧显示的日期，重新渲染后保持滚动位置；首次渲染时为空，滚动到今天附近。 */
    anchorDay?: number;
    zoom?: TTimelineZoom;
    range?: ITimelineRange;
    unscheduledOpen: boolean;
}

const states = new WeakMap<Element, Map<string, ITimelineState>>();

const getTimelineState = (blockElement: Element, viewID: string) => {
    let views = states.get(blockElement);
    if (!views) {
        views = new Map();
        states.set(blockElement, views);
    }
    let state = views.get(viewID);
    if (!state) {
        state = {unscheduledOpen: false};
        views.set(viewID, state);
    }
    return state;
};

export const refreshTimelineReadonly = (protyle: IProtyle) => {
    protyle.wysiwyg.element.querySelectorAll<HTMLElement>('.av[data-av-type="timeline"]').forEach(blockElement => {
        const data = getAVData(blockElement);
        if (data?.viewType === "timeline") {
            void renderTimeline(blockElement, protyle, data);
        }
    });
};

const openTimelineItem = (protyle: IProtyle, blockElement: HTMLElement, row: IAVRow) => {
    const primary = row.cells.find(cell => cell.valueType === "block" || cell.value?.type === "block");
    if (!primary?.value) {
        return;
    }
    return openDatabaseRowByData(protyle, {
        avID: blockElement.dataset.avId, databaseBlockID: blockElement.dataset.nodeId, notebookID: protyle.notebookId,
        itemID: row.id, valueID: primary.id || primary.value.id,
        title: primary.value.block?.content || window.siyuan.languages.untitled,
        boundBlockID: primary.value.block?.id, isDetached: !!primary.value.isDetached,
        focusPrimary: true,
    });
};

const writeTimelineDates = (protyle: IProtyle, blockElement: HTMLElement, view: IAVTable, row: IAVRow,
                            next: ITimelineDates) => {
    const {doOperations, undoOperations} = getTimelineDateOperations({
        view, row, next, avID: blockElement.dataset.avId, blockID: blockElement.dataset.nodeId,
    });
    if (doOperations.length === 0) {
        return;
    }
    transaction(protyle, [...doOperations, {
        action: "doUpdateUpdated", id: blockElement.dataset.nodeId, data: dayjs().format("YYYYMMDDHHmmss"),
    }], [...undoOperations, {
        action: "doUpdateUpdated", id: blockElement.dataset.nodeId, data: blockElement.getAttribute("updated"),
    }]);
};

const getSideCellHTML = (view: IAVTable, row: IAVRow, rowIndex: number, column: IAVColumn, width?: number) => {
    const index = view.columns.findIndex(item => item.id === column.id);
    const cell = row.cells[index];
    const content = cell?.value ? renderCell(cell.value, rowIndex, view.showIcon, "calendar", column.options,
        column.dateFormat, column.renderTemplate, false) : "";
    const checkClass = column.type === "checkbox" ? (cell?.value?.checkbox?.checked ? " av__cell-check" : " av__cell-uncheck") : "";
    return `<div class="av__timeline-field${checkClass}" data-field-id="${column.id}" data-col-id="${column.id}" data-dtype="${column.type}" data-wrap="false"${width ? ` style="width:${width}px"` : ""} title="${escapeAttr(column.name)}">${content}</div>`;
};

const getSideHTML = (view: IAVTable, row: IAVRow, rowIndex: number, columns: IAVColumn[], nameWidth: number) =>
    columns.map(column => getSideCellHTML(view, row, rowIndex, column,
        column.type === "block" ? nameWidth : TIMELINE_FIELD_WIDTH)).join("");

const getBarHTML = (entry: ITimelineEntry, range: ITimelineRange, zoom: TTimelineZoom, draggable: boolean) => {
    const bar = getTimelineBar(entry.span, range, zoom);
    if (!bar) {
        return "";
    }
    const title = entry.item.title || window.siyuan.languages.untitled;
    const background = getConditionalBackground(entry.item.color);
    const tip = `${title}\n${formatTimelineSpan(entry.span, window.siyuan.config.lang)}${entry.item.invalid ? `\n${window.siyuan.languages.calendarInvalidRange}` : ""}`;
    return `<div class="av__timeline-bar${bar.clippedStart ? " av__timeline-bar--clip-start" : ""}${bar.clippedEnd ? " av__timeline-bar--clip-end" : ""}${draggable ? " av__timeline-bar--draggable" : ""}" role="button" tabindex="0" data-timeline-item="${entry.item.id}" data-id="${entry.item.id}" title="${escapeAttr(tip)}" style="left:${bar.left}px;width:${bar.width}px;${background ? `--b3-av-timeline-background:${background};` : ""}">
    ${draggable && !bar.clippedStart ? `<span class="av__timeline-resize av__timeline-resize--start" data-timeline-resize="start" title="${escapeAttr(window.siyuan.languages.calendarResizeStart)}"></span>` : ""}
    ${entry.item.invalid ? '<svg class="av__timeline-warning"><use xlink:href="#iconInfo"></use></svg>' : ""}<span class="av__timeline-bar-label">${escapeHtml(title)}</span>
    ${draggable && !bar.clippedEnd ? `<span class="av__timeline-resize av__timeline-resize--end" data-timeline-resize="end" title="${escapeAttr(window.siyuan.languages.calendarResizeEnd)}"></span>` : ""}
</div>`;
};

const getLinesHTML = (range: ITimelineRange, zoom: TTimelineZoom, header: ReturnType<typeof getTimelineHeader>) => {
    const width = TIMELINE_DAY_WIDTH[zoom];
    let html = "";
    if (zoom === "day" || zoom === "week") {
        for (let day = range.start; day < range.end; day = addCalendarDays(day, 1)) {
            const weekday = new Date(day).getDay();
            if (weekday === 0 || weekday === 6) {
                html += `<div class="av__timeline-weekend" style="left:${getTimelineX(day, range, zoom)}px;width:${width}px"></div>`;
            }
        }
    }
    header.bottom.forEach(cell => {
        html += `<div class="av__timeline-line" style="left:${getTimelineX(cell.start, range, zoom)}px"></div>`;
    });
    return html;
};

const getHeaderHTML = (header: ReturnType<typeof getTimelineHeader>, range: ITimelineRange, zoom: TTimelineZoom) => {
    const width = TIMELINE_DAY_WIDTH[zoom];
    const today = calendarDay(Date.now());
    const cells = (items: typeof header.top, bottom: boolean) => items.map(cell => {
        const isToday = bottom && zoom === "day" && cell.start === today;
        return `<div class="av__timeline-scale-cell${isToday ? " av__timeline-scale-cell--today" : ""}" style="left:${getTimelineX(cell.start, range, zoom)}px;width:${cell.days * width}px"${cell.title ? ` title="${escapeAttr(cell.title)}"` : ""}><span class="fn__ellipsis">${escapeHtml(cell.label)}</span></div>`;
    }).join("");
    return `<div class="av__timeline-scale-row">${cells(header.top, false)}</div><div class="av__timeline-scale-row">${cells(header.bottom, true)}</div>`;
};

export const renderTimeline = async (blockElement: HTMLElement, protyle: IProtyle, data: IAV, cb?: (data: IAV) => void) => {
    const search = blockElement.querySelector<HTMLElement>('[data-type="av-search"]');
    if (deferAvSearchRender(search, () => {
        // 组合输入结束后重新读取当前视图和查询，并沿用渲染请求的过期检查。
        blockElement.removeAttribute("data-render");
        void avRender(blockElement, protyle, cb);
    })) {
        return;
    }
    const view = data.view as IAVTable;
    const settings = getTimelineSettings(view);
    const zoom = getTimelineZoom(settings.zoom);
    const state = getTimelineState(blockElement, data.viewID);
    const previousScroll = blockElement.querySelector<HTMLElement>(".av__timeline-scroll");
    if (previousScroll && state.range && state.zoom) {
        // 记录当前视口左侧的日期，范围或刻度变化后按日期恢复位置。
        state.anchorDay = getTimelineDayAt(previousScroll.scrollLeft, state.range, state.zoom);
    }
    const editable = canEditTimeline(protyle);
    const query = search?.textContent || "";
    const isSearching = search === document.activeElement;
    const selection = captureAvSearchSelection(search);
    const fields = getTimelineFields(view);
    const dateEditable = editable && fields.editable;
    const mobile = isMobile();
    const nameWidth = mobile ? TIMELINE_MOBILE_NAME_WIDTH : TIMELINE_NAME_WIDTH;
    const sideColumns = getTimelineSideColumns(view, mobile);
    const sideWidth = sideColumns.reduce((width, column) => width + (column.type === "block" ? nameWidth : TIMELINE_FIELD_WIDTH), 0);
    const entries = getTimelineEntries(view);
    const entriesByID = new Map(entries.map(entry => [entry.item.id, entry]));
    const unscheduled = view.timelineUnscheduled || [];
    const unscheduledByID = new Map(unscheduled.map(row => [row.id, row]));
    const range = getTimelineRange(entries.map(entry => entry.span), zoom);
    const header = getTimelineHeader(range, zoom, window.siyuan.config.lang);
    const width = getTimelineWidth(range, zoom);
    const dayWidth = TIMELINE_DAY_WIDTH[zoom];
    const today = calendarDay(Date.now());
    const todayLeft = today >= range.start && today < range.end ? getTimelineX(today, range, zoom) + dayWidth / 2 : -1;
    const draggable = dateEditable && !mobile;

    let body = "";
    if (!fields.start) {
        body = `<div class="av__timeline-empty"><svg><use xlink:href="#iconAlignSettings"></use></svg><p>${window.siyuan.languages.timelineSelectStartField}</p>
            ${editable ? `<div class="av__timeline-setup">${getTimelineSettingsHTML(view)}</div><div class="av__timeline-create-fields">${AV_DATE_TYPES.map(type => `<button class="b3-button b3-button--outline" data-timeline-create-field="${type}">${window.siyuan.languages.newCol} ${getColNameByType(type)}</button>`).join("")}</div>` : ""}</div>`;
    } else {
        const rowsHTML = entries.map(entry => `<div class="av__timeline-row" data-id="${entry.row.id}">
    <div class="av__timeline-side" data-timeline-open="${entry.row.id}">${getSideHTML(view, entry.row, entry.rowIndex, sideColumns, nameWidth)}</div>
    <div class="av__timeline-lane" data-timeline-lane style="width:${width}px">${getBarHTML(entry, range, zoom, draggable && !entry.item.invalid)}</div>
</div>`).join("");
        const newRowHTML = dateEditable ? `<div class="av__timeline-row av__timeline-row--new">
    <div class="av__timeline-side"><div class="av__timeline-field ft__on-surface" style="width:${sideWidth}px"><svg class="av__timeline-add-icon"><use xlink:href="#iconAdd"></use></svg>${window.siyuan.languages.newRow}</div></div>
    <div class="av__timeline-lane" data-timeline-lane data-timeline-new style="width:${width}px" title="${escapeAttr(window.siyuan.languages.timelineCreateHint)}"></div>
</div>` : "";
        body = `<div class="av__timeline-scroll" data-prevent-swipe="true">
    <div class="av__timeline-grid" style="width:${sideWidth + width}px;--av-timeline-side-width:${sideWidth}px;--av-timeline-day-width:${dayWidth}px">
        <div class="av__timeline-head">
            <div class="av__timeline-side av__timeline-side--head">${sideColumns.map(column => `<div class="av__timeline-field av__timeline-field--head" style="width:${column.type === "block" ? nameWidth : TIMELINE_FIELD_WIDTH}px"><span class="fn__ellipsis">${escapeHtml(column.name)}</span></div>`).join("")}</div>
            <div class="av__timeline-scale" style="width:${width}px">${getHeaderHTML(header, range, zoom)}</div>
        </div>
        <div class="av__body av__timeline-body" data-group-id="">
            <div class="av__timeline-lines" style="width:${width}px">${getLinesHTML(range, zoom, header)}${todayLeft >= 0 ? `<div class="av__timeline-today" style="left:${todayLeft}px"></div>` : ""}</div>
            ${rowsHTML}
            ${newRowHTML}
        </div>
    </div>
</div>`;
    }
    const unscheduledHTML = fields.start && unscheduled.length > 0 ? `<div class="av__timeline-unscheduled${state.unscheduledOpen ? "" : " fn__none"}" data-timeline-unscheduled>
    ${dateEditable && !mobile ? `<div class="av__timeline-unscheduled-hint ft__on-surface">${window.siyuan.languages.timelineUnscheduledHint}</div>` : ""}
    <div class="av__timeline-unscheduled-list">${unscheduled.map((row, index) => {
        const primary = sideColumns.find(column => column.type === "block");
        return `<div class="av__timeline-chip${draggable ? " av__timeline-chip--draggable" : ""}" role="button" tabindex="0" data-timeline-unscheduled-item="${row.id}" data-id="${row.id}">${primary ? getSideCellHTML(view, row, index, primary) : ""}</div>`;
    }).join("")}</div>
</div>` : "";
    blockElement.removeAttribute(Constants.ATTRIBUTE_V_SCROLL);
    replaceAVContainer(blockElement, `<div class="av__container fn__block">
        ${genTabHeaderHTML(data, !!query || isSearching, editable, blockElement, editable)}
        <div class="av__timeline" contenteditable="false" data-zoom="${zoom}">
            <div class="av__timeline-toolbar">
                ${fields.start && unscheduled.length > 0 ? `<button type="button" class="b3-button b3-button--cancel av__timeline-unscheduled-toggle" data-timeline-action="unscheduled" aria-expanded="${state.unscheduledOpen}">
                    <svg class="av__timeline-unscheduled-arrow${state.unscheduledOpen ? " av__timeline-unscheduled-arrow--open" : ""}"><use xlink:href="#iconRight"></use></svg>${window.siyuan.languages.timelineUnscheduled}<span class="counter">${unscheduled.length}</span></button>` : ""}
                ${fields.start && !fields.editable ? `<span class="av__timeline-source ft__on-surface">${window.siyuan.languages.calendarReadOnlyDate}</span>` : ""}
                <div class="av__timeline-controls">
                    ${fields.start ? `<button type="button" class="b3-button b3-button--cancel" data-timeline-action="today">${window.siyuan.languages.calendarToday}</button>` : ""}
                    <select class="b3-select" data-timeline-zoom aria-label="${escapeAttr(window.siyuan.languages.zoom)}"${editable ? "" : " disabled"}>${TIMELINE_ZOOMS.map(item => `<option value="${item}"${item === zoom ? " selected" : ""}>${getTimelineZoomLabel(item)}</option>`).join("")}</select>
                </div>
            </div>
            ${unscheduledHTML}
            ${body}
        </div>
        <div class="av__cursor" contenteditable="true">${Constants.ZWSP}</div>
    </div>`);
    blockElement.dataset.render = "true";
    setAVData(blockElement, data);
    const root = blockElement.querySelector<HTMLElement>(".av__timeline");
    const scroller = root.querySelector<HTMLElement>(".av__timeline-scroll");
    if (scroller) {
        const anchor = state.zoom ? state.anchorDay : undefined;
        const target = anchor !== undefined ? getTimelineX(Math.max(range.start, Math.min(anchor, addCalendarDays(range.end, -1))), range, zoom) :
            Math.max(0, getTimelineX(today, range, zoom) - Math.max(0, scroller.clientWidth - sideWidth) / 3);
        scroller.scrollLeft = target;
    }
    state.zoom = zoom;
    state.range = range;

    const refresh = () => {
        blockElement.removeAttribute("data-render");
        void avRender(blockElement, protyle);
    };
    const create = (span: {startDay: number; endDay: number}) => {
        if (!dateEditable) {
            return;
        }
        createAttributeViewItem({protyle, blockElement, templateID: data.defaultTemplateID,
            position: {calendarDate: span.startDay, calendarEndDate: span.endDay}});
    };
    const laneDay = (lane: HTMLElement, clientX: number) =>
        getTimelineDayAt(clientX - lane.getBoundingClientRect().left, range, zoom);
    const findLane = (clientX: number, clientY: number) => {
        const element = document.elementFromPoint(clientX, clientY) as HTMLElement;
        return element && root.contains(element) ? element.closest<HTMLElement>("[data-timeline-lane]") : undefined;
    };

    let suppressClick = false;
    root.addEventListener("click", event => {
        if (suppressClick) {
            event.stopImmediatePropagation();
            event.preventDefault();
        }
    }, true);

    // 条形拖动、端点调整、空白处拖动新建和待安排条目拖入共用指针会话，移动超过阈值后才开始拖动。
    root.addEventListener("pointerdown", event => {
        const target = event.target as HTMLElement;
        if (event.button !== 0 || event.pointerType === "touch" || !draggable || target.closest(".av__celltext--ref[data-id]")) {
            return;
        }
        const barElement = target.closest<HTMLElement>("[data-timeline-item]");
        const chipElement = target.closest<HTMLElement>("[data-timeline-unscheduled-item]");
        const laneElement = target.closest<HTMLElement>("[data-timeline-lane]");
        const entry = barElement ? entriesByID.get(barElement.dataset.timelineItem) : undefined;
        const chipRow = chipElement ? unscheduledByID.get(chipElement.dataset.timelineUnscheduledItem) : undefined;
        if (!entry && !chipRow && !(laneElement && !barElement)) {
            return;
        }
        event.stopPropagation();
        const endpoint = target.closest<HTMLElement>("[data-timeline-resize]")?.dataset.timelineResize as "start" | "end" | undefined;
        const startX = event.clientX;
        const startY = event.clientY;
        const originDay = laneElement ? laneDay(laneElement, startX) : undefined;
        const original = barElement ? {left: barElement.offsetLeft, width: barElement.offsetWidth} : undefined;
        let dragging = false;
        let preview: HTMLElement | undefined;
        let candidate: ITimelineDates | undefined;
        let createSpan: {startDay: number; endDay: number} | undefined;
        let dropDay: number | undefined;
        const controller = new AbortController();
        const clean = () => {
            controller.abort();
            preview?.remove();
            root.classList.remove("av__timeline--dragging");
            barElement?.classList.remove("av__timeline-bar--dragging");
            if (barElement && original) {
                barElement.style.left = `${original.left}px`;
                barElement.style.width = `${original.width}px`;
            }
            if (dragging) {
                suppressClick = true;
                setTimeout(() => {
                    suppressClick = false;
                });
            }
        };
        const showPreview = (lane: HTMLElement, startDay: number, endDay: number) => {
            if (!preview) {
                preview = document.createElement("div");
                preview.className = "av__timeline-bar av__timeline-bar--preview";
                preview.setAttribute("aria-hidden", "true");
            }
            if (preview.parentElement !== lane) {
                lane.append(preview);
            }
            const bar = getTimelineBar({startDay, endDay}, range, zoom);
            preview.style.left = `${bar?.left || 0}px`;
            preview.style.width = `${bar?.width || 0}px`;
            preview.title = formatTimelineSpan({startDay, endDay}, window.siyuan.config.lang);
        };
        const move = (clientX: number, clientY: number) => {
            if (entry && barElement && original) {
                const dates = getTimelineDates(view, entry.row);
                if (!dates) {
                    return;
                }
                if (endpoint) {
                    candidate = resizeTimelineDates(dates, endpoint, laneDay(laneElement, clientX), fields.useEndField);
                } else {
                    candidate = moveTimelineDates(dates, Math.round((clientX - startX) / dayWidth));
                }
                root.classList.toggle("av__timeline--invalid", !candidate);
                if (!candidate) {
                    return;
                }
                const next = {...entry.item, start: candidate.start.content,
                    end: candidate.end?.isNotEmpty ? candidate.end.content : candidate.start.hasEndDate && candidate.start.isNotEmpty2 ? candidate.start.content2 : candidate.start.content};
                const bar = getTimelineBar(getTimelineCreateSpan(calendarDay(next.start), calendarDay(Math.max(next.start, next.end))), range, zoom);
                if (bar) {
                    barElement.style.left = `${bar.left}px`;
                    barElement.style.width = `${bar.width}px`;
                }
                return;
            }
            const lane = findLane(clientX, clientY);
            if (chipRow) {
                dropDay = lane ? laneDay(lane, clientX) : undefined;
                if (lane && dropDay !== undefined) {
                    showPreview(lane, dropDay, dropDay);
                } else {
                    preview?.remove();
                }
                return;
            }
            if (laneElement && originDay !== undefined) {
                createSpan = getTimelineCreateSpan(originDay, laneDay(laneElement, clientX));
                showPreview(laneElement, createSpan.startDay, createSpan.endDay);
            }
        };
        document.addEventListener("pointermove", moveEvent => {
            if (moveEvent.pointerId !== event.pointerId) {
                return;
            }
            if (!root.isConnected) {
                clean();
                return;
            }
            if (!dragging && Math.hypot(moveEvent.clientX - startX, moveEvent.clientY - startY) < 5) {
                return;
            }
            moveEvent.preventDefault();
            if (!dragging) {
                dragging = true;
                window.siyuan.menus.menu.remove();
                root.classList.add("av__timeline--dragging");
                barElement?.classList.add("av__timeline-bar--dragging");
            }
            move(moveEvent.clientX, moveEvent.clientY);
        }, {signal: controller.signal, passive: false});
        document.addEventListener("pointerup", upEvent => {
            if (upEvent.pointerId !== event.pointerId) {
                return;
            }
            const wasDragging = dragging;
            const day = dropDay;
            const span = createSpan;
            const next = candidate;
            clean();
            if (!root.isConnected) {
                return;
            }
            if (!wasDragging) {
                // 在空白处单击新建当天的条目；单击条形和待安排条目由点击事件打开条目。
                if (!barElement && !chipElement && laneElement && originDay !== undefined) {
                    suppressClick = true;
                    setTimeout(() => {
                        suppressClick = false;
                    });
                    create({startDay: originDay, endDay: originDay});
                }
                return;
            }
            if (entry && next) {
                writeTimelineDates(protyle, blockElement, view, entry.row, next);
            } else if (chipRow && day !== undefined) {
                const dates = getTimelineDates(view, chipRow);
                if (dates) {
                    writeTimelineDates(protyle, blockElement, view, chipRow, {...dates, start: scheduleTimelineDate(dates.start, day)});
                }
            } else if (span) {
                create(span);
            }
        }, {signal: controller.signal});
        document.addEventListener("pointercancel", cancelEvent => {
            if (cancelEvent.pointerId === event.pointerId) {
                clean();
            }
        }, {signal: controller.signal});
        window.addEventListener("blur", clean, {signal: controller.signal});
        document.addEventListener("keydown", keyEvent => {
            if (keyEvent.key === "Escape") {
                keyEvent.preventDefault();
                dragging = false;
                candidate = undefined;
                createSpan = undefined;
                dropDay = undefined;
                clean();
            }
        }, {signal: controller.signal, capture: true});
    });

    const openRow = (rowID: string) => {
        const row = entriesByID.get(rowID)?.row || unscheduledByID.get(rowID);
        if (row) {
            void openTimelineItem(protyle, blockElement, row);
        }
    };
    root.addEventListener("click", event => {
        const target = event.target as HTMLElement;
        // 主键引用交给编辑器处理，保留普通跳转、分屏修饰键和移动端的引用交互。
        if (target.closest(".av__celltext--ref[data-id]")) {
            return;
        }
        event.stopPropagation();
        window.siyuan.menus.menu.remove();
        const item = target.closest<HTMLElement>("[data-timeline-item], [data-timeline-unscheduled-item], [data-timeline-open]");
        if (item) {
            openRow(item.dataset.timelineItem || item.dataset.timelineUnscheduledItem || item.dataset.timelineOpen);
            return;
        }
        const lane = target.closest<HTMLElement>("[data-timeline-lane]");
        if (lane) {
            // 鼠标在空白处的单击和拖动由指针会话处理；触屏轻点新建行创建当天的条目，轻点条目所在行打开条目。
            if (lane.hasAttribute("data-timeline-new")) {
                if (dateEditable) {
                    const day = laneDay(lane, event.clientX);
                    create({startDay: day, endDay: day});
                }
            } else {
                openRow(lane.closest<HTMLElement>(".av__timeline-row")?.dataset.id);
            }
            return;
        }
        const action = target.closest<HTMLElement>("[data-timeline-action]")?.dataset.timelineAction;
        if (action === "today" && scroller) {
            scroller.scrollLeft = Math.max(0, getTimelineX(today, range, zoom) - Math.max(0, scroller.clientWidth - sideWidth) / 3);
        } else if (action === "unscheduled") {
            state.unscheduledOpen = !state.unscheduledOpen;
            root.querySelector("[data-timeline-unscheduled]")?.classList.toggle("fn__none", !state.unscheduledOpen);
            root.querySelector(".av__timeline-unscheduled-arrow")?.classList.toggle("av__timeline-unscheduled-arrow--open", state.unscheduledOpen);
            target.closest("[data-timeline-action]")?.setAttribute("aria-expanded", String(state.unscheduledOpen));
        }
        const fieldButton = target.closest<HTMLElement>("[data-timeline-create-field]");
        if (fieldButton && editable) {
            addTimelineDateField(protyle, blockElement, data, fieldButton.dataset.timelineCreateField as TAVCol);
        }
    });
    root.addEventListener("keydown", event => {
        event.stopPropagation();
        const item = (event.target as HTMLElement).closest<HTMLElement>("[data-timeline-item], [data-timeline-unscheduled-item]");
        if (item && (event.key === "Enter" || event.key === " ")) {
            event.preventDefault();
            openRow(item.dataset.timelineItem || item.dataset.timelineUnscheduledItem);
        }
    });
    root.addEventListener("contextmenu", event => {
        const item = (event.target as HTMLElement).closest<HTMLElement>("[data-timeline-item], [data-timeline-unscheduled-item], [data-timeline-open]");
        event.preventDefault();
        event.stopPropagation();
        if (!item) {
            return;
        }
        const rowID = item.dataset.timelineItem || item.dataset.timelineUnscheduledItem || item.dataset.timelineOpen;
        const row = entriesByID.get(rowID)?.row || unscheduledByID.get(rowID);
        if (!row) {
            return;
        }
        toggleMenu({target: item, toggle: false, build: () => {
            const menu = new Menu();
            menu.addItem({icon: "iconOpen", label: window.siyuan.languages.openBy, click: () => {
                void openTimelineItem(protyle, blockElement, row);
            }});
            if (dateEditable && entriesByID.has(rowID)) {
                menu.addItem({icon: "iconTrashcan", label: window.siyuan.languages.calendarClearDate, click: () => {
                    const dates = getTimelineDates(view, row);
                    if (dates) {
                        writeTimelineDates(protyle, blockElement, view, row, {
                            start: {...dates.start, isNotEmpty: false, isNotEmpty2: false, hasEndDate: false},
                            end: dates.end ? {...dates.end, isNotEmpty: false} : undefined,
                        });
                    }
                }});
            }
            menu.open({x: event.clientX, y: event.clientY});
        }});
    });
    root.querySelector<HTMLSelectElement>("[data-timeline-zoom]")?.addEventListener("change", event => {
        const value = getTimelineZoom((event.target as HTMLSelectElement).value);
        if (scroller) {
            // 切换刻度前记录视口中央的日期，切换后保持该日期仍在中央附近。
            const center = getTimelineDayAt(scroller.scrollLeft + Math.max(0, scroller.clientWidth - sideWidth) / 2, range, zoom);
            state.anchorDay = addCalendarDays(center, -Math.round(Math.max(0, scroller.clientWidth - sideWidth) / 2 / TIMELINE_DAY_WIDTH[value]));
        }
        if (setTimelineSettings(protyle, blockElement, data, {...settings, zoom: value})) {
            void renderTimeline(blockElement, protyle, data, cb);
        }
    });
    if (!fields.start && editable) {
        bindTimelineSettings({protyle, blockElement, data, menuElement: root, onChange: refresh});
    }
    bindAvSearch({blockElement, query, isSearching, selection, onChange: () => updateSearch(blockElement, protyle)});
    renderAVRichTextElements(blockElement);
    finishAVLocate(blockElement, protyle, data);
    cb?.(data);
};

export const addTimelineDateField = (protyle: IProtyle, blockElement: HTMLElement, data: IAV, type: TAVCol) => {
    if (!canEditTimeline(protyle)) {
        return;
    }
    const id = Lute.NewNodeID();
    const context = {avID: data.id, blockID: blockElement.dataset.nodeId, viewID: data.viewID};
    const settings = getTimelineSettings(data.view as IAVTable);
    transaction(protyle, [{
        ...context, action: "addAttrViewCol", id, type, name: getColNameByType(type),
    }, {
        ...context, action: "setAttrViewTimeline", data: {...settings, startKeyID: id},
    }], [{
        ...context, action: "setAttrViewTimeline", data: settings,
    }, {
        ...context, action: "removeAttrViewCol", id,
    }]);
};
