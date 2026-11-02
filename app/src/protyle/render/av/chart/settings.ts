import {isAVDateType} from "../capabilities";
import {transaction} from "../../../wysiwyg/transaction";
import {escapeAttr, escapeHtml} from "../../../../util/escape";
import {Menu} from "../../../../plugin/Menu";
import {openViewSettingMenu} from "../viewSettingMenu";
import {
    CHART_DATE_BUCKETS,
    CHART_HEIGHTS,
    CHART_MEASURES,
    CHART_SORTS,
    CHART_TYPES,
    getChartSettings,
    IChartLabels,
    isChartGroupColumn,
    isChartStackColumn,
    isChartValueColumn,
    normalizeChartHeight,
    TChartDateBucket,
    TChartMeasure,
    TChartSort,
    TChartType,
} from "./model";

type TChartChoiceKey = "chartType" | "xKeyID" | "dateBucket" | "measure" | "valueKeyID" | "stackKeyID" | "sort" | "height";
type TChartToggleKey = "showEmptyGroups" | "showLegend" | "showDataLabels";

export interface IChartSettingItem {
    key: TChartChoiceKey;
    label: string;
    choices: Array<{value: string; label: string}>;
}

export interface IChartToggleItem {
    key: TChartToggleKey;
    label: string;
}

export const getChartTypeLabel = (type: TChartType) => ({
    bar: window.siyuan.languages.chartTypeBar,
    horizontalBar: window.siyuan.languages.chartTypeHorizontalBar,
    line: window.siyuan.languages.chartTypeLine,
    donut: window.siyuan.languages.chartTypeDonut,
    number: window.siyuan.languages.chartTypeNumber,
})[type];

export const getChartLabels = (): IChartLabels => ({
    lang: window.siyuan.config.lang,
    empty: window.siyuan.languages.empty,
    checked: window.siyuan.languages.checked,
    unchecked: window.siyuan.languages.unchecked,
    weekOf: window.siyuan.languages.chartWeekOf,
    measures: {
        count: window.siyuan.languages.chartMeasureCount,
        sum: window.siyuan.languages.calcOperatorSum,
        average: window.siyuan.languages.calcOperatorAverage,
        min: window.siyuan.languages.calcOperatorMin,
        max: window.siyuan.languages.calcOperatorMax,
    },
});

const getBucketLabel = (bucket: TChartDateBucket) => ({
    day: window.siyuan.languages.timelineZoomDay,
    week: window.siyuan.languages.timelineZoomWeek,
    month: window.siyuan.languages.timelineZoomMonth,
    quarter: window.siyuan.languages.timelineZoomQuarter,
    year: window.siyuan.languages.chartBucketYear,
})[bucket];

const getSortLabel = (sort: TChartSort) => ({
    default: window.siyuan.languages.default,
    valueDesc: window.siyuan.languages.chartSortValueDesc,
    valueAsc: window.siyuan.languages.chartSortValueAsc,
    labelAsc: window.siyuan.languages.chartSortLabelAsc,
})[sort];

/** 按当前图形类型和字段返回可用的设置项，与当前设置无关的项不显示。 */
export const getChartSettingItems = (view: IAVTable): IChartSettingItem[] => {
    const settings = getChartSettings(view);
    const none = {value: "", label: window.siyuan.languages.calcOperatorNone};
    const fields = (columns: IAVColumn[]) => columns.map(column => ({value: column.id, label: column.name}));
    const xColumn = view.columns.find(column => column.id === settings.xKeyID);
    const measures = getChartLabels().measures;
    const items: IChartSettingItem[] = [{
        key: "chartType", label: window.siyuan.languages.chartType,
        choices: CHART_TYPES.map(type => ({value: type, label: getChartTypeLabel(type)})),
    }];
    if (settings.chartType !== "number") {
        items.push({
            key: "xKeyID", label: window.siyuan.languages.chartGroupBy,
            choices: [none, ...fields(view.columns.filter(isChartGroupColumn))],
        });
        if (isAVDateType(xColumn?.type)) {
            items.push({
                key: "dateBucket", label: window.siyuan.languages.chartDateBucket,
                choices: CHART_DATE_BUCKETS.map(bucket => ({value: bucket, label: getBucketLabel(bucket)})),
            });
        }
    }
    items.push({
        key: "measure", label: window.siyuan.languages.chartMeasure,
        choices: CHART_MEASURES.map(measure => ({value: measure, label: measures[measure as TChartMeasure]})),
    });
    if (settings.measure !== "count") {
        items.push({
            key: "valueKeyID", label: window.siyuan.languages.chartValueField,
            choices: [none, ...fields(view.columns.filter(isChartValueColumn))],
        });
    }
    if (settings.chartType !== "number" && settings.chartType !== "donut") {
        items.push({
            key: "stackKeyID", label: window.siyuan.languages.chartStackBy,
            choices: [none, ...fields(view.columns.filter(column => isChartStackColumn(column) && column.id !== settings.xKeyID))],
        });
    }
    if (settings.chartType !== "number") {
        items.push({
            key: "sort", label: window.siyuan.languages.sort,
            choices: CHART_SORTS.map(sort => ({value: sort, label: getSortLabel(sort)})),
        });
    }
    items.push({
        key: "height", label: window.siyuan.languages.height,
        choices: CHART_HEIGHTS.map(height => ({value: height.toString(), label: `${height}px`})),
    });
    return items;
};

export const getChartToggleItems = (view: IAVTable): IChartToggleItem[] => {
    const type = getChartSettings(view).chartType;
    if (type === "number") {
        return [];
    }
    return [
        {key: "showEmptyGroups", label: window.siyuan.languages.chartShowEmptyGroups},
        {key: "showLegend", label: window.siyuan.languages.chartShowLegend},
        {key: "showDataLabels", label: window.siyuan.languages.chartShowDataLabels},
    ];
};

const getSettingValue = (settings: IAVChartSettings, key: TChartChoiceKey) =>
    key === "height" ? normalizeChartHeight(settings.height).toString() : settings[key];

/** 更新单个设置项；分组字段与子分组字段不能相同，切换分组字段后清空相同的子分组字段。 */
export const updateChartSettings = (settings: IAVChartSettings, key: TChartChoiceKey | TChartToggleKey,
                                    value: string | boolean): IAVChartSettings => {
    const next = {...settings};
    switch (key) {
        case "height":
            next.height = normalizeChartHeight(Number(value));
            break;
        case "showEmptyGroups":
        case "showLegend":
        case "showDataLabels":
            next[key] = !!value;
            break;
        case "chartType":
            next.chartType = CHART_TYPES.includes(value as TChartType) ? value as TChartType : "bar";
            break;
        case "dateBucket":
            next.dateBucket = CHART_DATE_BUCKETS.includes(value as TChartDateBucket) ? value as TChartDateBucket : "month";
            break;
        case "measure":
            next.measure = CHART_MEASURES.includes(value as TChartMeasure) ? value as TChartMeasure : "count";
            break;
        case "sort":
            next.sort = CHART_SORTS.includes(value as TChartSort) ? value as TChartSort : "default";
            break;
        default:
            next[key] = String(value);
    }
    if (key === "xKeyID" && next.stackKeyID === next.xKeyID) {
        next.stackKeyID = "";
    }
    return next;
};

export const getChartSettingsHTML = (view: IAVTable, asMenu = false) => {
    const settings = getChartSettings(view);
    const choices = getChartSettingItems(view).map(item => {
        const value = getSettingValue(settings, item.key);
        const selected = item.choices.find(choice => choice.value === value);
        if (asMenu) {
            return `<button class="b3-menu__item" data-chart-setting="${item.key}">
    <span class="fn__flex-center">${item.label}</span><span class="fn__flex-1"></span>
    <span class="b3-menu__accelerator">${escapeHtml(selected?.label || window.siyuan.languages.calcOperatorNone)}</span>
    <svg class="b3-menu__icon b3-menu__icon--small"><use xlink:href="#iconRight"></use></svg>
</button>`;
        }
        return `<label class="av__chart-setting"><span>${item.label}</span>
    <select class="b3-select" data-chart-setting="${item.key}" aria-label="${escapeAttr(item.label)}">
        ${item.choices.map(choice => `<option value="${escapeAttr(choice.value)}"${choice.value === value ? " selected" : ""}>${escapeHtml(choice.label)}</option>`).join("")}
    </select>
</label>`;
    }).join("");
    if (!asMenu) {
        return choices;
    }
    return choices + getChartToggleItems(view).map(item => `<label class="b3-menu__item">
    <span class="fn__flex-center">${item.label}</span>
    <span class="fn__space fn__flex-1"></span>
    <input data-chart-toggle="${item.key}" type="checkbox" class="b3-switch b3-switch--menu"${settings[item.key] ? " checked" : ""}>
</label>`).join("");
};

export const canEditChart = (protyle: IProtyle) => !protyle.disabled && !window.siyuan.isPublish &&
    !protyle.options.history?.created && !protyle.options.history?.snapshot;

export const setChartSettings = (protyle: IProtyle, blockElement: Element, data: IAV, next: IAVChartSettings) => {
    const view = data.view as IAVTable;
    const previous = getChartSettings(view);
    if (JSON.stringify(previous) === JSON.stringify(next)) {
        return false;
    }
    const operation = {
        action: "setAttrViewChart" as const,
        avID: data.id,
        blockID: blockElement.getAttribute("data-node-id"),
        viewID: data.viewID,
    };
    transaction(protyle, [{...operation, data: next}], [{...operation, data: previous}]);
    view.chart = next;
    return true;
};

export const bindChartSettings = (options: {
    protyle: IProtyle;
    blockElement: Element;
    data: IAV;
    menuElement: Element;
    onChange?: () => void;
}) => {
    const controls = options.menuElement.querySelectorAll<HTMLSelectElement | HTMLButtonElement | HTMLInputElement>(
        "[data-chart-setting], [data-chart-toggle]");
    if (!canEditChart(options.protyle)) {
        controls.forEach(control => {
            control.disabled = true;
        });
        return;
    }
    const update = (key: TChartChoiceKey | TChartToggleKey, value: string | boolean) => {
        const view = options.data.view as IAVTable;
        if (setChartSettings(options.protyle, options.blockElement, options.data,
            updateChartSettings(getChartSettings(view), key, value))) {
            options.onChange?.();
        }
    };
    controls.forEach(control => {
        if (control.dataset.chartToggle) {
            const key = control.dataset.chartToggle as TChartToggleKey;
            control.addEventListener("change", () => update(key, (control as HTMLInputElement).checked));
            return;
        }
        const key = control.dataset.chartSetting as TChartChoiceKey;
        if (control.tagName === "BUTTON") {
            control.addEventListener("click", (event) => {
                const view = options.data.view as IAVTable;
                const item = getChartSettingItems(view).find(current => current.key === key);
                if (!item) {
                    return;
                }
                const current = getSettingValue(getChartSettings(view), key);
                const menu = new Menu();
                item.choices.forEach(choice => menu.addItem({
                    iconHTML: "", label: escapeHtml(choice.label),
                    checked: current === choice.value,
                    click: () => update(key, choice.value),
                }));
                openViewSettingMenu(menu, control as HTMLElement);
                event.preventDefault();
                event.stopPropagation();
            });
        } else {
            control.addEventListener("change", () => update(key, (control as HTMLSelectElement).value));
        }
    });
};
