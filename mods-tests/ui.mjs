// Browser check of the mods against a running test instance (see README.md):
// logs in, opens a page holding a database with a Status field, switches it to a chart,
// and fails on any page error. Screenshots go to ./shots.
import {mkdirSync} from "node:fs";
import {chromium} from "playwright";
import {api, BASE, ACCESS_CODE, check, finish, createNotebook, createDatabaseDoc, newID, raw} from "./lib.mjs";

const shots = new URL("./shots/", import.meta.url).pathname;
mkdirSync(shots, {recursive: true});

const notebook = await createNotebook("mods-ui-" + Date.now().toString(36));
const {docID, avID, blockID} = await createDatabaseDoc(notebook, "/Tasks");
const statusKey = newID();
await api("/api/av/addAttributeViewKey", {avID, keyID: statusKey, keyName: "Status", keyType: "status", keyIcon: "", previousKeyID: ""});
const view = (await api("/api/av/renderAttributeView", {id: avID})).view;
const primary = view.columns.find(column => column.type === "block");
await api("/api/av/appendAttributeViewDetachedBlocksWithValues", {avID, blocksValues: [["Write", "Done"], ["Ship", "Not started"]].map(([name, status]) => [
    {keyID: primary.id, type: "block", block: {content: name}},
    {keyID: statusKey, type: "status", mSelect: [{content: status}]},
])});

const browser = await chromium.launch({channel: process.env.PW_CHANNEL || "chrome", headless: true});
const page = await browser.newPage({viewport: {width: 1400, height: 900}});
const pageErrors = [];
page.on("pageerror", error => pageErrors.push(error.message));

await page.goto(`${BASE}/stage/build/desktop/?id=${docID}`);
if (page.url().includes("check-auth")) {
    await page.fill("#authCode", ACCESS_CODE);
    await page.keyboard.press("Enter");
}
await page.waitForSelector(".protyle-wysiwyg", {timeout: 30000});
const table = page.locator(`.av[data-av-id="${avID}"]`).first();
await table.waitFor({timeout: 30000});
await page.waitForFunction(id => document.querySelectorAll(`.av[data-av-id="${id}"] .av__row:not(.av__row--header)`).length >= 2, avID, {timeout: 15000}).catch(() => {});
check("the database shows in the page", await table.isVisible());
check("its rows show", await table.locator(".av__row:not(.av__row--header)").count() >= 2);
check("the Status chips show", await table.getByText("Done", {exact: true}).count() >= 1 && await table.getByText("Not started", {exact: true}).count() >= 1);
await page.screenshot({path: shots + "table.png"});

await api("/api/av/changeAttrViewLayout", {avID, blockID, layoutType: "chart"});
await page.reload();
await page.waitForSelector(`.av[data-av-id="${avID}"]`, {timeout: 30000});
const chart = page.locator(`.av[data-av-id="${avID}"] canvas, .av[data-av-id="${avID}"] svg`).first();
await chart.waitFor({timeout: 15000}).catch(() => {});
check("the chart draws", await chart.count() >= 1);
await page.screenshot({path: shots + "chart.png"});

check("no page error", pageErrors.length === 0, pageErrors.slice(0, 3).join(" | "));
await browser.close();
await raw("/api/notebook/removeNotebook", {notebook});
finish();
