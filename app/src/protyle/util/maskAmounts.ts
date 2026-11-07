import {Constants} from "../../constants";
import {setStorageVal} from "./compatibility";
import {getAllEditor} from "../../layout/getAll";
import {findAmountRanges} from "./maskAmountsText";

/**
 * revlis3m mods: masking amounts in marked notebooks.
 *
 * Numbers inside editors of a marked notebook are painted over with a CSS custom highlight, so the document is never
 * modified. Chart values are hidden through their formatters. Every marked notebook starts masked when the app loads;
 * the eye button in the editor toolbar reveals it until the next load. The block being edited stays visible.
 */

const HIGHLIGHT = "sy-mask-amounts";
const revealed = new Set<string>();
const chartRefreshers = new Set<(element: Element) => void>();
let observer: MutationObserver | undefined;
let scheduled = 0;

export const getMaskedNotebooks = (): string[] => {
    const value = window.siyuan.storage?.[Constants.LOCAL_MASK_NOTEBOOKS];
    return Array.isArray(value?.ids) ? value.ids : [];
};

export const isMaskNotebook = (notebookId: string) => !!notebookId && getMaskedNotebooks().includes(notebookId);

export const isNotebookMasked = (notebookId: string) => isMaskNotebook(notebookId) && !revealed.has(notebookId);

/** Whether an element inside an editor currently hides its amounts. */
export const isElementMasked = (element: Element) => {
    const protyleElement = element.closest(".protyle");
    return !!protyleElement && isNotebookMasked(protyleElement.getAttribute("data-mask-notebook") || "");
};

/** Charts register a callback that redraws the charts inside an element. */
export const onMaskChange = (refresh: (element: Element) => void) => {
    chartRefreshers.add(refresh);
};

const supported = () => typeof CSS !== "undefined" && "highlights" in CSS && typeof Highlight !== "undefined";

const editingBlock = () => {
    const selection = document.getSelection();
    const node = selection?.rangeCount ? selection.getRangeAt(0).startContainer : null;
    const element = node instanceof Element ? node : node?.parentElement;
    const block = element?.closest("[data-node-id]");
    // An attribute view row or cell must not reveal the whole database.
    if (!block || block.getAttribute("data-type") === "NodeAttributeView" || block.closest(".av")) {
        return null;
    }
    return element.closest(".protyle-wysiwyg") ? block : null;
};

const paint = () => {
    scheduled = 0;
    if (!supported()) {
        return;
    }
    const ranges: Range[] = [];
    const editing = editingBlock();
    document.querySelectorAll(".protyle[data-mask-notebook]").forEach(protyleElement => {
        if (!isNotebookMasked(protyleElement.getAttribute("data-mask-notebook"))) {
            return;
        }
        protyleElement.querySelectorAll(".protyle-title__input, .protyle-wysiwyg").forEach(root => {
            // Database toolbars (view tabs, "1/3" counters) are interface, not amounts.
            const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
                acceptNode: (node) => node.parentElement?.closest(".av__header") ?
                    NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT,
            });
            let node = walker.nextNode() as Text;
            while (node) {
                const text = node.data;
                if (/\d/.test(text) && !(editing && editing.contains(node))) {
                    for (const [start, end] of findAmountRanges(text)) {
                        const range = new Range();
                        range.setStart(node, start);
                        range.setEnd(node, end);
                        ranges.push(range);
                    }
                }
                node = walker.nextNode() as Text;
            }
        });
    });
    CSS.highlights.set(HIGHLIGHT, new Highlight(...ranges));
};

const schedule = () => {
    if (!scheduled) {
        scheduled = window.setTimeout(paint, 120);
    }
};

const start = () => {
    if (observer || !supported()) {
        return;
    }
    observer = new MutationObserver(schedule);
    observer.observe(document.body, {subtree: true, childList: true, characterData: true});
    document.addEventListener("selectionchange", schedule);
    window.addEventListener("resize", schedule);
};

const updateButton = (protyle: IProtyle) => {
    const button = protyle.breadcrumb?.element.parentElement?.querySelector<HTMLElement>('[data-type="mask-amounts"]');
    if (!button) {
        return;
    }
    const marked = isMaskNotebook(protyle.notebookId);
    button.classList.toggle("fn__none", !marked);
    const masked = isNotebookMasked(protyle.notebookId);
    button.querySelector("use").setAttribute("xlink:href", masked ? "#iconEyeoff" : "#iconEye");
    button.setAttribute("aria-label", masked ? window.siyuan.languages.showAmounts : window.siyuan.languages.hideAmounts);
    button.classList.toggle("block__icon--active", masked);
};

const refreshAll = () => {
    document.querySelectorAll<HTMLElement>(".protyle[data-mask-notebook]").forEach(element => {
        chartRefreshers.forEach(refresh => refresh(element));
    });
    getAllEditor().forEach(editor => updateButton(editor.protyle));
    paint();
};

/** Called when an editor loads a document: tags it with its notebook and updates the eye button. */
export const syncAmountMask = (protyle: IProtyle) => {
    if (!protyle?.element) {
        return;
    }
    if (protyle.notebookId) {
        protyle.element.setAttribute("data-mask-notebook", protyle.notebookId);
    }
    updateButton(protyle);
    if (getMaskedNotebooks().length > 0) {
        start();
        schedule();
    }
};

export const toggleAmountsRevealed = (protyle: IProtyle) => {
    const id = protyle.notebookId;
    if (revealed.has(id)) {
        revealed.delete(id);
    } else {
        revealed.add(id);
    }
    updateButton(protyle);
    refreshAll();
};

export const setMaskNotebook = (notebookId: string, enabled: boolean) => {
    const ids = getMaskedNotebooks().filter(id => id !== notebookId);
    if (enabled) {
        ids.push(notebookId);
    }
    revealed.delete(notebookId);
    window.siyuan.storage[Constants.LOCAL_MASK_NOTEBOOKS] = {ids};
    setStorageVal(Constants.LOCAL_MASK_NOTEBOOKS, {ids});
    if (ids.length > 0) {
        start();
    }
    refreshAll();
};
