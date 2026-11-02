// 此文件由内核字段能力声明生成，请运行 pnpm run api:generate 更新。

import type {AVKeyType, AVKeyCapability, AVFilterProfile, AVFilterCapability, AVCalcOperator} from "../../../types/av";

export const AV_KEY_TYPES: readonly AVKeyType[] = ["block","text","number","date","select","mSelect","url","email","phone","mAsset","template","created","updated","checkbox","relation","rollup","lineNumber","location","status"];

export const AV_KEY_CAPABILITIES: Readonly<Record<AVKeyType, AVKeyCapability>> = {
    "block": {"groups":["richText","link","scalarContent"],"filterProfile":"text","valueKind":"text","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "text": {"groups":["richText","scalarContent","attributePlaceholder","newItemTemplate","chartGroup"],"filterProfile":"text","valueKind":"text","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "number": {"groups":["scalarContent","attributePlaceholder","newItemTemplate","chartGroup","chartValue"],"filterProfile":"number","valueKind":"number","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"="},
    "date": {"groups":["attributePlaceholder","newItemTemplate","chartGroup"],"filterProfile":"date","valueKind":"date","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"="},
    "select": {"groups":["rollupCell","newItemTemplate","chartGroup","chartStack"],"filterProfile":"select","valueKind":"options","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"="},
    "mSelect": {"groups":["rollupCell","newItemTemplate","chartGroup"],"filterProfile":"mSelect","valueKind":"options","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "url": {"groups":["link","scalarContent","attributePlaceholder","newItemTemplate","chartGroup"],"filterProfile":"text","valueKind":"text","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "email": {"groups":["richText","link","scalarContent","attributePlaceholder","newItemTemplate","chartGroup"],"filterProfile":"text","valueKind":"text","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "phone": {"groups":["richText","link","scalarContent","attributePlaceholder","newItemTemplate","chartGroup"],"filterProfile":"text","valueKind":"text","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "mAsset": {"groups":["scalarContent","rollupCell","newItemTemplate","renderDependentFilter"],"filterProfile":"text","valueKind":"assets","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "template": {"groups":["richText","scalarContent","attributePlaceholder","rollupCell","noFilterDefault","renderDependentFilter","renderAutoFill","rollupAlwaysRender","chartGroup","chartValue"],"filterProfile":"template","valueKind":"text","editable":false,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains"},
    "created": {"groups":["noFilterDefault","renderDependentFilter","skipRowCopy","renderAutoFill","rollupForeignRender","chartGroup"],"filterProfile":"date","valueKind":"timestamp","editable":false,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"="},
    "updated": {"groups":["noFilterDefault","renderDependentFilter","skipRowCopy","renderAutoFill","rollupForeignRender","chartGroup"],"filterProfile":"date","valueKind":"timestamp","editable":false,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"="},
    "checkbox": {"groups":["newItemTemplate","chartGroup","chartStack"],"filterProfile":"checkbox","valueKind":"checkbox","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"="},
    "relation": {"groups":["rollupCell","newItemTemplate","rollupForeignRender","chartGroup"],"filterProfile":"relation","valueKind":"relation","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"Contains any item"},
    "rollup": {"groups":["noFilterDefault","renderDependentFilter","skipRowCopy","chartValue"],"filterProfile":"rollup","valueKind":"rollup","editable":false,"filterable":true,"sortable":true,"groupable":false,"defaultOperator":"Contains"},
    "lineNumber": {"groups":["none"],"filterProfile":"none","valueKind":"lineNumber","editable":false,"filterable":false,"sortable":false,"groupable":false,"defaultOperator":""},
    "location": {"groups":["attributePlaceholder","rollupCell","newItemTemplate","noFilterDefault"],"filterProfile":"text","valueKind":"location","editable":true,"filterable":true,"sortable":true,"groupable":false,"defaultOperator":"Contains"},
    "status": {"groups":["rollupCell","newItemTemplate","chartGroup","chartStack"],"filterProfile":"status","valueKind":"options","editable":true,"filterable":true,"sortable":true,"groupable":true,"defaultOperator":"="},
};

export const AV_FILTER_CAPABILITIES: Readonly<Record<AVFilterProfile, AVFilterCapability>> = {
    "checkbox": {"accepted":["=","!=","Is true","Is false"],"offered":["=","!="]},
    "date": {"accepted":["=","\u003e","\u003c","\u003e=","\u003c=","Is between","Is empty","Is not empty"],"offered":["=","\u003e","\u003c","\u003e=","\u003c=","Is between","Is empty","Is not empty"]},
    "mSelect": {"accepted":["Contains","Does not contains","Is empty","Is not empty"],"offered":["Contains","Does not contains","Is empty","Is not empty"]},
    "none": {"accepted":[],"offered":[]},
    "number": {"accepted":["=","!=","\u003e","\u003c","\u003e=","\u003c=","Is empty","Is not empty"],"offered":["=","!=","\u003e","\u003c","\u003e=","\u003c=","Is empty","Is not empty"]},
    "relation": {"accepted":["Contains any item","Does not contain any item","Contains","Does not contains","Is empty","Is not empty"],"offered":["Contains any item","Does not contain any item","Contains","Does not contains","Is empty","Is not empty"],"offeredRollup":["Contains","Does not contains","Is empty","Is not empty"]},
    "rollup": {"accepted":["=","!=","\u003e","\u003e=","\u003c","\u003c=","Contains","Does not contains","Contains any item","Does not contain any item","Is empty","Is not empty","Starts with","Ends with","Is between"],"offered":[]},
    "select": {"accepted":["=","!=","Is empty","Is not empty"],"offered":["=","!=","Is empty","Is not empty"]},
    "status": {"accepted":["=","!=","Is empty","Is not empty","Is complete","Is not complete"],"offered":["=","!=","Is empty","Is not empty","Is complete","Is not complete"]},
    "template": {"accepted":["=","!=","Contains","Does not contains","Starts with","Ends with","Is empty","Is not empty","\u003e","\u003c","\u003e=","\u003c="],"offered":["=","!=","Contains","Does not contains","Starts with","Ends with","Is empty","Is not empty","\u003e","\u003c","\u003e=","\u003c="]},
    "text": {"accepted":["=","!=","Contains","Does not contains","Starts with","Ends with","Is empty","Is not empty"],"offered":["=","!=","Contains","Does not contains","Starts with","Ends with","Is empty","Is not empty"]},
};

export const AV_CALC_FILTER_NUMBER_OPERATORS: readonly AVCalcOperator[] = ["Average","Checked","Count all","Count complete","Count empty","Count not complete","Count not empty","Count unique values","Count values","Max","Median","Min","Percent checked","Percent complete","Percent empty","Percent not empty","Percent unchecked","Percent unique values","Sum","Unchecked"];
