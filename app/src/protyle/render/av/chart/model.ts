import {hasAVKeyGroup, isAVDateType} from "../capabilities";

export const CHART_TYPES = ["bar", "horizontalBar", "line", "donut", "number"] as const;
export const CHART_MEASURES = ["count", "sum", "average", "min", "max"] as const;
export const CHART_SORTS = ["default", "valueDesc", "valueAsc", "labelAsc"] as const;
export const CHART_DATE_BUCKETS = ["day", "week", "month", "quarter", "year"] as const;
export const CHART_HEIGHTS = [240, 360, 480, 640, 800] as const;
export const CHART_DEFAULT_HEIGHT = 360;
export const CHART_MIN_HEIGHT = 160;
export const CHART_MAX_HEIGHT = 1600;

export type TChartType = typeof CHART_TYPES[number];
export type TChartMeasure = typeof CHART_MEASURES[number];
export type TChartSort = typeof CHART_SORTS[number];
export type TChartDateBucket = typeof CHART_DATE_BUCKETS[number];

// 字段用途由内核能力声明的 chartGroup、chartValue 和 chartStack 分组决定。
export const isChartGroupColumn = (column: Pick<IAVColumn, "type"> | undefined) =>
    !!column && hasAVKeyGroup(column.type, "chartGroup");

export const isChartValueColumn = (column: Pick<IAVColumn, "type"> | undefined) =>
    !!column && hasAVKeyGroup(column.type, "chartValue");

export const isChartStackColumn = (column: Pick<IAVColumn, "type"> | undefined) =>
    !!column && hasAVKeyGroup(column.type, "chartStack");

const pick = <T extends string>(values: readonly T[], value: string | undefined, fallback: T): T =>
    values.includes(value as T) ? value as T : fallback;

export const normalizeChartHeight = (height: number | undefined) =>
    typeof height === "number" && height >= CHART_MIN_HEIGHT && height <= CHART_MAX_HEIGHT ? Math.round(height) : CHART_DEFAULT_HEIGHT;

// 未返回设置时按默认的计数柱状图处理，保证切换布局后的首次渲染可用。
export const getChartSettings = (view: Pick<IAVTable, "chart">): IAVChartSettings => {
    const settings = view.chart;
    return {
        chartType: pick(CHART_TYPES, settings?.chartType, "bar"),
        xKeyID: settings?.xKeyID || "",
        dateBucket: pick(CHART_DATE_BUCKETS, settings?.dateBucket, "month"),
        measure: pick(CHART_MEASURES, settings?.measure, "count"),
        valueKeyID: settings?.valueKeyID || "",
        stackKeyID: settings?.stackKeyID || "",
        sort: pick(CHART_SORTS, settings?.sort, "default"),
        showEmptyGroups: !!settings?.showEmptyGroups,
        showLegend: settings ? !!settings.showLegend : true,
        showDataLabels: !!settings?.showDataLabels,
        height: normalizeChartHeight(settings?.height),
    };
};

export interface IChartLabels {
    lang: string;
    empty: string;
    checked: string;
    unchecked: string;
    /** 包含 ${x} 的周标签模板，${x} 为周一的日期。 */
    weekOf: string;
    measures: Record<TChartMeasure, string>;
}

export interface IChartTheme {
    text: string;
    subText: string;
    axis: string;
    split: string;
    primary: string;
    background: string;
    tooltipBackground: string;
    /** 没有选项颜色的分组和系列依次使用的颜色。 */
    palette: string[];
    /** 将选项颜色解析为画布可用的颜色，无法解析时返回空字符串。 */
    resolveColor: (reference: {color?: string, resolvedColor?: IAVColor}) => string;
}

const toLocale = (lang: string) => {
    const locale = (lang || "en").replace("_", "-");
    try {
        return Intl.DateTimeFormat.supportedLocalesOf([locale]).length > 0 ? locale : "en";
    } catch {
        return "en";
    }
};

const formatDate = (time: number, lang: string, options: Intl.DateTimeFormatOptions) =>
    new Intl.DateTimeFormat(toLocale(lang), options).format(new Date(time));

// 日期分组的开始时间由内核按其本地时区计算；客户端与内核时区不同时按标识中的日期显示，避免错位。
const getBucketDate = (key: string) => {
    const match = /^(\d{4})(?:-(\d{2}))?(?:-(\d{2}))?$/.exec(key);
    if (!match) {
        return undefined;
    }
    return new Date(Number(match[1]), match[2] ? Number(match[2]) - 1 : 0, match[3] ? Number(match[3]) : 1).getTime();
};

const getISOWeekStart = (key: string) => {
    const match = /^(\d{4})-W(\d{2})$/.exec(key);
    if (!match) {
        return undefined;
    }
    // ISO 第 1 周包含 1 月 4 日。
    const january4 = new Date(Number(match[1]), 0, 4);
    const monday = new Date(january4.getTime());
    monday.setDate(january4.getDate() - (january4.getDay() + 6) % 7 + (Number(match[2]) - 1) * 7);
    return monday.getTime();
};

export const formatChartGroupLabel = (group: Pick<IAVChartCategory, "key" | "label" | "empty">, type: string,
                                      bucket: TChartDateBucket, labels: IChartLabels) => {
    if (group.empty) {
        return labels.empty;
    }
    if (type === "checkbox") {
        return group.key === "checked" ? labels.checked : labels.unchecked;
    }
    if (isAVDateType(type)) {
        if (bucket === "week") {
            const start = getISOWeekStart(group.key);
            return start === undefined ? group.key : labels.weekOf.replace("${x}",
                formatDate(start, labels.lang, {year: "numeric", month: "short", day: "numeric"}));
        }
        if (bucket === "quarter") {
            const match = /^(\d{4})-Q(\d)$/.exec(group.key);
            return match ? `Q${match[2]} ${match[1]}` : group.key;
        }
        const date = getBucketDate(group.key);
        if (date === undefined) {
            return group.key;
        }
        if (bucket === "day") {
            return formatDate(date, labels.lang, {year: "numeric", month: "short", day: "numeric"});
        }
        if (bucket === "month") {
            return formatDate(date, labels.lang, {year: "numeric", month: "short"});
        }
        return group.key;
    }
    return group.label || group.key;
};

export const formatChartValue = (value: number | null | undefined, lang: string) => {
    if (value === null || value === undefined || !Number.isFinite(value)) {
        return "-";
    }
    return new Intl.NumberFormat(toLocale(lang), {maximumFractionDigits: 2}).format(value);
};

export const getChartMeasureLabel = (data: Pick<IAVChartData, "measure">, valueColumn: IAVColumn | undefined,
                                     labels: IChartLabels) => {
    const measure = labels.measures[data.measure] || labels.measures.count;
    return data.measure === "count" || !valueColumn ? measure : `${measure} · ${valueColumn.name}`;
};

/** 返回数据点的示例条目和条目数，环形图只有一个系列。 */
export const getChartPoint = (data: IAVChartData, seriesIndex: number, categoryIndex: number) => {
    const series = data.series[seriesIndex];
    const category = data.categories[categoryIndex];
    if (!series || !category) {
        return undefined;
    }
    return {
        category,
        series,
        items: series.items?.[categoryIndex] || [],
        count: series.counts?.[categoryIndex] || 0,
        value: series.values?.[categoryIndex] ?? null,
    };
};

export interface IChartOptionInput {
    data: IAVChartData;
    settings: IAVChartSettings;
    columns: IAVColumn[];
    theme: IChartTheme;
    labels: IChartLabels;
}

const getGroupColor = (group: {color?: string, resolvedColor?: IAVColor}, theme: IChartTheme, index: number) =>
    (group.color || group.resolvedColor ? theme.resolveColor(group) : "") || theme.palette[index % theme.palette.length] ||
    theme.primary;

/**
 * 由内核汇总结果生成 ECharts 配置；数字图不使用 ECharts，返回 undefined。
 * 文字、坐标轴和网格颜色来自主题，选项分组沿用选项颜色，单系列的非选项分组使用主题主色。
 */
export const getChartOption = (input: IChartOptionInput): Record<string, any> | undefined => {
    const {data, settings, theme, labels} = input;
    const type = settings.chartType as TChartType;
    if (type === "number" || !data || data.categories.length === 0 || data.series.length === 0) {
        return undefined;
    }
    const bucket = settings.dateBucket as TChartDateBucket;
    const categoryLabels = data.categories.map(category => formatChartGroupLabel(category, data.xType, bucket, labels));
    const valueColumn = input.columns.find(column => column.id === settings.valueKeyID);
    const measureLabel = getChartMeasureLabel(data, valueColumn, labels);
    const stacked = !!data.stackType && type !== "donut";
    const seriesLabels = data.series.map(series => stacked ?
        formatChartGroupLabel(series, data.stackType, bucket, labels) : measureLabel);
    const valueFormatter = (value: number | null) => formatChartValue(value, labels.lang);
    const textStyle = {color: theme.text};
    const tooltip = {
        backgroundColor: theme.tooltipBackground,
        borderColor: theme.split,
        textStyle: {color: theme.text},
        valueFormatter,
        confine: true,
    };
    const legend = {
        show: settings.showLegend && (stacked ? data.series.length > 0 : type === "donut"),
        type: "scroll",
        bottom: 0,
        textStyle: {color: theme.subText},
        pageTextStyle: {color: theme.subText},
        pageIconColor: theme.subText,
    };
    const label = {
        show: settings.showDataLabels,
        color: theme.text,
        formatter: (params: { value: number | null }) => valueFormatter(params.value),
    };
    const option: Record<string, any> = {
        backgroundColor: "transparent",
        textStyle,
        tooltip,
        legend,
        animationDuration: 300,
    };
    if (type === "donut") {
        const series = data.series[0];
        option.tooltip = {...tooltip, trigger: "item"};
        option.series = [{
            type: "pie",
            name: measureLabel,
            radius: ["45%", "72%"],
            center: ["50%", legend.show ? "45%" : "50%"],
            avoidLabelOverlap: true,
            itemStyle: {borderColor: theme.background, borderWidth: 2},
            label: {
                show: settings.showDataLabels,
                color: theme.text,
                formatter: (params: { name: string, value: number | null }) => `${params.name}: ${valueFormatter(params.value)}`,
            },
            labelLine: {lineStyle: {color: theme.axis}},
            data: data.categories.map((category, index) => ({
                name: categoryLabels[index],
                value: series.values[index],
                itemStyle: {color: getGroupColor(category, theme, index)},
            })),
        }];
        return option;
    }
    const horizontal = type === "horizontalBar";
    const categoryAxis = {
        type: "category",
        data: categoryLabels,
        inverse: horizontal,
        axisLine: {lineStyle: {color: theme.axis}},
        axisTick: {show: false},
        axisLabel: {color: theme.subText, hideOverlap: true},
    };
    const valueAxis = {
        type: "value",
        axisLine: {show: false},
        axisLabel: {color: theme.subText, formatter: valueFormatter},
        splitLine: {lineStyle: {color: theme.split}},
    };
    option.grid = {left: 8, right: 16, top: 16, bottom: legend.show ? 40 : 8, containLabel: true};
    option.xAxis = horizontal ? valueAxis : categoryAxis;
    option.yAxis = horizontal ? categoryAxis : valueAxis;
    option.tooltip = {...tooltip, trigger: "axis", axisPointer: {type: type === "line" ? "line" : "shadow"}};
    const optionColors = !stacked && data.categories.some(category => category.color || category.resolvedColor);
    option.series = data.series.map((series, seriesIndex) => {
        const color = stacked ? getGroupColor(series, theme, seriesIndex) : theme.primary;
        const values = series.values.map((value, index) => optionColors && type !== "line" ? {
            value,
            itemStyle: {color: getGroupColor(data.categories[index], theme, index)},
        } : value);
        if (type === "line") {
            return {
                type: "line",
                name: seriesLabels[seriesIndex],
                data: values,
                color,
                smooth: false,
                symbolSize: 6,
                connectNulls: false,
                label: {...label, position: "top"},
            };
        }
        return {
            type: "bar",
            name: seriesLabels[seriesIndex],
            data: values,
            color,
            stack: stacked ? "total" : undefined,
            barMaxWidth: 48,
            itemStyle: {borderRadius: stacked ? 0 : (horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0])},
            label: {...label, position: stacked ? "inside" : (horizontal ? "right" : "top")},
        };
    });
    return option;
};
