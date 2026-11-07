import * as assert from "node:assert/strict";
import {test} from "node:test";
import {
    formatChartGroupLabel,
    formatChartValue,
    getChartMeasureLabel,
    getChartOption,
    getChartPoint,
    getChartSettings,
    IChartLabels,
    IChartTheme,
    isChartGroupColumn,
    isChartStackColumn,
    isChartValueColumn,
    normalizeChartHeight,
} from "./model";

const labels: IChartLabels = {
    lang: "en_US", empty: "Empty", checked: "Checked", unchecked: "Unchecked", weekOf: "Week of ${x}",
    measures: {count: "Count", sum: "Sum", average: "Average", min: "Min", max: "Max"},
};

const theme: IChartTheme = {
    text: "text", subText: "sub", axis: "axis", split: "split", primary: "primary", background: "bg",
    tooltipBackground: "tooltip", palette: ["p0", "p1", "p2"],
    resolveColor: (reference) => reference.resolvedColor ? "custom" : reference.color ? `c${reference.color}` : "",
};

const columns = [
    {id: "name", name: "Name", type: "block"},
    {id: "status", name: "Status", type: "status"},
    {id: "points", name: "Points", type: "number"},
    {id: "due", name: "Due", type: "date"},
] as IAVColumn[];

const settings = (patch: Partial<IAVChartSettings> = {}): IAVChartSettings => ({
    ...getChartSettings({chart: undefined}), xKeyID: "status", ...patch,
});

const optionData = (patch: Partial<IAVChartData> = {}): IAVChartData => ({
    measure: "count", xType: "status", stackType: "", total: 5, itemCount: 5,
    categories: [
        {key: "Todo", label: "Todo", color: "5", value: 2, count: 2},
        {key: "Done", label: "Done", color: "11", value: 3, count: 3},
        {key: "", label: "", empty: true, value: 0, count: 0},
    ],
    series: [{key: "", label: "", values: [2, 3, 0], counts: [2, 3, 0],
        items: [[{id: "a", title: "A", blockID: "", valueID: "va", isDetached: true}], [], []]}],
    ...patch,
});

test("chart settings fall back to defaults and normalize invalid values", () => {
    assert.deepEqual(getChartSettings({}), {
        chartType: "bar", xKeyID: "", dateBucket: "month", measure: "count", valueKeyID: "", stackKeyID: "",
        sort: "default", showEmptyGroups: false, showLegend: true, showDataLabels: false, height: 360,
    });
    const normalized = getChartSettings({chart: {
        chartType: "pie" as IAVChartSettings["chartType"], xKeyID: "x", dateBucket: "hour" as IAVChartSettings["dateBucket"],
        measure: "median" as IAVChartSettings["measure"], valueKeyID: "v", stackKeyID: "s", sort: "random" as IAVChartSettings["sort"],
        showEmptyGroups: true, showLegend: false, showDataLabels: true, height: 0,
    }});
    assert.equal(normalized.chartType, "bar");
    assert.equal(normalized.dateBucket, "month");
    assert.equal(normalized.measure, "count");
    assert.equal(normalized.sort, "default");
    assert.equal(normalized.showLegend, false);
    assert.equal(normalized.height, 360);
    assert.equal(normalizeChartHeight(480), 480);
    assert.equal(normalizeChartHeight(100), 360);
    assert.equal(normalizeChartHeight(2000), 360);
});

test("chart field predicates follow the kernel field types", () => {
    assert.equal(isChartGroupColumn({type: "status"}), true);
    assert.equal(isChartGroupColumn({type: "created"}), true);
    assert.equal(isChartGroupColumn({type: "relation"}), true);
    assert.equal(isChartGroupColumn({type: "mAsset"}), false);
    assert.equal(isChartGroupColumn({type: "block"}), true);
    assert.equal(isChartGroupColumn(undefined), false);
    assert.equal(isChartValueColumn({type: "rollup"}), true);
    assert.equal(isChartValueColumn({type: "text"}), false);
    assert.equal(isChartStackColumn({type: "checkbox"}), true);
    assert.equal(isChartStackColumn({type: "mSelect"}), false);
    assert.equal(isChartStackColumn({type: "relation"}), true);
    assert.equal(isChartStackColumn({type: "block"}), true);
});

test("chart group labels localize empty, checkbox and date buckets", () => {
    assert.equal(formatChartGroupLabel({key: "", label: "", empty: true}, "status", "month", labels), "Empty");
    assert.equal(formatChartGroupLabel({key: "checked", label: ""}, "checkbox", "month", labels), "Checked");
    assert.equal(formatChartGroupLabel({key: "unchecked", label: ""}, "checkbox", "month", labels), "Unchecked");
    assert.equal(formatChartGroupLabel({key: "2026-W45", label: "2026-W45"}, "date", "week", labels), "Week of Nov 2, 2026");
    assert.equal(formatChartGroupLabel({key: "2026-W01", label: "2026-W01"}, "created", "week", labels), "Week of Dec 29, 2025");
    assert.equal(formatChartGroupLabel({key: "2026-Q4", label: "2026-Q4"}, "date", "quarter", labels), "Q4 2026");
    assert.equal(formatChartGroupLabel({key: "2026-11", label: "2026-11"}, "date", "month", labels), "Nov 2026");
    assert.equal(formatChartGroupLabel({key: "2026-11-02", label: "2026-11-02"}, "updated", "day", labels), "Nov 2, 2026");
    assert.equal(formatChartGroupLabel({key: "2026", label: "2026"}, "date", "year", labels), "2026");
    assert.equal(formatChartGroupLabel({key: "proj001", label: "Project"}, "relation", "month", labels), "Project");
    assert.equal(formatChartGroupLabel({key: "1.5", label: ""}, "number", "month", labels), "1.5");
});

test("chart values and measure labels are formatted for display", () => {
    assert.equal(formatChartValue(1234.5678, "en_US"), "1,234.57");
    assert.equal(formatChartValue(null, "en_US"), "-");
    assert.equal(formatChartValue(2, "not a locale"), "2");
    assert.equal(getChartMeasureLabel({measure: "count"}, columns[2], labels), "Count");
    assert.equal(getChartMeasureLabel({measure: "average"}, columns[2], labels), "Average · Points");
});

test("bar options use option colors per group and theme colors for axes", () => {
    const option = getChartOption({data: optionData(), settings: settings(), columns, theme, labels});
    assert.deepEqual(option.xAxis.data, ["Todo", "Done", "Empty"]);
    assert.equal(option.xAxis.type, "category");
    assert.equal(option.yAxis.type, "value");
    assert.equal(option.xAxis.axisLabel.color, "sub");
    assert.equal(option.yAxis.splitLine.lineStyle.color, "split");
    assert.equal(option.series.length, 1);
    assert.equal(option.series[0].type, "bar");
    assert.equal(option.series[0].name, "Count");
    assert.deepEqual(option.series[0].data.map((item: {itemStyle: {color: string}}) => item.itemStyle.color), ["c5", "c11", "p2"]);
    assert.equal(option.series[0].stack, undefined);
    assert.equal(option.legend.show, false);
    assert.equal(option.series[0].label.show, false);
    assert.equal(option.tooltip.trigger, "axis");
    assert.equal(option.tooltip.valueFormatter(1000), "1,000");
});

test("stacked, horizontal and line options build one series per stack value", () => {
    const stacked = optionData({
        stackType: "checkbox",
        series: [
            {key: "checked", label: "", values: [1, 2, 0], counts: [1, 2, 0], items: [[], [], []]},
            {key: "unchecked", label: "", values: [1, 1, 0], counts: [1, 1, 0], items: [[], [], []]},
        ],
    });
    const horizontal = getChartOption({data: stacked, settings: settings({chartType: "horizontalBar", showDataLabels: true}), columns, theme, labels});
    assert.equal(horizontal.yAxis.type, "category");
    assert.equal(horizontal.yAxis.inverse, true);
    assert.equal(horizontal.xAxis.type, "value");
    assert.deepEqual(horizontal.series.map((series: {name: string}) => series.name), ["Checked", "Unchecked"]);
    assert.deepEqual(horizontal.series.map((series: {stack: string}) => series.stack), ["total", "total"]);
    assert.deepEqual(horizontal.series.map((series: {color: string}) => series.color), ["p0", "p1"]);
    assert.deepEqual(horizontal.series[0].data, [1, 2, 0]);
    assert.equal(horizontal.legend.show, true);
    assert.equal(horizontal.series[0].label.show, true);
    assert.equal(horizontal.series[0].label.formatter({value: 2.345}), "2.35");

    const line = getChartOption({data: stacked, settings: settings({chartType: "line", showLegend: false}), columns, theme, labels});
    assert.deepEqual(line.series.map((series: {type: string}) => series.type), ["line", "line"]);
    assert.equal(line.series[0].stack, undefined);
    assert.equal(line.legend.show, false);
    assert.equal(line.tooltip.axisPointer.type, "line");
});

test("donut options map categories to slices and number charts skip ECharts", () => {
    const donut = getChartOption({data: optionData(), settings: settings({chartType: "donut"}), columns, theme, labels});
    assert.equal(donut.series[0].type, "pie");
    assert.deepEqual(donut.series[0].data.map((item: {name: string, value: number}) => [item.name, item.value]),
        [["Todo", 2], ["Done", 3], ["Empty", 0]]);
    assert.equal(donut.series[0].data[1].itemStyle.color, "c11");
    assert.equal(donut.legend.show, true);
    assert.equal(donut.tooltip.trigger, "item");
    assert.equal(getChartOption({data: optionData(), settings: settings({chartType: "number"}), columns, theme, labels}), undefined);
    assert.equal(getChartOption({data: optionData({categories: []}), settings: settings(), columns, theme, labels}), undefined);
});

test("date groups without option colors use the primary color", () => {
    const option = getChartOption({data: optionData({
        xType: "date", categories: [{key: "2026-01", label: "2026-01", value: 1, count: 1}],
        series: [{key: "", label: "", values: [null], counts: [1], items: [[]]}], measure: "average",
    }), settings: settings({xKeyID: "due", measure: "average", valueKeyID: "points"}), columns, theme, labels});
    assert.deepEqual(option.series[0].data, [null]);
    assert.equal(option.series[0].color, "primary");
    assert.equal(option.series[0].name, "Average · Points");
    assert.deepEqual(option.xAxis.data, ["Jan 2026"]);
});

test("chart points return the sampled items of a data point", () => {
    const point = getChartPoint(optionData(), 0, 0);
    assert.equal(point.category.key, "Todo");
    assert.equal(point.count, 2);
    assert.equal(point.value, 2);
    assert.deepEqual(point.items.map(item => item.id), ["a"]);
    assert.equal(getChartPoint(optionData(), 1, 0), undefined);
    assert.equal(getChartPoint(optionData(), 0, 9), undefined);
});
