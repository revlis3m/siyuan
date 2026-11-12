// Helpers for the mods checks: SiYuan kernel API calls against a test instance.
import {readFileSync} from "node:fs";

export const BASE = process.env.SIYUAN_URL || "http://127.0.0.1:6806";
export const WORKSPACE = process.env.SIYUAN_WORKSPACE || "";
export const ACCESS_CODE = process.env.SIYUAN_ACCESS_CODE || "test1234";

let token = process.env.SIYUAN_TOKEN || "";
export const apiToken = () => {
    if (!token) {
        token = JSON.parse(readFileSync(`${WORKSPACE}/conf/conf.json`, "utf8")).api.token;
    }
    return token;
};

// Raw call: returns the whole response envelope {code, msg, data}.
export const raw = async (path, body = {}) => {
    const response = await fetch(BASE + path, {
        method: "POST",
        headers: {"Authorization": "Token " + apiToken(), "Content-Type": "application/json"},
        body: JSON.stringify(body),
    });
    const text = await response.text();
    try {
        return JSON.parse(text);
    } catch {
        return {code: response.status, msg: text.slice(0, 200), data: null};
    }
};

// Call that must succeed: returns data, throws with the kernel's message otherwise.
export const api = async (path, body = {}) => {
    const result = await raw(path, body);
    if (result.code !== 0) {
        throw new Error(`${path} failed (${result.code}): ${result.msg}`);
    }
    return result.data;
};

export const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));

// A new notebook is not ready for writes right away: retry for a few seconds.
export const retry = async (fn, tries = 20, wait = 500) => {
    for (let i = 0; ; i++) {
        try {
            return await fn();
        } catch (error) {
            if (i >= tries - 1) throw error;
            await sleep(wait);
        }
    }
};

export const newID = () => {
    const now = new Date();
    const pad = (n, l = 2) => String(n).padStart(l, "0");
    const stamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
    return `${stamp}-${Math.random().toString(36).slice(2, 9).padEnd(7, "0")}`;
};

export const createNotebook = async (name) => (await api("/api/notebook/createNotebook", {name})).notebook.id;

// Creates a document holding one database and returns {docID, avID, blockID}.
export const createDatabaseDoc = async (notebook, path) => {
    const avID = newID();
    const docID = await retry(() => api("/api/filetree/createDocWithMd", {
        notebook, path, markdown: `<div data-type="NodeAttributeView" data-av-id="${avID}" data-av-type="table"></div>`,
    }));
    await api("/api/av/renderAttributeView", {id: avID});
    // The block index is written a moment after the document.
    const blockID = await retry(async () => {
        const blocks = await api("/api/query/sql", {stmt: `SELECT id FROM blocks WHERE root_id='${docID}' AND type='av'`});
        if (!blocks[0]?.id) throw new Error("database block not indexed yet");
        return blocks[0].id;
    });
    return {docID, avID, blockID};
};

export const renderAV = (avID, viewID) => api("/api/av/renderAttributeView", {id: avID, viewID});

// Test bookkeeping.
let failed = 0, passed = 0;
export const check = (name, ok, detail = "") => {
    if (ok) {
        passed++;
        console.log(`  ok    ${name}`);
    } else {
        failed++;
        console.log(`  FAIL  ${name}${detail ? " — " + detail : ""}`);
    }
};
export const finish = () => {
    console.log(`\n${passed} passed, ${failed} failed`);
    process.exit(failed ? 1 : 0);
};
