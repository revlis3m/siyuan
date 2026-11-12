/**
 * revlis3m mods: PIN lock for chosen notebooks.
 *
 * A locked notebook is a closed notebook that the kernel only opens after /api/notebook/unlockNotebookPIN. It locks
 * again (closes) after the idle delay without activity in that notebook, or with "Lock now". It is a privacy lock,
 * not encryption.
 */
import {openInputDialog} from "../dialog/inputDialog";
import {showMessage} from "../dialog/message";
import {fetchPost, fetchSyncPost} from "./fetch";
import {escapeHtml} from "./escape";
import {getFileTreeIconHTML} from "../emoji/fileTreeIcon";
import {getAllEditor} from "../layout/getAll";

const protectedNotebooks = new Set<string>();
let minutes = 15;
let loaded: Promise<void> | undefined;

// Closed notebooks in the document tree: a lock icon for those with a PIN.
const decorateClosedNotebooks = () => {
    document.querySelectorAll<HTMLElement>("li.b3-list-item[data-url]").forEach(item => {
        if (!item.querySelector(':scope > [data-type="open"]') || item.getAttribute("data-encrypted") === "true") {
            return;
        }
        const pin = protectedNotebooks.has(item.getAttribute("data-url"));
        if (pin === item.hasAttribute("data-pin")) {
            return;
        }
        item.toggleAttribute("data-pin", pin);
        const icon = item.querySelector<HTMLElement>(":scope > .b3-list-item__icon");
        if (icon) {
            if (pin) {
                icon.setAttribute("data-pin-icon", icon.innerHTML);
                icon.innerHTML = getFileTreeIconHTML("", "lock");
            } else if (icon.hasAttribute("data-pin-icon")) {
                icon.innerHTML = icon.getAttribute("data-pin-icon");
                icon.removeAttribute("data-pin-icon");
            }
        }
    });
};

export const refreshNotebookPINs = () => {
    loaded = fetchSyncPost("/api/notebook/getNotebookPINStatus", {}).then(response => {
        if (response.code !== 0) {
            return;
        }
        protectedNotebooks.clear();
        (response.data.notebooks || []).forEach((item: { notebook: string }) => protectedNotebooks.add(item.notebook));
        minutes = response.data.minutes;
        decorateClosedNotebooks();
    });
    return loaded;
};

export const notebookPINsLoaded = () => loaded || refreshNotebookPINs();

export const isPINNotebook = (notebookId: string) => protectedNotebooks.has(notebookId);

/** Asks the PIN, then opens the notebook. */
export const openPINNotebook = (notebookId: string, name: string) => {
    const dialogKey = "notebookPIN-" + notebookId;
    if (window.siyuan.dialogs.some(item => item.element.getAttribute("data-key") === dialogKey)) {
        return;
    }
    const dialog = openInputDialog({
        title: window.siyuan.languages.notebookPINUnlock.replace("${x}", escapeHtml(name)),
        value: "",
        type: "password",
        placeholder: window.siyuan.languages.notebookPIN,
        onConfirm: async (pin, dialog) => {
            const input = dialog.element.querySelector<HTMLInputElement>("[data-dialog-input]");
            const confirm = dialog.element.querySelector<HTMLButtonElement>("[data-input-confirm]");
            if (!pin) {
                return false;
            }
            confirm.disabled = true;
            const response = await fetchSyncPost("/api/notebook/unlockNotebookPIN", {notebook: notebookId, pin});
            if (response.code !== 0) {
                confirm.disabled = false;
                input.value = "";
                input.focus();
                return false;
            }
            dialog.destroy();
            fetchPost("/api/notebook/openNotebook", {notebook: notebookId});
        },
    });
    dialog.element.setAttribute("data-key", dialogKey);
};

const field = (id: string, label: string, type = "password", value = "") =>
    `<label class="fn__flex b3-label config__item" style="padding: 8px 0 0;align-items:center">
    <span class="fn__flex-1">${label}</span>
    <input class="b3-text-field fn__size200" data-pin-field="${id}" type="${type}" value="${value}"${type === "number" ? ' min="0" step="1"' : ""}>
</label>`;

/** Sets or changes the PIN of a notebook, and the idle minutes before notebooks lock again. */
export const setNotebookPIN = async (notebookId: string, name: string) => {
    await notebookPINsLoaded();
    const changing = isPINNotebook(notebookId);
    openInputDialog({
        title: (changing ? window.siyuan.languages.notebookPINChange : window.siyuan.languages.notebookPINLock)
            .replace("…", "") + " · " + escapeHtml(name),
        value: "",
        type: "password",
        label: window.siyuan.languages.notebookPINNew,
        description: window.siyuan.languages.notebookPINTip,
        prefixContent: changing ? field("current", window.siyuan.languages.notebookPINCurrent) : "",
        extraContent: field("confirm", window.siyuan.languages.notebookPINConfirm) +
            field("minutes", window.siyuan.languages.notebookPINAutoLock, "number", String(minutes)),
        onConfirm: async (pin, dialog) => {
            const value = (id: string) => dialog.element.querySelector<HTMLInputElement>(`[data-pin-field="${id}"]`)?.value || "";
            if (pin.length < 4) {
                showMessage(window.siyuan.languages.notebookPINShort);
                return false;
            }
            if (pin !== value("confirm")) {
                showMessage(window.siyuan.languages.notebookPINMismatch);
                return false;
            }
            const response = await fetchSyncPost("/api/notebook/setNotebookPIN", {
                notebook: notebookId, pin, currentPIN: value("current"),
            });
            if (response.code !== 0) {
                return false;
            }
            const nextMinutes = Math.max(0, Math.floor(Number(value("minutes")) || 0));
            if (nextMinutes !== minutes) {
                // Changing the delay needs a PIN: the one just set for this notebook.
                await fetchSyncPost("/api/notebook/setNotebookPINMinutes", {notebook: notebookId, pin, minutes: nextMinutes});
            }
            dialog.destroy();
            await refreshNotebookPINs();
            showMessage(window.siyuan.languages.notebookPINSet.replace("${x}", String(nextMinutes)));
        },
    });
};

export const removeNotebookPIN = (notebookId: string, name: string) => {
    openInputDialog({
        title: window.siyuan.languages.notebookPINRemove.replace("…", "") + " · " + escapeHtml(name),
        value: "",
        type: "password",
        placeholder: window.siyuan.languages.notebookPINCurrent,
        onConfirm: async (pin, dialog) => {
            const response = await fetchSyncPost("/api/notebook/removeNotebookPIN", {notebook: notebookId, pin});
            if (response.code !== 0) {
                return false;
            }
            dialog.destroy();
            await refreshNotebookPINs();
        },
    });
};

const lastActivity = new Map<string, number>();

/**
 * Tells the kernel that the user is active in a PIN notebook (in one of its pages or in its part of the document
 * tree), which postpones that notebook's relock. Activity anywhere else does not count.
 */
export const touchNotebookPINFromEvent = (event: Event) => {
    if (window.siyuan.isPublish) {
        return;
    }
    if (!loaded) {
        refreshNotebookPINs();
        return;
    }
    const target = event.target as HTMLElement;
    if (!protectedNotebooks.size || !target || typeof target.closest !== "function") {
        return;
    }
    let notebookId = target.closest("ul[data-url]")?.getAttribute("data-url") || "";
    if (!notebookId) {
        const element = target.closest(".protyle");
        if (element) {
            notebookId = getAllEditor().find(editor => editor.protyle.element === element)?.protyle.notebookId ||
                element.getAttribute("data-notebook-id") || "";
        }
    }
    if (!notebookId || !protectedNotebooks.has(notebookId)) {
        return;
    }
    const now = Date.now();
    if (now - (lastActivity.get(notebookId) || 0) < 30000) {
        return;
    }
    lastActivity.set(notebookId, now);
    fetchPost("/api/notebook/touchNotebookPIN", {notebook: notebookId});
};

export const lockNotebookNow = (notebookId: string) => {
    fetchPost("/api/notebook/lockNotebookPIN", {notebook: notebookId});
};

/** Re-applies the lock icons after the document tree re-renders its closed notebooks. */
export const decorateNotebookPINs = () => {
    if (loaded) {
        decorateClosedNotebooks();
    }
};
