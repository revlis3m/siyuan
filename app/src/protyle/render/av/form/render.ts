/**
 * revlis3m mods: Form view. Shows the view's visible fields as a form; Submit adds an item to the database.
 * Per-field settings (required, default value, show only when…) are in the ⚙ next to each label, the form title,
 * description and button text in the ⚙ of the form header. Unsent answers survive re-renders of the database.
 */
import {Constants} from "../../../../constants";
import {escapeAttr, escapeHtml} from "../../../../util/escape";
import {Menu} from "../../../../plugin/Menu";
import {toggleMenu} from "../../../../menus/menuToggle";
import {fetchSyncPost} from "../../../../util/fetch";
import {showMessage} from "../../../../dialog/message";
import {openInputDialog} from "../../../../dialog/inputDialog";
import {transaction} from "../../../wysiwyg/transaction";
import {avRender, genTabHeaderHTML} from "../render";
import {replaceAVContainer} from "../container";
import {setAVData} from "../virtualScroll";
import {finishAVLocate} from "../locate";
import {
    buildFormValues,
    defaultTextFromRaw,
    getFormDefault,
    getFormField,
    getFormSettings,
    isFormConditionType,
    isFormFieldShown,
    isFormInputType,
    missingRequiredFields,
    TFormRaw,
    updateFormField,
} from "./model";

const drafts = new Map<string, Record<string, TFormRaw>>();
// When each form last added an item, so "Added" survives the re-render that follows.
const addedAt = new Map<string, number>();
const relationCache = new Map<string, Promise<{ id: string, name: string }[]>>();

export const canEditForm = (protyle: IProtyle) => !protyle.disabled && !window.siyuan.isPublish &&
    !protyle.options.history?.created && !protyle.options.history?.snapshot;

const draftKey = (blockElement: Element, data: IAV) => `${blockElement.getAttribute("data-node-id")}:${data.viewID}`;

const loadRelationItems = (avID: string) => {
    if (!relationCache.has(avID)) {
        relationCache.set(avID, fetchSyncPost("/api/av/getAttributeViewPrimaryKeyValues", {id: avID, pageSize: 1000})
            .then(response => (response.code === 0 ? response.data?.rows?.values || [] : [])
                .map((value: IAVCellValue) => ({id: value.blockID, name: value.block?.content || window.siyuan.languages.untitled}))
                .sort((a: { name: string }, b: { name: string }) => a.name.localeCompare(b.name)))
            .catch((): { id: string, name: string }[] => []));
        // Items change: keep the list for a short while only.
        setTimeout(() => relationCache.delete(avID), 30000);
    }
    return relationCache.get(avID);
};

const optionsOf = (column: IAVColumn) => (column.options || []).map(option => option.name);

const inputHTML = (column: IAVColumn, raw: TFormRaw) => {
    const name = escapeAttr(column.name || "");
    switch (column.type) {
        case "number":
            return `<input class="b3-text-field fn__block" data-form-input="number" inputmode="decimal" aria-label="${name}" value="${escapeAttr(raw as string)}">`;
        case "date":
            return `<input class="b3-text-field fn__block" data-form-input="date" type="date" aria-label="${name}" value="${escapeAttr(raw as string)}">`;
        case "select":
        case "status":
            return `<select class="b3-select fn__block" data-form-input="select" aria-label="${name}">
    <option value=""></option>
    ${optionsOf(column).map(option => `<option value="${escapeAttr(option)}"${option === raw ? " selected" : ""}>${escapeHtml(option)}</option>`).join("")}
</select>`;
        case "mSelect":
            return `<div class="av__form-choices" data-form-input="mSelect">${optionsOf(column).map(option => `<label class="av__form-choice">
    <input type="checkbox" value="${escapeAttr(option)}"${(raw as string[]).includes(option) ? " checked" : ""}> ${escapeHtml(option)}
</label>`).join("")}</div>`;
        case "checkbox":
            return `<label class="av__form-choice"><input type="checkbox" data-form-input="checkbox"${raw ? " checked" : ""}> ${escapeHtml(column.name || "")}</label>`;
        case "relation":
            return `<select class="b3-select fn__block" data-form-input="relation" aria-label="${name}" data-av-id="${escapeAttr(column.relation?.avID || "")}">
    <option value=""></option>
    ${(raw as string[]).map(id => `<option value="${escapeAttr(id)}" selected>…</option>`).join("")}
</select>`;
        case "block":
        case "text":
            return `<input class="b3-text-field fn__block" data-form-input="text" aria-label="${name}" value="${escapeAttr(raw as string)}">`;
        default:
            return `<input class="b3-text-field fn__block" data-form-input="text" type="${column.type === "email" ? "email" : (column.type === "url" ? "url" : (column.type === "phone" ? "tel" : "text"))}" aria-label="${name}" value="${escapeAttr(raw as string)}">`;
    }
};

const readInput = (fieldElement: Element, type: string): TFormRaw => {
    if (type === "mSelect") {
        return Array.from(fieldElement.querySelectorAll<HTMLInputElement>(".av__form-choices input:checked")).map(item => item.value);
    }
    if (type === "checkbox") {
        return !!fieldElement.querySelector<HTMLInputElement>('[data-form-input="checkbox"]')?.checked;
    }
    const element = fieldElement.querySelector<HTMLInputElement | HTMLSelectElement>("[data-form-input]");
    if (type === "relation") {
        return element?.value ? [element.value] : [];
    }
    return element?.value || "";
};

export const setFormSettings = (protyle: IProtyle, blockElement: Element, data: IAV, next: IAVFormSettings) => {
    const view = data.view as IAVTable;
    const previous = getFormSettings(view);
    if (JSON.stringify(previous) === JSON.stringify(next)) {
        return;
    }
    const operation = {
        action: "setAttrViewForm" as const,
        avID: data.id,
        blockID: blockElement.getAttribute("data-node-id"),
        viewID: data.viewID,
    };
    transaction(protyle, [{...operation, data: next}], [{...operation, data: previous}]);
    view.form = next;
};

const openHeaderMenu = (protyle: IProtyle, blockElement: HTMLElement, data: IAV, target: HTMLElement) => {
    const settings = getFormSettings(data.view as IAVTable);
    const edit = (label: string, key: "title" | "description" | "submitLabel", multiline = false) => ({
        icon: "iconEdit",
        label,
        click: () => {
            openInputDialog({
                title: label,
                value: settings[key],
                multiline,
                onConfirm: (value, dialog) => {
                    dialog.destroy();
                    setFormSettings(protyle, blockElement, data, {...settings, [key]: value.trim()});
                },
            });
        },
    });
    toggleMenu({target, toggle: false, build: () => {
        const menu = new Menu();
        menu.addItem(edit(window.siyuan.languages.formTitle, "title"));
        menu.addItem(edit(window.siyuan.languages.formDescription, "description", true));
        menu.addItem(edit(window.siyuan.languages.formSubmitLabel, "submitLabel"));
        menu.addSeparator();
        menu.addItem({type: "readonly", iconHTML: "", label: `<span class="ft__on-surface ft__smaller">${escapeHtml(window.siyuan.languages.formFieldsTip)}</span>`});
        const rect = target.getBoundingClientRect();
        menu.open({x: rect.left, y: rect.bottom, h: rect.height});
    }});
};

const openFieldMenu = (protyle: IProtyle, blockElement: HTMLElement, data: IAV, column: IAVColumn, target: HTMLElement,
                       currentRaw: TFormRaw) => {
    const view = data.view as IAVTable;
    const settings = getFormSettings(view);
    const field = getFormField(settings, column.id);
    const save = (change: Partial<IAVFormField>) => setFormSettings(protyle, blockElement, data, updateFormField(settings, column.id, change));
    const lang = window.siyuan.languages;
    const defaults: IMenu[] = [
        {label: lang.formDefaultNone, checked: !field.default, click: () => save({default: "", value: ""})},
        {label: lang.formDefaultCurrent, checked: field.default === "value", click: () => save({default: "value", value: defaultTextFromRaw(currentRaw)})},
        {label: lang.formDefaultLast, checked: field.default === "last", click: () => save({default: "last", value: ""})},
    ];
    if (column.type === "date") {
        defaults.splice(1, 0, {label: lang.formDefaultToday, checked: field.default === "today", click: () => save({default: "today", value: ""})});
    }
    const conditionFields = view.columns.filter(item => !item.hidden && item.id !== column.id && isFormConditionType(item.type));
    const conditions: IMenu[] = [{
        label: lang.formShowAlways, checked: !field.showWhenKeyID, click: () => save({showWhenKeyID: "", showWhenValues: []}),
    }];
    conditionFields.forEach(item => {
        const values = item.type === "checkbox" ? ["true", "false"] : optionsOf(item);
        conditions.push({
            label: escapeHtml(item.name || ""),
            type: "submenu",
            checked: field.showWhenKeyID === item.id,
            submenu: values.map(value => {
                const selected = field.showWhenKeyID === item.id && field.showWhenValues.includes(value);
                return {
                    label: escapeHtml(item.type === "checkbox" ? (value === "true" ? lang.formChecked : lang.formUnchecked) : value),
                    checked: selected,
                    click: () => {
                        const current = field.showWhenKeyID === item.id ? field.showWhenValues : [];
                        const next = selected ? current.filter(v => v !== value) : [...current, value];
                        save(next.length ? {showWhenKeyID: item.id, showWhenValues: next} : {showWhenKeyID: "", showWhenValues: []});
                    },
                };
            }),
        });
    });
    toggleMenu({target, toggle: false, build: () => {
        const menu = new Menu();
        menu.addItem({type: "readonly", iconHTML: "", label: `<b>${escapeHtml(column.name || "")}</b>`});
        menu.addItem({icon: "iconInfo", label: lang.formRequired, checked: field.required, click: () => save({required: !field.required})});
        menu.addItem({icon: "iconEdit", label: lang.formDefault, type: "submenu", submenu: defaults});
        menu.addItem({icon: "iconEye", label: lang.formShowWhen, type: "submenu", submenu: conditions});
        const rect = target.getBoundingClientRect();
        menu.open({x: rect.left, y: rect.bottom, h: rect.height});
    }});
};

export const renderForm = async (blockElement: HTMLElement, protyle: IProtyle, data: IAV, cb?: (data: IAV) => void) => {
    const view = data.view as IAVTable;
    const settings = getFormSettings(view);
    const editable = canEditForm(protyle);
    const lang = window.siyuan.languages;
    const columns = view.columns.filter(column => !column.hidden && isFormInputType(column.type));
    const skipped = view.columns.filter(column => !column.hidden && !isFormInputType(column.type));
    const key = draftKey(blockElement, data);
    const today = new Date();
    const defaults = () => Object.fromEntries(columns.map(column => [column.id, getFormDefault(column, settings, view.formLast, today)]));
    const answers: Record<string, TFormRaw> = {...defaults(), ...(drafts.get(key) || {})};

    // Keep the focused field and caret across re-renders.
    const active = document.activeElement as HTMLInputElement;
    const focusKey = blockElement.contains(active) ? active.closest("[data-form-key]")?.getAttribute("data-form-key") : "";
    const caret = focusKey && typeof active.selectionStart === "number" ? active.selectionStart : null;

    const fieldsHTML = columns.map(column => {
        const field = getFormField(settings, column.id);
        const desc = column.desc ? `<div class="av__form-desc">${escapeHtml(column.desc)}</div>` : "";
        return `<div class="av__form-field" data-form-key="${escapeAttr(column.id)}" data-form-type="${column.type}">
    <div class="av__form-label">
        <span>${escapeHtml(column.name || "")}${field.required ? '<span class="av__form-required">*</span>' : ""}</span>
        ${field.showWhenKeyID ? `<svg class="av__form-flag ariaLabel" aria-label="${escapeAttr(lang.formShowWhen)}"><use xlink:href="#iconEye"></use></svg>` : ""}
        ${editable ? `<span class="block__icon block__icon--show ariaLabel" data-type="form-field-menu" aria-label="${escapeAttr(lang.formFieldSettings)}"><svg><use xlink:href="#iconSettings"></use></svg></span>` : ""}
    </div>
    ${desc}
    ${inputHTML(column, answers[column.id])}
</div>`;
    }).join("");
    const skippedHTML = editable && skipped.length ? `<div class="av__form-desc">${escapeHtml(lang.formSkippedFields.replace("${x}", skipped.map(column => column.name).join(", ")))}</div>` : "";
    replaceAVContainer(blockElement, `<div class="av__container fn__block">
    ${genTabHeaderHTML(data, false, editable, blockElement, editable)}
    <form class="av__form" contenteditable="false" novalidate>
        <div class="av__form-head">
            <div class="fn__flex-1">
                <div class="av__form-title">${escapeHtml(settings.title || data.view.name || lang.formView)}</div>
                ${settings.description ? `<div class="av__form-desc">${escapeHtml(settings.description)}</div>` : ""}
            </div>
            ${editable ? `<span class="block__icon block__icon--show ariaLabel" data-type="form-settings" aria-label="${escapeAttr(lang.formSettings)}"><svg><use xlink:href="#iconSettings"></use></svg></span>` : ""}
        </div>
        ${fieldsHTML}
        ${skippedHTML}
        <div class="av__form-actions">
            <button class="b3-button" type="submit"${editable ? "" : " disabled"}>${escapeHtml(settings.submitLabel || lang.formSubmit)}</button>
            <button class="b3-button b3-button--cancel" type="button" data-type="form-reset">${escapeHtml(lang.formReset)}</button>
            <span class="av__form-status">${Date.now() - (addedAt.get(key) || 0) < 3000 ? escapeHtml(lang.formAdded) : ""}</span>
        </div>
    </form>
    <div class="av__cursor" contenteditable="true">${Constants.ZWSP}</div>
</div>`);
    blockElement.dataset.render = "true";
    setAVData(blockElement, data);
    const root = blockElement.querySelector<HTMLFormElement>(".av__form");
    const ids = columns.map(column => column.id);

    const applyConditions = () => {
        root.querySelectorAll<HTMLElement>(".av__form-field").forEach(item => {
            item.classList.toggle("fn__none", !isFormFieldShown(settings, item.dataset.formKey, answers, ids));
        });
    };
    applyConditions();

    // Relation pickers: load the items of the linked database.
    root.querySelectorAll<HTMLSelectElement>('select[data-form-input="relation"]').forEach(select => {
        const avID = select.dataset.avId;
        if (!avID) {
            return;
        }
        void loadRelationItems(avID).then(items => {
            if (!select.isConnected) {
                return;
            }
            const current = (answers[select.closest<HTMLElement>("[data-form-key]").dataset.formKey] as string[])[0] || "";
            select.innerHTML = "<option value=\"\"></option>" + items.map(item =>
                `<option value="${escapeAttr(item.id)}"${item.id === current ? " selected" : ""}>${escapeHtml(item.name)}</option>`).join("");
        });
    });

    const update = (fieldElement: HTMLElement) => {
        const id = fieldElement.dataset.formKey;
        answers[id] = readInput(fieldElement, fieldElement.dataset.formType);
        fieldElement.classList.remove("av__form-field--error");
        drafts.set(key, {...answers});
        applyConditions();
    };
    root.addEventListener("input", event => {
        const fieldElement = (event.target as HTMLElement).closest<HTMLElement>(".av__form-field");
        if (fieldElement) {
            update(fieldElement);
        }
    });
    root.addEventListener("change", event => {
        const fieldElement = (event.target as HTMLElement).closest<HTMLElement>(".av__form-field");
        if (fieldElement) {
            update(fieldElement);
        }
    });
    root.addEventListener("keydown", event => {
        // Typing in the form must not reach the editor; Enter in a single-line field submits.
        event.stopPropagation();
        if (event.key === "Enter" && (event.target as HTMLElement).tagName === "INPUT" && !event.isComposing &&
            (event.ctrlKey || event.metaKey || (event.target as HTMLInputElement).type !== "checkbox")) {
            event.preventDefault();
            root.requestSubmit();
        }
    });
    root.addEventListener("click", event => {
        event.stopPropagation();
        const target = event.target as HTMLElement;
        const menuButton = target.closest<HTMLElement>('[data-type="form-field-menu"]');
        if (menuButton) {
            const fieldElement = menuButton.closest<HTMLElement>(".av__form-field");
            const column = columns.find(item => item.id === fieldElement.dataset.formKey);
            openFieldMenu(protyle, blockElement, data, column, menuButton, answers[column.id]);
            return;
        }
        const settingsButton = target.closest<HTMLElement>('[data-type="form-settings"]');
        if (settingsButton) {
            openHeaderMenu(protyle, blockElement, data, settingsButton);
            return;
        }
        if (target.closest('[data-type="form-reset"]')) {
            drafts.delete(key);
            blockElement.removeAttribute("data-render");
            void avRender(blockElement, protyle);
        }
    });
    root.addEventListener("submit", async event => {
        event.preventDefault();
        if (!editable) {
            return;
        }
        const missing = missingRequiredFields(columns, settings, answers);
        if (missing.length) {
            missing.forEach(id => root.querySelector(`[data-form-key="${CSS.escape(id)}"]`)?.classList.add("av__form-field--error"));
            root.querySelector<HTMLElement>(".av__form-field--error [data-form-input], .av__form-field--error input")?.focus();
            showMessage(lang.formMissing);
            return;
        }
        const button = root.querySelector<HTMLButtonElement>('button[type="submit"]');
        button.disabled = true;
        const values = buildFormValues(columns, settings, answers);
        const response = await fetchSyncPost("/api/av/appendAttributeViewDetachedBlocksWithValues", {
            avID: data.id,
            blocksValues: [values as never],
        });
        button.disabled = false;
        if (response.code !== 0) {
            return;
        }
        // "Last used" defaults take the values just sent.
        const last = {...(view.formLast || {})};
        values.forEach(value => {
            last[value.keyID] = value;
        });
        view.formLast = last;
        drafts.delete(key);
        const fresh = Object.fromEntries(columns.map(column => [column.id, getFormDefault(column, settings, last, new Date())]));
        Object.keys(answers).forEach(id => delete answers[id]);
        Object.assign(answers, fresh);
        root.querySelectorAll<HTMLElement>(".av__form-field").forEach(fieldElement => {
            const column = columns.find(item => item.id === fieldElement.dataset.formKey);
            const holder = document.createElement("div");
            holder.innerHTML = inputHTML(column, answers[column.id]);
            fieldElement.querySelector(":scope > [data-form-input], :scope > .av__form-choices, :scope > label.av__form-choice")
                ?.replaceWith(holder.firstElementChild);
            fieldElement.classList.remove("av__form-field--error");
        });
        root.querySelectorAll<HTMLSelectElement>('select[data-form-input="relation"]').forEach(select => {
            const fieldKey = select.closest<HTMLElement>("[data-form-key]").dataset.formKey;
            void loadRelationItems(select.dataset.avId).then(items => {
                const current = (answers[fieldKey] as string[])[0] || "";
                select.innerHTML = "<option value=\"\"></option>" + items.map(item =>
                    `<option value="${escapeAttr(item.id)}"${item.id === current ? " selected" : ""}>${escapeHtml(item.name)}</option>`).join("");
            });
        });
        applyConditions();
        addedAt.set(key, Date.now());
        root.querySelector(".av__form-status").textContent = lang.formAdded;
        setTimeout(() => {
            const status = blockElement.querySelector(".av__form-status");
            if (status && Date.now() - (addedAt.get(key) || 0) >= 3000) {
                status.textContent = "";
            }
        }, 3100);
        root.querySelector<HTMLElement>(".av__form-field:not(.fn__none) [data-form-input], .av__form-field:not(.fn__none) input")?.focus();
    });

    if (focusKey) {
        const input = root.querySelector<HTMLInputElement>(`[data-form-key="${CSS.escape(focusKey)}"] [data-form-input], [data-form-key="${CSS.escape(focusKey)}"] input`);
        input?.focus();
        if (input && caret !== null && typeof input.setSelectionRange === "function" && input.type !== "date") {
            try {
                input.setSelectionRange(caret, caret);
            } catch {
                // Some input types have no caret.
            }
        }
    }
    finishAVLocate(blockElement, protyle, data);
    cb?.(data);
};
