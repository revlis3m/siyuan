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
const opened = async () => (await api("/api/notebook/lsNotebooks")).notebooks.find(item => item.id === pinNotebook)?.closed === false;
check("a short PIN is refused", (await raw("/api/notebook/setNotebookPIN", {notebook: pinNotebook, pin: "12"})).code !== 0);
check("a PIN can be set", (await raw("/api/notebook/setNotebookPIN", {notebook: pinNotebook, pin: "4821"})).code === 0);
check("Lock now closes the notebook", (await raw("/api/notebook/lockNotebookPIN", {notebook: pinNotebook})).code === 0 && !(await opened()));
const statusData = await api("/api/notebook/getNotebookPINStatus");
check("it is reported as locked", statusData.notebooks.find(item => item.notebook === pinNotebook)?.locked === true, JSON.stringify(statusData));
await raw("/api/notebook/openNotebook", {notebook: pinNotebook});
await sleep(300);
check("a locked notebook does not open without the PIN", !(await opened()));
check("a wrong PIN is refused", (await raw("/api/notebook/unlockNotebookPIN", {notebook: pinNotebook, pin: "0000"})).code !== 0);
check("the right PIN is accepted", (await raw("/api/notebook/unlockNotebookPIN", {notebook: pinNotebook, pin: "4821"})).code === 0);
await raw("/api/notebook/openNotebook", {notebook: pinNotebook});
await sleep(300);
check("the notebook opens once unlocked", await opened());
check("the PIN can be removed", (await raw("/api/notebook/removeNotebookPIN", {notebook: pinNotebook, pin: "4821"})).code === 0);

// Leave the test instance clean.
for (const id of [notebook, pinNotebook]) {
    await raw("/api/notebook/removeNotebook", {notebook: id});
}
finish();
