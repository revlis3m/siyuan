import {isElementMasked, onMaskChange} from "../../../util/maskAmounts";
import {Constants} from "../../../../constants";
import {escapeHtml} from "../../../../util/escape";
import {loadECharts} from "../../../util/echarts";
import {Menu} from "../../../../plugin/Menu";
import {toggleMenu} from "../../../../menus/menuToggle";
import {openDatabaseRowByData} from "../openDatabaseRow";
import {avRender, genTabHeaderHTML, updateSearch} from "../render";
import {replaceAVContainer} from "../container";
import {bindAvSearch, captureAvSearchSelection, deferAvSearchRender} from "../search";
import {getAVData, setAVData} from "../virtualScroll";
import {finishAVLocate} from "../locate";
import {
    formatChartGroupLabel,
    formatChartValue,
    getChartMeasureLabel,
    getChartOption,
    getChartPoint,
    getChartSettings,
    IChartTheme,
    isChartGroupColumn,
    TChartDateBucket,
} from "./model";
import {bindChartSettings, canEditChart, getChartLabels, getChartSettingsHTML} from "./settings";

interface IChartInstance {
    canvas: HTMLElement;
    observer: ResizeObserver;
    refresh: () => void;
}

const instances = new WeakMap<Element, IChartInstance>();
// 没有选项颜色的分组依次使用的内置颜色序号，跳过与背景同色的 13。
const PALETTE_INDEXES = [6, 4, 2, 9, 10, 3, 11, 1, 12, 7, 8, 5];

/** 释放数据库块上的图表实例，重新渲染或切换到其他布局前调用。 */
export const disposeAVChart = (blockElement: Element) => {
    const instance = instances.get(blockElement);
    if (!instance) {
        return;
    }
    instances.delete(blockElement);
    instance.observer.disconnect();
    window.echarts?.dispose(instance.canvas);
};

const resolveColor = (probe: HTMLElement, expression: string) => {
    probe.style.color = "";
    probe.style.color = expression;
    return probe.style.color ? getComputedStyle(probe).color : "";
};

// 画布无法使用 CSS 变量，渲染时在数据库块内按当前主题和自定义颜色解析为具体颜色。
const getChartTheme = (probe: HTMLElement): IChartTheme => {
    const resolve = (expression: string) => resolveColor(probe, expression);
    const dark = window.siyuan.config.appearance.mode === 1;
    return {
        text: resolve("var(--b3-theme-on-background)"),
        subText: resolve("var(--b3-theme-on-surface)"),
        axis: resolve("var(--b3-border-color)"),
        split: resolve("var(--b3-border-color)"),
        primary: resolve("var(--b3-theme-primary)"),
        background: resolve("var(--b3-theme-background)"),
        tooltipBackground: resolve("var(--b3-menu-background)"),
        palette: PALETTE_INDEXES.map(index => resolve(`var(--b3-font-color${index})`)),
        resolveColor: (reference) => {
            const custom = dark ? reference.resolvedColor?.dark?.color : reference.resolvedColor?.light?.color;
            if (custom) {
                return resolve(custom);
            }
            const index = Number(reference.color);
            if (!Number.isInteger(index) || index < 1) {
                return "";
            }
            return resolve(index === 13 ? "var(--b3-theme-on-surface)" :
                `var(--b3-font-color${index}, var(--b3-theme-on-surface))`);
        },
    };
};

// revlis3m mods: redraw charts when amounts are masked or revealed.
onMaskChange(element => element.querySelectorAll('.av[data-av-type="chart"]').forEach(block => instances.get(block)?.refresh()));

/** 外观切换后按新主题的颜色重新绘制图表，等待主题样式表加载完成后再解析颜色。 */
export const refreshAVChartTheme = (element: Element) => {
    const blocks = Array.from(element.querySelectorAll('.av[data-av-type="chart"]'));
    if (blocks.every(block => !instances.has(block))) {
        return;
    }
    const started = Date.now();
    const check = () => {
        const pending = Array.from(document.querySelectorAll<HTMLLinkElement>('link[rel="stylesheet"]')).some(link => !link.sheet);
        if (pending && Date.now() - started < 3000) {
            setTimeout(check, 50);
            return;
        }
        blocks.forEach(block => instances.get(block)?.refresh());
    };
    check();
};

export const refreshChartReadonly = (protyle: IProtyle) => {
    protyle.wysiwyg.element.querySelectorAll<HTMLElement>('.av[data-av-type="chart"]').forEach(blockElement => {
        const data = getAVData(blockElement);
        if (data?.viewType === "chart") {
            void renderChart(blockElement, protyle, data);
        }
    });
};

const openChartItem = (protyle: IProtyle, blockElement: HTMLElement, item: IAVChartItem) => openDatabaseRowByData(protyle, {
    avID: blockElement.dataset.avId, databaseBlockID: blockElement.dataset.nodeId, notebookID: protyle.notebookId,
    itemID: item.id, valueID: item.valueID, title: item.title || window.siyuan.languages.untitled,
    boundBlockID: item.blockID, isDetached: item.isDetached, focusPrimary: true,
});

// 点击数据点后列出该分组（及子分组）中的示例条目，选择条目后打开。
const openChartPointMenu = (protyle: IProtyle, blockElement: HTMLElement, data: IAV, seriesIndex: number,
                            categoryIndex: number, position: {x: number, y: number}, target: Element) => {
    const view = data.view as IAVTable;
    const chartData = view.chartData;
    const point = chartData ? getChartPoint(chartData, seriesIndex, categoryIndex) : undefined;
    if (!point) {
        return;
    }
    const labels = getChartLabels();
    const bucket = getChartSettings(view).dateBucket as TChartDateBucket;
    let title = formatChartGroupLabel(point.category, chartData.xType, bucket, labels);
    if (chartData.stackType && point.series) {
        title += ` · ${formatChartGroupLabel(point.series, chartData.stackType, bucket, labels)}`;
    }
    toggleMenu({target, toggle: false, build: () => {
        const menu = new Menu();
        buildChartPointMenu(menu, protyle, blockElement, title, point);
        menu.open(position);
    }});
};

const buildChartPointMenu = (menu: Menu, protyle: IProtyle, blockElement: HTMLElement, title: string,
                             point: ReturnType<typeof getChartPoint>) => {
    menu.addItem({
        type: "readonly", iconHTML: "",
        label: `${escapeHtml(title)}<div class="ft__on-surface ft__smaller">${escapeHtml(window.siyuan.languages.chartItemCount.replace("${x}", point.count.toString()))}</div>`,
    });
    if (point.items.length > 0) {
        menu.addSeparator();
    }
    point.items.forEach(item => {
        menu.addItem({
            icon: "iconOpen",
            label: escapeHtml(item.title || window.siyuan.languages.untitled),
            click: () => {
                void openChartItem(protyle, blockElement, item);
            },
        });
    });
    if (point.count > point.items.length) {
        menu.addItem({
            type: "readonly", iconHTML: "",
            label: `<span class="ft__on-surface">${escapeHtml(window.siyuan.languages.chartItemsMore.replace("${x}",
                (point.count - point.items.length).toString()))}</span>`,
        });
    }
};

const mountChart = async (blockElement: HTMLElement, protyle: IProtyle, data: IAV, canvas: HTMLElement) => {
    const loaded = await loadECharts(Constants.PROTYLE_CDN);
    if (!loaded || !canvas.isConnected || instances.has(blockElement) ||
        blockElement.querySelector(".av__chart-canvas") !== canvas) {
        return;
    }
    const probe = blockElement.querySelector<HTMLElement>(".av__chart-probe");
    const view = data.view as IAVTable;
    const build = () => getChartOption({
        data: view.chartData, settings: getChartSettings(view), columns: view.columns,
        theme: getChartTheme(probe), labels: getChartLabels(), masked: isElementMasked(blockElement),
    });
    const option = build();
    if (!option) {
        return;
    }
    const chart = window.echarts.init(canvas);
    chart.setOption(option);
    const observer = new ResizeObserver(() => {
        if (!canvas.isConnected) {
            if (instances.get(blockElement)?.canvas === canvas) {
                disposeAVChart(blockElement);
            }
            return;
        }
        chart.resize();
    });
    observer.observe(canvas);
    instances.set(blockElement, {
        canvas, observer, refresh: () => {
            const next = build();
            if (next) {
                chart.setOption(next);
            }
        },
    });
    chart.on("click", (params: {componentType?: string, seriesIndex?: number, dataIndex?: number, event?: {event?: MouseEvent}}) => {
        if (params.componentType !== "series" || params.dataIndex === undefined) {
            return;
        }
        const event = params.event?.event;
        const rect = canvas.getBoundingClientRect();
        openChartPointMenu(protyle, blockElement, data, params.seriesIndex || 0, params.dataIndex,
            event ? {x: event.clientX, y: event.clientY} : {x: rect.left + rect.width / 2, y: rect.top + rect.height / 2}, canvas);
    });
};

export const renderChart = async (blockElement: HTMLElement, protyle: IProtyle, data: IAV, cb?: (data: IAV) => void) => {
    const search = blockElement.querySelector<HTMLElement>('[data-type="av-search"]');
    if (deferAvSearchRender(search, () => {
        // 组合输入结束后重新读取当前视图和查询，并沿用渲染请求的过期检查。
        blockElement.removeAttribute("data-render");
        void avRender(blockElement, protyle, cb);
    })) {
        return;
    }
    const view = data.view as IAVTable;
    const settings = getChartSettings(view);
    const chartData = view.chartData;
    const editable = canEditChart(protyle);
    const query = search?.textContent || "";
    const isSearching = search === document.activeElement;
    const selection = captureAvSearchSelection(search);
    const labels = getChartLabels();
    const xColumn = view.columns.find(column => column.id === settings.xKeyID);
    const valueColumn = view.columns.find(column => column.id === settings.valueKeyID);
    const height = settings.height;
    let body: string;
    if (settings.chartType === "number") {
        body = `<div class="av__chart-number" style="height:${height}px">
    <div class="av__chart-number-value">${escapeHtml(formatChartValue(chartData?.total ?? null, labels.lang))}</div>
    <div class="av__chart-number-label">${escapeHtml(chartData ? getChartMeasureLabel(chartData, valueColumn, labels) : labels.measures.count)}</div>
</div>`;
    } else if (!isChartGroupColumn(xColumn)) {
        body = `<div class="av__chart-empty" style="min-height:${height}px"><svg><use xlink:href="#iconAlignBottom"></use></svg>
    <p>${window.siyuan.languages.chartSelectGroupField}</p>
    ${editable ? `<div class="av__chart-setup">${getChartSettingsHTML(view)}</div>` : ""}
</div>`;
    } else if (!chartData || chartData.categories.length === 0 || chartData.series.length === 0) {
        body = `<div class="av__chart-empty" style="min-height:${height}px"><svg><use xlink:href="#iconAlignBottom"></use></svg>
    <p>${window.siyuan.languages.chartNoData}</p>
</div>`;
    } else {
        body = `<div class="av__chart-canvas" style="height:${height}px"></div>`;
    }
    disposeAVChart(blockElement);
    blockElement.removeAttribute(Constants.ATTRIBUTE_V_SCROLL);
    replaceAVContainer(blockElement, `<div class="av__container fn__block">
    ${genTabHeaderHTML(data, !!query || isSearching, editable, blockElement, editable)}
    <div class="av__chart" contenteditable="false" data-chart-type="${settings.chartType}">
        <span class="av__chart-probe" aria-hidden="true"></span>
        ${body}
    </div>
    <div class="av__cursor" contenteditable="true">${Constants.ZWSP}</div>
</div>`);
    blockElement.dataset.render = "true";
    setAVData(blockElement, data);
    const root = blockElement.querySelector<HTMLElement>(".av__chart");
    root.addEventListener("click", event => {
        // 图表区域的点击由图表自身处理，不触发条目选择等数据库交互。
        event.stopPropagation();
    });
    if (root.querySelector(".av__chart-setup")) {
        bindChartSettings({
            protyle, blockElement, data, menuElement: root, onChange: () => {
                blockElement.removeAttribute("data-render");
                void avRender(blockElement, protyle);
            },
        });
    }
    bindAvSearch({blockElement, query, isSearching, selection, onChange: () => updateSearch(blockElement, protyle)});
    finishAVLocate(blockElement, protyle, data);
    const canvas = root.querySelector<HTMLElement>(".av__chart-canvas");
    if (canvas) {
        void mountChart(blockElement, protyle, data, canvas);
    }
    cb?.(data);
};
