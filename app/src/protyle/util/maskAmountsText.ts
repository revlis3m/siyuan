/** revlis3m mods: finding amounts to mask in a text (pure, no DOM). */

// Amount-like numbers: digit groups with spaces, dots or commas as separators, optional decimals.
const NUMBER = /\d+(?:[   .,']\d+)*/g;
// Dates and times are left visible.
const DATE_TIME = /\b\d{4}-\d{1,2}-\d{1,2}\b|\b\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4}\b|\b\d{1,2}:\d{2}(?::\d{2})?\b|\b\d{4}-W\d{1,2}\b|\b\d{4}-Q[1-4]\b/g;

/** Character ranges [start, end) of amount-like numbers in a text, skipping dates and times. */
export const findAmountRanges = (text: string): [number, number][] => {
    const skipped: [number, number][] = [];
    for (const match of text.matchAll(DATE_TIME)) {
        skipped.push([match.index, match.index + match[0].length]);
    }
    const ranges: [number, number][] = [];
    for (const match of text.matchAll(NUMBER)) {
        let value = match[0];
        // A trailing separator belongs to the sentence, not the number ("12, 13").
        while (/[   .,']$/.test(value)) {
            value = value.slice(0, -1);
        }
        const start = match.index;
        const end = start + value.length;
        if (!skipped.some(([from, to]) => start < to && end > from)) {
            ranges.push([start, end]);
        }
    }
    return ranges;
};
