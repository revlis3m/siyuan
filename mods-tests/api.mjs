// API checks of the mods against a running test instance (see README.md):
// Status property, Timeline / Chart / Form layouts, notebook PIN lock.
import {api, raw, check, finish, createNotebook, createDatabaseDoc, renderAV, newID, sleep} from "./lib.mjs";

const stamp = Date.now().toString(36);
const notebook = await createNotebook("mods-check-" + stamp);
const {avID, blockID} = await createDatabaseDoc(notebook, "/Tasks");

console.log("Status property");
const statusKey = newID();
await api("/api/av/addAttributeViewKey", {avID, keyID: statusKey, keyName: "Status", keyType: "status", keyIcon: "", previousKeyID: ""});
const amountKey = newID();
await api("/api/av/addAttributeViewKey", {avID, keyID: amountKey, keyName: "Amount", keyType: "number", keyIcon: "", previousKeyID: statusKey});
let view = (await renderAV(avID)).view;
const statusCol = view.columns.find(column => column.id === statusKey);
check("a Status column can be added", statusCol?.type === "status", JSON.stringify(statusCol?.type));
check("it starts with 3 options", (statusCol?.options || []).length === 3, JSON.stringify(statusCol?.options));
check("the options are in the To-do / In progress / Complete groups",
    JSON.stringify((statusCol?.options || []).map(option => option.group)) === JSON.stringify(["todo", "inProgress", "complete"]),
    JSON.stringify(statusCol?.options));
const primary = view.columns.find(column => column.type === "block");
const doneName = statusCol?.options?.[2]?.name;
const todoName = statusCol?.options?.[0]?.name;
const rows = [["Write", doneName, 10], ["Test", doneName, 20], ["Ship", todoName, 30]].map(([name, status, amount]) => [
    {keyID: primary.id, type: "block", block: {content: name}},
    {keyID: statusKey, type: "status", mSelect: [{content: status}]},
    {keyID: amountKey, type: "number", number: {content: amount, isNotEmpty: true}},
]);
const appended = await raw("/api/av/appendAttributeViewDetachedBlocksWithValues", {avID, blocksValues: rows});
check("rows with a Status value can be added", appended.code === 0, appended.msg);
view = (await renderAV(avID)).view;
check("the table shows the 3 rows", view.rowCount === 3 || view.rows?.length === 3, String(view.rowCount));
const statusCells = (view.rows || []).map(row => row.cells.find(cell => cell.value?.keyID === statusKey)?.value?.mSelect?.[0]?.content);
check("Status values are stored", statusCells.filter(value => value === doneName).length === 2, JSON.stringify(statusCells));

for (const layout of ["timeline", "chart", "form"]) {
    console.log(`${layout} layout`);
    const changed = await raw("/api/av/changeAttrViewLayout", {avID, blockID, layoutType: layout});
    check(`the view can switch to ${layout}`, changed.code === 0, changed.msg);
    const rendered = await raw("/api/av/renderAttributeView", {id: avID});
    check(`${layout} renders`, rendered.code === 0 && rendered.data?.viewType === layout, rendered.msg || rendered.data?.viewType);
}
await api("/api/av/changeAttrViewLayout", {avID, blockID, layoutType: "table"});

console.log("Notebook PIN lock");
const pinNotebook = await createNotebook("mods-lock-" + stamp);
const money = await createDatabaseDoc(pinNotebook, "/Money");
const opened = async () => (await api("/api/notebook/lsNotebooks")).notebooks.find(item => item.id === pinNotebook)?.closed === false;
check("a short PIN is refused", (await raw("/api/notebook/setNotebookPIN", {notebook: pinNotebook, pin: "12"})).code !== 0);
check("a PIN can be set", (await raw("/api/notebook/setNotebookPIN", {notebook: pinNotebook, pin: "4821"})).code === 0);
check("Lock now closes the notebook", (await raw("/api/notebook/lockNotebookPIN", {notebook: pinNotebook})).code === 0 && !(await opened()));
const statusData = await api("/api/notebook/getNotebookPINStatus");
check("it is reported as locked", statusData.notebooks.find(item => item.notebook === pinNotebook)?.locked === true, JSON.stringify(statusData));
await raw("/api/notebook/openNotebook", {notebook: pinNotebook});
await sleep(300);
check("a locked notebook does not open without the PIN", !(await opened()));

console.log("Notebook PIN lock: side doors while locked");
const refused = result => result.code !== 0 && /PIN/.test(result.msg || "");
const file = await raw("/api/file/getFile", {path: `/data/${pinNotebook}/${money.docID}.sy`});
check("its files cannot be read through the file API", refused(file), JSON.stringify(file).slice(0, 120));
const dir = await raw("/api/file/readDir", {path: `/data/${pinNotebook}`});
check("its folder cannot be listed", refused(dir), JSON.stringify(dir).slice(0, 120));
const database = await raw("/api/av/renderAttributeView", {id: money.avID});
check("its databases cannot be rendered", refused(database), JSON.stringify(database).slice(0, 120));
const databaseFile = await raw("/api/file/getFile", {path: `/data/storage/av/${money.avID}.json`});
check("its database files cannot be read", refused(databaseFile), JSON.stringify(databaseFile).slice(0, 120));
const history = await raw("/api/history/getNotebookHistory", {notebook: pinNotebook});
check("its history cannot be asked for", refused(history), JSON.stringify(history).slice(0, 120));
const exported = await raw("/api/export/exportData", {});
check("the whole workspace cannot be exported", refused(exported), JSON.stringify(exported).slice(0, 120));
const otherTable = await raw("/api/av/renderAttributeView", {id: avID});
check("other notebooks keep working", otherTable.code === 0, otherTable.msg);
check("the delay cannot be changed without a PIN", (await raw("/api/notebook/setNotebookPINMinutes", {minutes: 0})).code !== 0);
check("the delay cannot be changed with a wrong PIN", (await raw("/api/notebook/setNotebookPINMinutes", {notebook: pinNotebook, pin: "0000", minutes: 0})).code !== 0);
check("the delay is unchanged", (await api("/api/notebook/getNotebookPINStatus")).minutes === 15);
check("a wrong PIN is refused", (await raw("/api/notebook/unlockNotebookPIN", {notebook: pinNotebook, pin: "0000"})).code !== 0);
check("the right PIN is accepted", (await raw("/api/notebook/unlockNotebookPIN", {notebook: pinNotebook, pin: "4821"})).code === 0);
await raw("/api/notebook/openNotebook", {notebook: pinNotebook});
await sleep(300);
check("the notebook opens once unlocked", await opened());
check("the delay can be changed with the PIN", (await raw("/api/notebook/setNotebookPINMinutes", {notebook: pinNotebook, pin: "4821", minutes: 20})).code === 0 &&
    (await api("/api/notebook/getNotebookPINStatus")).minutes === 20);
await api("/api/notebook/setNotebookPINMinutes", {notebook: pinNotebook, pin: "4821", minutes: 15});
check("once unlocked, its databases render again", (await raw("/api/av/renderAttributeView", {id: money.avID})).code === 0);
check("activity in the notebook can be reported", (await raw("/api/notebook/touchNotebookPIN", {notebook: pinNotebook})).code === 0);
check("the PIN can be removed", (await raw("/api/notebook/removeNotebookPIN", {notebook: pinNotebook, pin: "4821"})).code === 0);

console.log("Notebook PIN lock: guessing");
const guessNotebook = await createNotebook("mods-guess-" + stamp);
await api("/api/notebook/setNotebookPIN", {notebook: guessNotebook, pin: "7305"});
await api("/api/notebook/lockNotebookPIN", {notebook: guessNotebook});
// 12 wrong PINs sent together: only 5 are counted before every PIN is refused.
const guesses = await Promise.all(Array.from({length: 12}, (_, i) =>
    raw("/api/notebook/unlockNotebookPIN", {notebook: guessNotebook, pin: String(1000 + i)})));
check("wrong PINs sent together are all refused", guesses.every(result => result.code !== 0));
check("after 5 wrong PINs the notebook refuses to check more", guesses.some(result => /too many/i.test(result.msg || "")),
    guesses.map(result => result.msg).join(" | ").slice(0, 200));
const right = await raw("/api/notebook/unlockNotebookPIN", {notebook: guessNotebook, pin: "7305"});
check("even the right PIN is refused during the lockout", right.code !== 0 && /too many/i.test(right.msg || ""), right.msg);

// Leave the test instance clean.
// (The guessing notebook stays: it is locked out, and the test instance is thrown away.)
for (const id of [notebook, pinNotebook]) {
    await raw("/api/notebook/removeNotebook", {notebook: id});
}
finish();
