/**
 * Creates a template-strings-like array that satisfies
 * Polymer's html tag validation. Polymer checks
 * Array.isArray(strings) && Array.isArray(strings.raw)
 * && values.length === strings.length - 1. When calling
 * html() as a function (not a tagged template), the
 * array must carry a .raw property.
 *
 * @param {string} content The fully interpolated HTML.
 * @return {!Array<string>} An array with a .raw property.
 */
export function templateContent(content) {
    const strings = [content];
    strings.raw = [content];
    return strings;
}
