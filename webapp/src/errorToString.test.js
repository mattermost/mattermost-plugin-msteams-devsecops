// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const fs = require('fs');
const path = require('path');

// Tests for the errorToString utility used in assets/iframe.html.tmpl.
// The function is inlined in the template (no module system); keep both in sync.
function errorToString(error) {
    if (!error) {
        return 'Unknown error';
    }
    if (typeof error === 'string') {
        return error;
    }
    if (error.message) {
        return String(error.message);
    }
    try {
        return String(JSON.stringify(error));
    } catch (_) {
        return String(error);
    }
}

describe('errorToString', () => {
    describe('falsy inputs', () => {
        test('null returns Unknown error', () => {
            expect(errorToString(null)).toBe('Unknown error');
        });

        test('undefined returns Unknown error', () => {
            expect(errorToString(undefined)).toBe('Unknown error');
        });

        test('false returns Unknown error', () => {
            expect(errorToString(false)).toBe('Unknown error');
        });

        test('0 returns Unknown error', () => {
            expect(errorToString(0)).toBe('Unknown error');
        });

        test('empty string returns Unknown error', () => {
            expect(errorToString('')).toBe('Unknown error');
        });
    });

    describe('string inputs', () => {
        test('plain string is returned as-is', () => {
            expect(errorToString('something went wrong')).toBe('something went wrong');
        });

        test('Teams SDK error codes like CancelledByUser are returned as-is', () => {
            expect(errorToString('CancelledByUser')).toBe('CancelledByUser');
        });
    });

    describe('Error objects', () => {
        test('Error instance returns its message', () => {
            expect(errorToString(new Error('test error'))).toBe('test error');
        });

        test('Error with empty message falls through to JSON.stringify', () => {
            const e = new Error('');

            // Error.message is '' which is falsy, so falls through to JSON.stringify.
            // JSON.stringify(new Error('')) returns '{}'.
            expect(errorToString(e)).toBe('{}');
        });
    });

    describe('plain objects', () => {
        test('object with message property returns the message', () => {
            expect(errorToString({message: 'auth failed', code: 42})).toBe('auth failed');
        });

        test('object without message is JSON-serialised', () => {
            expect(errorToString({code: 'TOKEN_EXPIRED'})).toBe('{"code":"TOKEN_EXPIRED"}');
        });

        test('empty object returns {}', () => {
            expect(errorToString({})).toBe('{}');
        });
    });

    describe('non-serialisable objects', () => {
        test('circular-reference object falls back to String()', () => {
            const circular = {};
            circular.self = circular;

            // JSON.stringify throws; String({}) => '[object Object]'
            expect(errorToString(circular)).toBe('[object Object]');
        });
    });
});

describe('sync with assets/iframe.html.tmpl', () => {
    // Collapse all whitespace so indentation differences between the template
    // (6-space indent) and this file (4-space indent) do not cause false failures.
    function normalize(src) {
        return src.replace(/\s+/g, ' ').trim();
    }

    test('errorToString body matches the copy inlined in the template', () => {
        const tmpl = fs.readFileSync(
            path.resolve(__dirname, '../../assets/iframe.html.tmpl'),
            'utf8',
        );

        // Read the source of this file directly so we compare raw text,
        // not the Babel-transpiled output of Function.prototype.toString().
        const testSrc = fs.readFileSync(__filename, 'utf8');

        // Extract from template: function ends at the first `}` that sits at
        // the 6-space indentation level (the function's own closing brace).
        const tmplMatch = tmpl.match(/function errorToString\(error\) \{[\s\S]*?\n {6}\}/);

        // Extract from this file: function ends at the first `}` at column 0.
        const testMatch = testSrc.match(/^function errorToString\(error\) \{[\s\S]*?^\}/m);

        expect(tmplMatch).not.toBeNull();
        expect(testMatch).not.toBeNull();
        expect(normalize(tmplMatch[0])).toBe(normalize(testMatch[0]));
    });
});
