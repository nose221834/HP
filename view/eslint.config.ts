import tsPlugin from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import importPlugin from 'eslint-plugin-import';
import prettierPlugin from 'eslint-plugin-prettier';
import solidPlugin from 'eslint-plugin-solid';

const prettierConfig = {
  singleQuote: true,
  semi: true,
  tabWidth: 2,
  trailingComma: 'es5',
  bracketSpacing: true,
  importOrder: [
    '^solid-js/(.*)$',
    '<THIRD_PARTY_MODULES>',
    '',
    '^types$',
    '^@local/(.*)$',
    '^@/config/(.*)$',
    '^@/lib/(.*)$',
    '^@/components/(.*)$',
    '^@/styles/(.*)$',
    '^[./]',
  ],
  importOrderSeparation: false,
  importOrderSortSpecifiers: true,
  importOrderBuiltinModulesToTop: true,
  importOrderParserPlugins: ['typescript', 'jsx', 'decorators-legacy'],
  importOrderMergeDuplicateImports: true,
  importOrderCombineTypeAndValueImports: true,
  plugins: ['@trivago/prettier-plugin-sort-imports'],
};

export default {
  ignores: ['node_modules', '.vite', '.pnpm-store'],
  parser: tsParser,
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module',
    ecmaFeatures: { jsx: true },
    project: './tsconfig.app.json',
  },
  plugins: {
    solid: solidPlugin,
    '@typescript-eslint': tsPlugin,
    import: importPlugin,
    prettier: prettierPlugin,
  },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
    'plugin:solid/typescript',
    'plugin:solid/jsx-a11y',
    'plugin:import/errors',
    'plugin:import/warnings',
    'plugin:import/typescript',
    'plugin:prettier/recommended',
    'prettier',
  ],
  settings: {
    'import/resolver': {
      typescript: {
        project: './tsconfig.app.json',
      },
    },
  },
  rules: {
    'solid/jsx-no-construct': 'error',
    'solid/jsx-no-script-url': 'error',
    'solid/jsx-no-undef': 'error',
    'solid/jsx-uses-solid': 'error',
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    '@typescript-eslint/explicit-function-return-type': [
      'error',
      { allowExpressions: true },
    ],
    'import/order': [
      'error',
      {
        groups: [
          'builtin',
          'external',
          'internal',
          ['parent', 'sibling', 'index'],
        ],
        pathGroups: [
          { pattern: 'solid-js/**', group: 'external', position: 'before' },
        ],
        'newlines-between': 'always',
        alphabetize: { order: 'asc', caseInsensitive: true },
      },
    ],
    'prettier/prettier': ['error', prettierConfig],
  },
  overrides: [
    {
      files: ['*.tsx'],
      rules: {
        'solid/jsx-quotes': ['error', 'prefer-double'],
      },
    },
  ],
};
