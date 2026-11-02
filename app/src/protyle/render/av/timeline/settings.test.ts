import * as capabilities from "../capabilities";
import * as dates from "./date";
import * as model from "./model";
import * as assert from "node:assert/strict";
import {test} from "node:test";
import {join} from "node:path";
import {ScriptTarget} from "typescript";
import {loadModule} from "../../../../testSupport/loadModule";

type TMenuItem = {label: string; checked: boolean; click: () => void};

const setup = () => {
    const transactions: Array<{do: IOperation[], undo: IOperation[]}> = [];
    const menus: TMenuItem[][] = [];
    const window = {siyuan: {isPublish: false, config: {lang: "en"}, languages: {
        calcOperatorNone: "None", timelineStartField: "Start", timelineEndField: "End", timelineEndFromStart: "From start",
        zoom: "Zoom", calendarColorField: "Color by", conditionalColors: "Conditional colors",
        timelineZoomDay: "Day", timelineZoomWeek: "Week", timelineZoomMonth: "Month", timelineZoomQuarter: "Quarter",
    }}};
    const methods = loadModule<typeof import("./settings")>(join(__dirname, "settings.ts"), {
        window,
        require: (id: string) => ({
            "../capabilities": capabilities,
            "./date": dates,
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

const columns = [{id: "name", type: "block", name: "Name"}, {id: "start", type: "date", name: "Start date"},
    {id: "end", type: "date", name: "Due"}, {id: "created", type: "created", name: "Created"},
    {id: "status", type: "status", name: "Status"}, {id: "tags", type: "mSelect", name: "Tags"},
    {id: "text", type: "text", name: "Text"}] as IAVColumn[];

const data = (timeline?: Partial<IAVTimelineSettings>) => ({id: "database", viewID: "timeline-view", view: {
    columns, timeline: timeline ? {startKeyID: "start", endKeyID: "end", zoom: "week", colorBy: "status", ...timeline} : undefined,
}}) as unknown as IAV;
// 被测模块在独立上下文中求值，比较前转换为当前上下文的普通对象。
const plain = <T>(value: T): T => JSON.parse(JSON.stringify(value));
const block = {dataset: {nodeId: "carrier"}, getAttribute: () => "carrier"} as unknown as HTMLElement;
const settingsOf = (operation: IOperation) => {
    if (operation.action !== "setAttrViewTimeline") {
        throw new Error(`unexpected action ${operation.action}`);
    }
    return operation.data;
};

test("timeline settings list date, option and conditional choices", () => {
    const {methods} = setup();
    const items = methods.getTimelineSettingItems(data({}).view as IAVTable);
    assert.deepEqual(plain(items.map(item => item.key)), ["startKeyID", "endKeyID", "zoom", "colorBy"]);
    assert.deepEqual(plain(items[0].choices.map(choice => choice.value)), ["", "start", "end", "created"]);
    assert.deepEqual(plain(items[1].choices.map(choice => choice.value)), ["", "end", "created"]);
    assert.deepEqual(plain(items[2].choices.map(choice => choice.value)), ["day", "week", "month", "quarter"]);
    assert.deepEqual(plain(items[3].choices.map(choice => choice.value)), ["", "status", "tags", "conditional"]);
    const html = methods.getTimelineSettingsHTML(data({}).view as IAVTable, true);
    assert.doesNotMatch(html, /<select/);
    assert.match(html, /data-timeline-setting="zoom"[\s\S]*Week/);
    assert.match(methods.getTimelineSettingsHTML(data().view as IAVTable), /<option value="" selected>None/);
});

test("changing the start field clears a matching end field", () => {
    const {methods} = setup();
    const current = {startKeyID: "start", endKeyID: "end", zoom: "week", colorBy: ""} as IAVTimelineSettings;
    assert.deepEqual(plain(methods.updateTimelineSettings(current, "startKeyID", "end")), {...current, startKeyID: "end", endKeyID: ""});
    assert.deepEqual(plain(methods.updateTimelineSettings(current, "startKeyID", "created")), {...current, startKeyID: "created"});
    assert.equal(methods.updateTimelineSettings(current, "zoom", "decade").zoom, "week");
});

test("timeline layout submenus mark the selection and write undoable settings", () => {
    const {methods, transactions, menus} = setup();
    const current = data({});
    let click: (event: {preventDefault: () => void; stopPropagation: () => void}) => void;
    let refreshed = 0;
    const button = {tagName: "BUTTON", dataset: {timelineSetting: "zoom"},
        addEventListener: (_name: string, handler: typeof click) => { click = handler; }};
    methods.bindTimelineSettings({protyle: {options: {}} as IProtyle, blockElement: block, data: current,
        menuElement: {querySelectorAll: () => [button]} as unknown as Element, onChange: () => { refreshed++; }});
    click({preventDefault() { return; }, stopPropagation() { return; }});
    assert.deepEqual(plain(menus[0].map(item => item.checked)), [false, true, false, false]);
    menus[0][1].click();
    assert.equal(transactions.length, 0);
    menus[0][3].click();
    assert.equal(transactions.length, 1);
    assert.equal(transactions[0].do[0].viewID, "timeline-view");
    assert.deepEqual(plain(settingsOf(transactions[0].do[0])), {startKeyID: "start", endKeyID: "end", zoom: "quarter", colorBy: "status"});
    assert.equal(settingsOf(transactions[0].undo[0]).zoom, "week");
    assert.equal((current.view as IAVTable).timeline.zoom, "quarter");
    assert.equal(refreshed, 1);
});

test("timeline settings stay read-only in publish and archived editors", () => {
    const {methods, transactions, window} = setup();
    for (const mode of ["disabled", "publish", "created", "snapshot"]) {
        const select = {disabled: false, addEventListener: () => assert.fail("read-only handler")};
        window.siyuan.isPublish = mode === "publish";
        const protyle = {disabled: mode === "disabled", options: {history: {[mode]: "archive"}}} as IProtyle;
        methods.bindTimelineSettings({protyle, blockElement: block, data: data({}),
            menuElement: {querySelectorAll: () => [select]} as unknown as Element});
        assert.equal(select.disabled, true);
    }
    assert.equal(transactions.length, 0);
});
