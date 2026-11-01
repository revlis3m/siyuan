import * as assert from "node:assert/strict";
import {readFileSync, readdirSync} from "node:fs";
import * as path from "node:path";
import {test} from "node:test";
import * as ts from "typescript";
import {AV_KEY_CAPABILITIES, AV_KEY_TYPES} from "./capabilities.generated";
import {
    AV_DATE_TYPES, getAVDefaultFilterOperator, getAVKeyCapability, getAVOfferedFilterOperators, hasAVAttributePlaceholder,
    hasAVCapability, hasAVScalarContent, isAVDateType, isAVLinkType, isAVNewItemTemplateType, isAVReadonlyType,
    isAVRichTextType, isAVSelectType, isAVSingleSelectType, isAVStatusCompletionOperator, isAVTextType,
    isAVTimestampType, isAVValuelessFilterOperator, usesAVRollupCellRenderer,
} from "./capabilities";

test("field capability groups preserve supported field behavior", () => {
    const groups: Array<[string, (type: string) => boolean, string[]]> = [
        ["date", isAVDateType, ["date", "created", "updated"]],
        ["select", isAVSelectType, ["select", "mSelect", "status"]],
        ["single select", isAVSingleSelectType, ["select", "status"]],
        ["text", isAVTextType, ["text", "block", "url", "email", "phone", "template"]],
        ["timestamp", isAVTimestampType, ["created", "updated"]],
        ["readonly", isAVReadonlyType, ["created", "updated", "template", "rollup", "lineNumber"]],
        ["rich text", isAVRichTextType, ["text", "block", "email", "phone", "template"]],
        ["link", isAVLinkType, ["block", "url", "email", "phone"]],
        ["scalar content", hasAVScalarContent, ["number", "text", "block", "url", "phone", "email", "template", "mAsset"]],
        ["attribute placeholder", hasAVAttributePlaceholder, ["text", "number", "date", "url", "phone", "template", "email", "location"]],
        ["rollup cell", usesAVRollupCellRenderer, ["template", "select", "mSelect", "mAsset", "relation", "location", "status"]],
        ["template fields", isAVNewItemTemplateType, ["text", "number", "date", "select", "mSelect", "url", "email", "phone", "mAsset", "checkbox", "relation", "location", "status"]],
    ];
    for (const [name, predicate, expected] of groups) {
        for (const type of [...AV_KEY_TYPES, "unknown", undefined, null]) {
            assert.equal(predicate(type), expected.includes(type), `${name}: ${type}`);
        }
    }
    assert.deepEqual(AV_DATE_TYPES, ["date", "created", "updated"]);
    for (const type of AV_KEY_TYPES) {
        const computed = ["created", "updated", "template", "rollup", "lineNumber"].includes(type);
        assert.equal(hasAVCapability(type, "editable"), !computed, type);
        assert.equal(hasAVCapability(type, "filterable"), type !== "lineNumber", type);
        assert.equal(hasAVCapability(type, "sortable"), type !== "lineNumber", type);
        assert.equal(hasAVCapability(type, "groupable"), type !== "lineNumber" && type !== "rollup" && type !== "location", type);
        const expected = type === "lineNumber" ? undefined : type === "relation" ? "Contains any item" :
            ["select", "number", "date", "created", "updated", "checkbox", "status"].includes(type) ? "=" : "Contains";
        assert.equal(getAVDefaultFilterOperator(type), expected, type);
    }
    assert.equal(getAVDefaultFilterOperator("relation", true), "Contains");
    assert.equal(getAVKeyCapability("unknown"), undefined);
    assert.equal(hasAVCapability(undefined, "editable"), false);
    assert.equal(AV_KEY_TYPES.length, Object.keys(AV_KEY_CAPABILITIES).length);
});

test("status fields offer completion filters only outside rollups", () => {
    const completion = ["Is complete", "Is not complete"];
    assert.deepEqual(getAVOfferedFilterOperators("status"), ["=", "!=", "Is empty", "Is not empty", ...completion]);
    assert.deepEqual(getAVOfferedFilterOperators("status", true), ["=", "!=", "Is empty", "Is not empty"]);
    for (const type of AV_KEY_TYPES.filter(item => item !== "status")) {
        for (const isRollup of [false, true]) {
            assert.ok(!getAVOfferedFilterOperators(type, isRollup).some(isAVStatusCompletionOperator), type);
        }
    }
    for (const operator of [...completion, "Is empty", "Is not empty"]) {
        assert.equal(isAVValuelessFilterOperator(operator), true, operator);
    }
    for (const operator of ["=", "!=", "Contains", "", undefined]) {
        assert.equal(isAVValuelessFilterOperator(operator), false, String(operator));
        assert.equal(isAVStatusCompletionOperator(operator), false, String(operator));
    }
});

test("field membership decisions use declared capabilities", () => {
    const root = __dirname;
    const fields = new Set<string>(AV_KEY_TYPES);
    const violations: string[] = [];
    const inspectDirectory = (dir: string) => {
        for (const entry of readdirSync(dir, {withFileTypes: true})) {
            const file = path.join(dir, entry.name);
            if (entry.isDirectory()) {
                inspectDirectory(file);
                continue;
            }
            if (!file.endsWith(".ts") || file.endsWith(".test.ts") || file.endsWith(".generated.ts")) {
                continue;
            }
            const source = ts.createSourceFile(file, readFileSync(file, "utf8"), ts.ScriptTarget.Latest, true);
            const visit = (node: ts.Node) => {
                if (ts.isArrayLiteralExpression(node) && node.elements.length &&
                    node.elements.every(item => ts.isStringLiteral(item) && fields.has(item.text))) {
                    // 菜单面板标识与字段类型独立，允许其按面板名称判断布局。
                    const menuPanel = entry.name === "openMenuPanel.ts" && node.parent.parent.getText(source).includes("options.type");
                    if (!menuPanel) {
                        violations.push(`${path.relative(root, file)}:${source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1}`);
                    }
                }
                ts.forEachChild(node, visit);
            };
            visit(source);
        }
    };
    inspectDirectory(root);
    assert.deepEqual(violations, []);
});

test("field type predicates read the precomputed index", () => {
    // 渲染按单元格反复判断字段类型，逐次读取能力表对象比集合查找慢一倍以上。
    // 这里锁定这五个常用谓词的实现，避免为了「集中来源」退回逐次属性链。
    const predicates = ["isAVDateType", "isAVTimestampType", "isAVSelectType", "isAVTextType", "isAVReadonlyType"];
    const file = path.join(__dirname, "capabilities.ts");
    const source = ts.createSourceFile(file, readFileSync(file, "utf8"), ts.ScriptTarget.Latest, true);
    const found: string[] = [];
    const visit = (node: ts.Node) => {
        if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && predicates.includes(node.name.text)) {
            found.push(node.name.text);
            const body = node.initializer?.getText(source) || "";
            assert.ok(!body.includes("getAVKeyCapability"),
                `${node.name.text} 不应逐次读取能力表`);
            assert.ok(body.includes(".has("), `${node.name.text} 应查询预计算索引`);
        }
        ts.forEachChild(node, visit);
    };
    visit(source);
    assert.deepEqual(found.sort(), [...predicates].sort());
});
