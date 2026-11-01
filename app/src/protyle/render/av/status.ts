import {escapeHtml} from "../../../util/escape";

type TAVStatusOption = IAVColumn["options"][number];

// 状态分组按待办、进行中、已完成排列，与内核排序和分组顺序一致。
export const AV_STATUS_GROUPS: readonly TAVStatusGroup[] = ["todo", "inProgress", "complete"];

const statusGroupLabelKeys: Record<TAVStatusGroup, string> = {
    todo: "statusGroupTodo",
    inProgress: "statusGroupInProgress",
    complete: "statusGroupComplete",
};

// 缺失或未知的分组按待办处理，兼容没有分组的旧选项。
export const getAVStatusGroup = (option?: { group?: string }): TAVStatusGroup => {
    const group = option?.group as TAVStatusGroup;
    return AV_STATUS_GROUPS.includes(group) ? group : "todo";
};

export const getAVStatusGroupLabel = (group: TAVStatusGroup) =>
    window.siyuan.languages[statusGroupLabelKeys[group]] as string;

// 返回按分组排列的选项，同组内保持字段中的选项顺序。
export const getAVStatusOrderedOptions = <T extends { group?: string }>(options: T[] = []): T[] =>
    AV_STATUS_GROUPS.flatMap(group => options.filter(option => getAVStatusGroup(option) === group));

// 生成分组标题行，标题为只读菜单项，键盘上下选择时会被跳过。
export const getAVStatusGroupHeadingHTML = (group: TAVStatusGroup) =>
    `<div class="b3-menu__item b3-menu__item--readonly" data-type="statusGroup" data-group="${group}">
    <span class="b3-menu__label ft__on-surface">${escapeHtml(getAVStatusGroupLabel(group))}</span>
</div>`;

// 状态字段的选项更新需要携带全部分组，避免内核把缺省分组解释为保留旧值以外的含义。
export const withAVStatusGroups = (options: TAVStatusOption[] = []): TAVStatusOption[] =>
    options.map(option => ({...option, group: getAVStatusGroup(option)}));

// 将选项插入到目标分组末尾，返回新的选项数组，原数组不变。
export const insertAVStatusOption = (options: TAVStatusOption[] = [], option: TAVStatusOption,
                                     group: TAVStatusGroup): TAVStatusOption[] => {
    const result = withAVStatusGroups(options);
    let index = -1;
    result.forEach((item, itemIndex) => {
        if (getAVStatusGroup(item) === group) {
            index = itemIndex;
        }
    });
    result.splice(index > -1 ? index + 1 : result.length, 0, {...option, group});
    return result;
};

// 将已有选项移动到另一个分组末尾；选项不存在或分组未变化时返回 undefined。
export const moveAVStatusOption = (options: TAVStatusOption[] = [], name: string,
                                   group: TAVStatusGroup): TAVStatusOption[] | undefined => {
    const option = options.find(item => item.name === name);
    if (!option || getAVStatusGroup(option) === group) {
        return undefined;
    }
    return insertAVStatusOption(options.filter(item => item.name !== name), option, group);
};
