import * as assert from "node:assert/strict";
import {test} from "node:test";
import {
    AV_STATUS_GROUPS,
    getAVStatusGroup,
    getAVStatusOrderedOptions,
    insertAVStatusOption,
    moveAVStatusOption,
    withAVStatusGroups,
} from "./status";

const options = (): IAVColumn["options"] => [
    {name: "Done", color: "11", group: "complete"},
    {name: "Backlog", color: "5"},
    {name: "Doing", color: "6", group: "inProgress"},
    {name: "Later", color: "1", group: ""},
];

test("status groups default to todo and keep their declared order", () => {
    assert.deepEqual(AV_STATUS_GROUPS, ["todo", "inProgress", "complete"]);
    assert.equal(getAVStatusGroup(undefined), "todo");
    assert.equal(getAVStatusGroup({group: ""}), "todo");
    assert.equal(getAVStatusGroup({group: "unknown"}), "todo");
    assert.equal(getAVStatusGroup({group: "inProgress"}), "inProgress");
    assert.deepEqual(getAVStatusOrderedOptions(options()).map(option => option.name), ["Backlog", "Later", "Doing", "Done"]);
    assert.deepEqual(getAVStatusOrderedOptions(undefined), []);
});

test("status option updates carry explicit groups without mutating the source", () => {
    const source = options();
    const snapshot = JSON.parse(JSON.stringify(source));
    assert.deepEqual(withAVStatusGroups(source).map(option => option.group), ["complete", "todo", "inProgress", "todo"]);

    const inserted = insertAVStatusOption(source, {name: "Review", color: "2"}, "inProgress");
    assert.deepEqual(inserted.map(option => `${option.name}:${option.group}`),
        ["Done:complete", "Backlog:todo", "Doing:inProgress", "Review:inProgress", "Later:todo"]);
    const appended = insertAVStatusOption([], {name: "First", color: "1"}, "complete");
    assert.deepEqual(appended, [{name: "First", color: "1", group: "complete"}]);

    const moved = moveAVStatusOption(source, "Backlog", "complete");
    assert.deepEqual(moved.map(option => `${option.name}:${option.group}`),
        ["Done:complete", "Backlog:complete", "Doing:inProgress", "Later:todo"]);
    assert.equal(moveAVStatusOption(source, "Backlog", "todo"), undefined);
    assert.equal(moveAVStatusOption(source, "Missing", "complete"), undefined);
    assert.deepEqual(source, snapshot);
});
