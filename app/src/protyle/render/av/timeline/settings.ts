import {isAVSelectType} from "../capabilities";
import {transaction} from "../../../wysiwyg/transaction";
import {escapeAttr, escapeHtml} from "../../../../util/escape";
import {Menu} from "../../../../plugin/Menu";
import {openViewSettingMenu} from "../viewSettingMenu";
import {getTimelineZoom, TIMELINE_ZOOMS, TTimelineZoom} from "./date";
import {getTimelineSettings, isTimelineDateColumn, TIMELINE_COLOR_CONDITIONAL} from "./model";

export const getTimelineZoomLabel = (zoom: TTimelineZoom) => ({
    day: window.siyuan.languages.timelineZoomDay,
    week: window.siyuan.languages.timelineZoomWeek,
    month: window.siyuan.languages.timelineZoomMonth,
    quarter: window.siyuan.languages.timelineZoomQuarter,
})[zoom];

type TTimelineSettingKey = keyof IAVTimelineSettings;

interface ITimelineSettingItem {
    key: TTimelineSettingKey;
    label: string;
    choices: Array<{value: string; label: string}>;
}

export const getTimelineSettingItems = (view: IAVTable): ITimelineSettingItem[] => {
    const settings = getTimelineSettings(view);
    const none = {value: "", label: window.siyuan.languages.calcOperatorNone};
    const fields = (columns: IAVColumn[]) => columns.map(column => ({value: column.id, label: column.name}));
    return [{
        key: "startKeyID", label: window.siyuan.languages.timelineStartField,
        choices: [none, ...fields(view.columns.filter(isTimelineDateColumn))],
    }, {
        key: "endKeyID", label: window.siyuan.languages.timelineEndField,
        choices: [{value: "", label: window.siyuan.languages.timelineEndFromStart},
            ...fields(view.columns.filter(column => isTimelineDateColumn(column) && column.id !== settings.startKeyID))],
    }, {
        key: "zoom", label: window.siyuan.languages.zoom,
        choices: TIMELINE_ZOOMS.map(zoom => ({value: zoom, label: getTimelineZoomLabel(zoom)})),
    }, {
        key: "colorBy", label: window.siyuan.languages.calendarColorField,
        choices: [none, ...fields(view.columns.filter(column => isAVSelectType(column.type))),
            {value: TIMELINE_COLOR_CONDITIONAL, label: window.siyuan.languages.conditionalColors}],
    }];
};

// 结束字段与开始字段相同时没有意义，切换开始字段后同步清空结束字段。
export const updateTimelineSettings = (settings: IAVTimelineSettings, key: TTimelineSettingKey, value: string): IAVTimelineSettings => {
    const next = {...settings, [key]: key === "zoom" ? getTimelineZoom(value) : value};
    if (key === "startKeyID" && next.endKeyID === value) {
        next.endKeyID = "";
    }
    return next;
};

export const getTimelineSettingsHTML = (view: IAVTable, asMenu = false) => {
    const settings = getTimelineSettings(view);
    return getTimelineSettingItems(view).map(item => {
        const value = settings[item.key];
        const selected = item.choices.find(choice => choice.value === value);
        if (asMenu) {
            return `<button class="b3-menu__item" data-timeline-setting="${item.key}">
    <span class="fn__flex-center">${item.label}</span><span class="fn__flex-1"></span>
    <span class="b3-menu__accelerator">${escapeHtml(selected?.label || window.siyuan.languages.calcOperatorNone)}</span>
    <svg class="b3-menu__icon b3-menu__icon--small"><use xlink:href="#iconRight"></use></svg>
</button>`;
        }
        return `<label class="av__timeline-setting"><span>${item.label}</span>
    <select class="b3-select" data-timeline-setting="${item.key}" aria-label="${escapeAttr(item.label)}">
        ${item.choices.map(choice => `<option value="${escapeAttr(choice.value)}"${choice.value === value ? " selected" : ""}>${escapeHtml(choice.label)}</option>`).join("")}
    </select>
</label>`;
    }).join("");
};

export const canEditTimeline = (protyle: IProtyle) => !protyle.disabled && !window.siyuan.isPublish &&
    !protyle.options.history?.created && !protyle.options.history?.snapshot;

export const setTimelineSettings = (protyle: IProtyle, blockElement: Element, data: IAV, next: IAVTimelineSettings) => {
    const view = data.view as IAVTable;
    const previous = getTimelineSettings(view);
    if (JSON.stringify(previous) === JSON.stringify(next)) {
        return false;
    }
    const operation = {
        action: "setAttrViewTimeline" as const,
        avID: data.id,
        blockID: blockElement.getAttribute("data-node-id"),
        viewID: data.viewID,
    };
    transaction(protyle, [{...operation, data: next}], [{...operation, data: previous}]);
    view.timeline = next;
    return true;
};

export const bindTimelineSettings = (options: {
    protyle: IProtyle;
    blockElement: Element;
    data: IAV;
    menuElement: Element;
    onChange?: () => void;
}) => {
    const controls = options.menuElement.querySelectorAll<HTMLSelectElement | HTMLButtonElement>("[data-timeline-setting]");
    if (!canEditTimeline(options.protyle)) {
        controls.forEach(control => {
            control.disabled = true;
        });
        return;
    }
    controls.forEach(control => {
        const key = control.dataset.timelineSetting as TTimelineSettingKey;
        const update = (value: string) => {
            const view = options.data.view as IAVTable;
            if (setTimelineSettings(options.protyle, options.blockElement, options.data,
                updateTimelineSettings(getTimelineSettings(view), key, value))) {
                options.onChange?.();
            }
        };
        if (control.tagName === "BUTTON") {
            control.addEventListener("click", (event) => {
                const view = options.data.view as IAVTable;
                const item = getTimelineSettingItems(view).find(current => current.key === key);
                const current = getTimelineSettings(view)[key];
                const menu = new Menu();
                item.choices.forEach(choice => menu.addItem({
                    iconHTML: "", label: escapeHtml(choice.label),
                    checked: current === choice.value,
                    click: () => update(choice.value),
                }));
                openViewSettingMenu(menu, control as HTMLElement);
                event.preventDefault();
                event.stopPropagation();
            });
        } else {
            control.addEventListener("change", () => update((control as HTMLSelectElement).value));
        }
    });
};
