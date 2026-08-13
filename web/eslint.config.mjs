import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import globals from 'globals'

/**
 * ESLint for `web/`.
 *
 * Added 2026-08-13 because the audit found the ban on `any` written in three
 * separate rule files and enforced by nothing: there was no ESLint config, no
 * `lint` script, and no CI step. `pnpm typecheck` cannot stand in for this —
 * `any` is type-safe by definition, so tsc is silent on exactly the thing the
 * rule forbids.
 *
 * The rule set is deliberately narrow. A config that lights up hundreds of
 * stylistic findings on day one gets disabled on day two, and then the ban is
 * unenforced again with extra steps. What is here is what the project's own
 * written rules already require, plus the two correctness checks that catch
 * bugs tsc does not see.
 *
 * Kept as `.mjs` rather than `.js`: package.json declares no `"type"`, so a
 * bare `.js` config would be parsed as CommonJS and the imports above would
 * throw.
 */
export default tseslint.config(
  {
    // Build output, coverage and Playwright artifacts are not source.
    ignores: ['dist/**', 'coverage/**', 'e2e-results/**', 'playwright-report/**'],
  },

  js.configs.recommended,
  ...tseslint.configs.recommended,

  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      globals: { ...globals.browser, ...globals.es2022 },
    },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      // The project rule, verbatim: no `any` in TypeScript. Error, not warn —
      // a warning is a control that cannot fail.
      '@typescript-eslint/no-explicit-any': 'error',

      // Hook dependency mistakes are the frontend bug class tsc is blind to,
      // and this codebase has effects that write settings on a timer.
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',

      // Underscore-prefixed args stay legal: they are how an unused parameter
      // is deliberately marked, and this codebase uses that form.
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
      ],
    },
  },

  {
    // Node context, not browser: config files and the Playwright suite.
    files: ['*.config.{ts,mjs,js}', 'e2e/**/*.ts'],
    languageOptions: { globals: { ...globals.node } },
  },

  {
    // Playwright fixtures are not React. Their API names a parameter `use`,
    // which the hooks rule reads as React's `use` hook and reports as a
    // violation in every fixture; and `async ({}, testInfo) => …` is the
    // documented way to take no fixtures. Both rules are off here rather than
    // globally — a rule switched off where it does not belong stays sharp
    // where it does.
    files: ['e2e/**/*.ts'],
    rules: {
      'react-hooks/rules-of-hooks': 'off',
      'no-empty-pattern': 'off',
    },
  },
)
