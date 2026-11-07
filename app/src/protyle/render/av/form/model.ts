/**
 * revlis3m mods: Form view logic without DOM, so it can be tested.
 *
 * The visible fields of the view, in order, are the questions. Each answer is kept as a raw value:
 * text for text-like, number and date (YYYY-MM-DD) fields, an option name for single choices,
 * a list for multiple choices and relations, a boolean for checkboxes.
 */

export type TFormRaw = string | string[] | boolean;

const TEXT_TYPES = ["block", "text", "url", "email", "phone"];
const FORM_TYPES = [...TEXT_TYPES, "number", "date", "select", "status", "mSelect", "checkbox", "relation"];

/** Whether a field type can be answered in a form; computed and file fields are left out. */
export const isFormInputType = (type: string) => FORM_TYPES.includes(type);

/** Fields whose value can decide whether another field is shown. */
export const isFormConditionType = (type: string) => ["select", "status", "checkbox"].includes(type);

export const getFormSettings = (view: Pick<IAVTable, "form">): IAVFormSettings => ({
    title: view.form?.title || "",
    description: view.form?.description || "",
    submitLabel: view.form?.submitLabel || "",
    fields: (view.form?.fields || []).filter(Boolean).map(field => ({
        keyID: field.keyID,
        required: !!field.required,
        default: field.default || "",
        value: field.value || "",
        showWhenKeyID: field.showWhenKeyID || "",
        showWhenValues: field.showWhenValues || [],
    })),
});

export const getFormField = (settings: IAVFormSettings, keyID: string): IAVFormField =>
    settings.fields.find(field => field.keyID === keyID) ||
    {keyID, required: false, default: "", value: "", showWhenKeyID: "", showWhenValues: []};

/** Returns new settings with one field's settings changed; fields back to plain defaults are dropped. */
export const updateFormField = (settings: IAVFormSettings, keyID: string, change: Partial<IAVFormField>): IAVFormSettings => {
    const next = {...getFormField(settings, keyID), ...change, keyID};
    const plain = !next.required && !next.default && !next.showWhenKeyID;
    const fields = settings.fields.filter(field => field.keyID !== keyID);
    if (!plain) {
        const index = settings.fields.findIndex(field => field.keyID === keyID);
        fields.splice(index < 0 ? fields.length : index, 0, next);
    }
    return {...settings, fields};
};

export const isFormRawEmpty = (raw: TFormRaw | undefined) => {
    if (Array.isArray(raw)) {
        return raw.length === 0;
    }
    if (typeof raw === "boolean") {
        return !raw;
    }
    return !raw || !raw.trim();
};

/** The text a condition compares: the option name of a choice, "true" or "false" for a checkbox. */
export const formConditionText = (raw: TFormRaw | undefined) => {
    if (typeof raw === "boolean") {
        return raw ? "true" : "false";
    }
    if (Array.isArray(raw)) {
        return raw[0] || "";
    }
    return raw || "";
};

/** Whether a field is shown, given the current answers. A hidden field is neither required nor submitted. */
export const isFormFieldShown = (settings: IAVFormSettings, keyID: string, answers: Record<string, TFormRaw>,
                                 visibleKeyIDs: string[]) => {
    const field = getFormField(settings, keyID);
    if (!field.showWhenKeyID || !visibleKeyIDs.includes(field.showWhenKeyID)) {
        return true;
    }
    return field.showWhenValues.includes(formConditionText(answers[field.showWhenKeyID]));
};

const pad = (value: number) => String(value).padStart(2, "0");

export const formDateText = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;

/** Raw answer from a database value (used for "last used" defaults). */
export const rawFromCellValue = (type: string, value: IAVCellValue | undefined): TFormRaw | undefined => {
    if (!value) {
        return undefined;
    }
    switch (type) {
        case "block":
            return value.block?.content || "";
        case "text":
            return value.text?.content || "";
        case "url":
            return value.url?.content || "";
        case "email":
            return value.email?.content || "";
        case "phone":
            return value.phone?.content || "";
        case "number":
            return value.number?.isNotEmpty ? String(value.number.content) : "";
        case "date":
            return value.date?.isNotEmpty ? formDateText(new Date(value.date.content)) : "";
        case "select":
        case "status":
            return value.mSelect?.[0]?.content || "";
        case "mSelect":
            return (value.mSelect || []).map(item => item.content);
        case "checkbox":
            return !!value.checkbox?.checked;
        case "relation":
            return [...(value.relation?.blockIDs || [])];
    }
    return undefined;
};

/** Raw answer from the text of a fixed default value (see AVFormField.value). */
export const rawFromDefaultText = (type: string, text: string): TFormRaw => {
    if (type === "checkbox") {
        return text === "true";
    }
    if (type === "mSelect" || type === "relation") {
        return text.split(",").map(item => item.trim()).filter(Boolean);
    }
    return text;
};

/** Text stored as a fixed default value. */
export const defaultTextFromRaw = (raw: TFormRaw | undefined) => {
    if (typeof raw === "boolean") {
        return raw ? "true" : "false";
    }
    if (Array.isArray(raw)) {
        return raw.join(",");
    }
    return raw || "";
};

/** The starting answer of a field: fixed value, today, the latest item's value, or empty. */
export const getFormDefault = (column: Pick<IAVColumn, "id" | "type">, settings: IAVFormSettings,
                               last: Record<string, IAVCellValue> | undefined, today: Date): TFormRaw => {
    const field = getFormField(settings, column.id);
    const empty: TFormRaw = column.type === "checkbox" ? false : (column.type === "mSelect" || column.type === "relation" ? [] : "");
    switch (field.default) {
        case "value":
            return rawFromDefaultText(column.type, field.value);
        case "today":
            return column.type === "date" ? formDateText(today) : empty;
        case "last":
            return rawFromCellValue(column.type, last?.[column.id]) ?? empty;
    }
    return empty;
};

/** Parses a number typed with spaces, commas or dots as thousands or decimal separators. */
export const parseFormNumber = (text: string): number | null => {
    let value = text.replace(/[\s  ']/g, "");
    if (!value) {
        return null;
    }
    const lastComma = value.lastIndexOf(",");
    const lastDot = value.lastIndexOf(".");
    if (lastComma > -1 && lastDot > -1) {
        // The last separator is the decimal one.
        value = lastComma > lastDot ? value.replace(/\./g, "").replace(",", ".") : value.replace(/,/g, "");
    } else if (lastComma > -1) {
        // "1,5" is a decimal, "1,500" or "1,500,000" thousands.
        value = /^-?\d{1,3}(,\d{3})+$/.test(value) ? value.replace(/,/g, "") : value.replace(",", ".");
    } else if (/^-?\d{1,3}(\.\d{3}){2,}$/.test(value)) {
        value = value.replace(/\./g, "");
    }
    const number = Number(value);
    return Number.isFinite(number) ? number : null;
};

/**
 * The database value for one answer, or null when there is nothing to store.
 * Dates are stored at noon local time without a time, so they do not shift across time zones.
 */
export const formCellValue = (column: Pick<IAVColumn, "id" | "type" | "options">, raw: TFormRaw | undefined): IAVCellValue | null => {
    const keyID = column.id;
    switch (column.type) {
        case "block":
            return {keyID, block: {content: typeof raw === "string" ? raw.trim() : ""}};
        case "text":
        case "url":
        case "email":
        case "phone": {
            const content = typeof raw === "string" ? raw.trim() : "";
            return content ? {keyID, [column.type]: {content}} as IAVCellValue : null;
        }
        case "number": {
            const number = typeof raw === "string" ? parseFormNumber(raw) : null;
            return number === null ? null : {keyID, number: {content: number, isNotEmpty: true}};
        }
        case "date": {
            const match = typeof raw === "string" ? /^(\d{4})-(\d{2})-(\d{2})$/.exec(raw) : null;
            if (!match) {
                return null;
            }
            const time = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), 12).getTime();
            return {keyID, date: {content: time, isNotEmpty: true, hasEndDate: false, isNotTime: true, content2: 0, isNotEmpty2: false}};
        }
        case "select":
        case "status":
        case "mSelect": {
            const names = (Array.isArray(raw) ? raw : (typeof raw === "string" && raw ? [raw] : []));
            if (names.length === 0) {
                return null;
            }
            return {
                keyID, mSelect: names.map(name => ({
                    content: name,
                    color: column.options?.find(option => option.name === name)?.color || "1",
                })),
            };
        }
        case "checkbox":
            return raw === true ? {keyID, checkbox: {checked: true}} : null;
        case "relation":
            return Array.isArray(raw) && raw.length > 0 ? {keyID, relation: {blockIDs: raw}} : null;
    }
    return null;
};

/** Required fields that are shown and still empty. */
export const missingRequiredFields = (columns: Pick<IAVColumn, "id">[], settings: IAVFormSettings,
                                      answers: Record<string, TFormRaw>) => {
    const ids = columns.map(column => column.id);
    return columns.filter(column => getFormField(settings, column.id).required &&
        isFormFieldShown(settings, column.id, answers, ids) && isFormRawEmpty(answers[column.id]))
        .map(column => column.id);
};

/** The values of a new item from the shown answers. */
export const buildFormValues = (columns: Pick<IAVColumn, "id" | "type" | "options">[], settings: IAVFormSettings,
                                answers: Record<string, TFormRaw>) => {
    const ids = columns.map(column => column.id);
    const values: IAVCellValue[] = [];
    columns.forEach(column => {
        if (!isFormFieldShown(settings, column.id, answers, ids)) {
            return;
        }
        const value = formCellValue(column, answers[column.id]);
        if (value) {
            values.push(value);
        }
    });
    return values;
};
