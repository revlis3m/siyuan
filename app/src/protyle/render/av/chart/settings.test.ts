import * as capabilities from "../capabilities";
import * as model from "./model";
import * as assert from "node:assert/strict";
import {test} from "node:test";
import {join} from "node:path";
import {ScriptTarget} from "typescript";
import {loadModule} from "../../../../testSupport/loadModule";

type TMenuItem = {label: string; checked: boolean; click: () => void};

const languages = {
    calcOperatorNone: "None", chartType: "Chart type", chartTypeBar: "Vertical bar", chartTypeHorizontalBar: "Horizontal bar",
    chartTypeLine: "Line", chartTypeDonut: "Donut", chartTypeNumber: "Number", chartGroupBy: "Group by",
    chartDateBucket: "Date grouping", chartBucketYear: "Year", chartMeasure: "Value", chartMeasureCount: "Count",
    calcOperatorSum: "Sum", calcOperatorAverage: "Average", calcOperatorMin: "Min", calcOperatorMax: "Max",
    chartValueField: "Value field", chartStackBy: "Sub-group by", sort: "Sort", default: "Default",
    chartSortValueDesc: "Value high to low", chartSortValueAsc: "Value low to high", chartSortLabelAsc: "Name A to Z",
    height: "Height", chartShowEmptyGroups: "Show empty groups", chartShowLegend: "Show legend",
    chartShowDataLabels: "Show data labels", timelineZoomDay: "Day", timelineZoomWeek: "Week", timelineZoomMonth: "Month",
    timelineZoomQuarter: "Quarter", empty: "Empty", checked: "Checked", unchecked: "Unchecked", chartWeekOf: "Week of ${x}",
};

const setup = () => {
    const transactions: Array<{do: IOperation[], undo: IOperation[]}> = [];
    const menus: TMenuItem[][] = [];
    const window = {siyuan: {isPublish: false, config: {lang: "en_US"}, languages}};
    const methods = loadModule<typeof import("./settings")>(join(__dirname, "settings.ts"), {
        window,
        require: (id: string) => ({
            "../capabilities": capabilities,
            "./model": model,
            "../../../wysiwyg/transaction": {transaction: (_protyle: IProtyle, perform: IOperation[], undo: IOperation[]) =>
                transactions.push({do: perform, undo})},
            "../../../../util/escape": {escapeAttr: (value: string) => value, escapeHtml: (value: string) => value},
            "../viewSettingMenu": {openViewSettingMenu: (menu: {open: () => void}) => menu.open()},
            "../../../../plugin/Menu": {Menu: class {
                items: TMenuItem[] = [];
                constructor() { menus.push(this.items); }
                addItem(item: TMenuItem) { this.items.push(item); }
                open() { return; }
            }},
        })[id],
    }, ScriptTarget.ES2022);
    return {methods, transactions, window, menus};
};

const columns = [{id: "name", type: "block", name: "Name"}, {id: "status", type: "status", name: "Status"},
    {id: "type", type: "select", name: "Type"}, {id: "done", type: "checkbox", name: "Done"},
    {id: "tags", type: "mSelect", name: "Tags"}, {id: "points", type: "number", name: "Points"},
    {id: "score", type: "rollup", name: "Score"}, {id: "due", type: "date", name: "Due"},
    {id: "file", type: "mAsset", name: "File"}] as IAVColumn[];

const data = (chart?: Partial<IAVChartSettings>) => ({id: "database", viewID: "chart-view", view: {
    columns, chart: chart ? {...model.getChartSettings({}), xKeyID: "status", ...chart} : undefined,
}}) as unknown as IAV;
// 被测模块在独立上下文中求值，比较前转换为当前上下文的普通对象。
const plain = <T>(value: T): T => JSON.parse(JSON.stringify(value));
const block = {dataset: {nodeId: "carrier"}, getAttribute: () => "carrier"} as unknown as HTMLElement;
const keysOf = (items: Array<{key: string}>) => plain(items.map(item => item.key));

test("chart settings list only the options that apply to the chart type", () => {
    const {methods} = setup();
    const bar = methods.getChartSettingItems(data({}).view as IAVTable);
    assert.deepEqual(keysOf(bar), ["chartType", "xKeyID", "measure", "stackKeyID", "sort", "height"]);
    assert.deepEqual(plain(bar.find(item => item.key === "xKeyID").choices.map(choice => choice.value)),
        ["", "name", "status", "type", "done", "tags", "points", "due"]);
    assert.deepEqual(plain(bar.find(item => item.key === "stackKeyID").choices.map(choice => choice.value)), ["", "name", "type", "done"]);
    assert.deepEqual(plain(bar.find(item => item.key === "chartType").choices.map(choice => choice.label)),
        ["Vertical bar", "Horizontal bar", "Line", "Donut", "Number"]);

    const dated = methods.getChartSettingItems(data({xKeyID: "due", measure: "sum"}).view as IAVTable);
    assert.deepEqual(keysOf(dated), ["chartType", "xKeyID", "dateBucket", "measure", "valueKeyID", "stackKeyID", "sort", "height"]);
    assert.deepEqual(plain(dated.find(item => item.key === "valueKeyID").choices.map(choice => choice.value)), ["", "points", "score"]);
    assert.deepEqual(plain(dated.find(item => item.key === "dateBucket").choices.map(choice => choice.label)),
        ["Day", "Week", "Month", "Quarter", "Year"]);

    assert.deepEqual(keysOf(methods.getChartSettingItems(data({chartType: "donut"}).view as IAVTable)),
        ["chartType", "xKeyID", "measure", "sort", "height"]);
    assert.deepEqual(keysOf(methods.getChartSettingItems(data({chartType: "number"}).view as IAVTable)),
        ["chartType", "measure", "height"]);
    assert.deepEqual(keysOf(methods.getChartToggleItems(data({chartType: "number"}).view as IAVTable)), []);
    assert.deepEqual(keysOf(methods.getChartToggleItems(data({}).view as IAVTable)), ["showEmptyGroups", "showLegend", "showDataLabels"]);
});

test("chart setting updates normalize values and keep the stack apart from the group field", () => {
    const {methods} = setup();
    const base = (data({stackKeyID: "type"}).view as IAVTable).chart;
    assert.equal(methods.updateChartSettings(base, "xKeyID", "type").stackKeyID, "");
    assert.equal(methods.updateChartSettings(base, "xKeyID", "done").stackKeyID, "type");
    assert.equal(methods.updateChartSettings(base, "height", "480").height, 480);
    assert.equal(methods.updateChartSettings(base, "height", "12").height, 360);
    assert.equal(methods.updateChartSettings(base, "chartType", "pie").chartType, "bar");
    assert.equal(methods.updateChartSettings(base, "measure", "max").measure, "max");
    assert.equal(methods.updateChartSettings(base, "sort", "labelAsc").sort, "labelAsc");
    assert.equal(methods.updateChartSettings(base, "dateBucket", "quarter").dateBucket, "quarter");
    assert.equal(methods.updateChartSettings(base, "showDataLabels", true).showDataLabels, true);
    assert.equal(base.showDataLabels, false);
});

test("chart settings write undoable transactions and skip unchanged settings", () => {
    const {methods, transactions} = setup();
    const current = data({});
    const previous = plain((current.view as IAVTable).chart);
    const next = methods.updateChartSettings(previous, "chartType", "line");
    assert.equal(methods.setChartSettings({} as IProtyle, block, current, next), true);
    assert.equal(transactions.length, 1);
    const [perform] = transactions[0].do;
    const [undo] = transactions[0].undo;
    assert.equal(perform.action, "setAttrViewChart");
    assert.equal(perform.viewID, "chart-view");
    assert.equal(perform.blockID, "carrier");
    assert.equal(perform.action === "setAttrViewChart" && perform.data.chartType, "line");
    assert.equal(undo.action === "setAttrViewChart" && undo.data.chartType, "bar");
    assert.equal((current.view as IAVTable).chart.chartType, "line");
    assert.equal(methods.setChartSettings({} as IProtyle, block, current, next), false);
    assert.equal(transactions.length, 1);
});

test("chart settings render menu entries and toggles", () => {
    const {methods} = setup();
    const html = methods.getChartSettingsHTML(data({showLegend: true}).view as IAVTable, true);
    assert.match(html, /data-chart-setting="chartType"[\s\S]*Vertical bar/);
    assert.match(html, /data-chart-setting="xKeyID"[\s\S]*Status/);
    assert.match(html, /data-chart-setting="height"[\s\S]*360px/);
    assert.match(html, /data-chart-toggle="showLegend" type="checkbox" class="b3-switch b3-switch--menu" checked/);
    assert.doesNotMatch(html, /data-chart-toggle="showDataLabels"[^>]*checked/);
    const form = methods.getChartSettingsHTML(data({}).view as IAVTable);
    assert.match(form, /<select class="b3-select" data-chart-setting="xKeyID"/);
    assert.doesNotMatch(form, /data-chart-toggle/);
});
