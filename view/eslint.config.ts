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

// ESLint 9.x フラット設定形式
export default [
  // グローバル設定（無視するファイル）
  {
    ignores: ['node_modules', '.vite', '.pnpm-store'],
  },

  // メインの設定
  {
    // 言語オプション（パーサーとパーサーオプションを含む）
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        ecmaFeatures: { jsx: true },
        project: './tsconfig.app.json',
      },
    },

    // プラグインの定義
    plugins: {
      solid: solidPlugin,
      '@typescript-eslint': tsPlugin,
      import: importPlugin,
      prettier: prettierPlugin,
    },

    // ルールセットを継承（フラット設定では扱いが異なる）
    rules: {
      // eslint:recommended のルール
      'no-console': 'warn',
      'no-unused-vars': 'off', // @typescript-eslint で代替
      'no-undef': 'error',

      // @typescript-eslint のルール
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_' },
      ],
      '@typescript-eslint/explicit-function-return-type': [
        'error',
        { allowExpressions: true },
      ],

      // solid のルール
      'solid/jsx-no-construct': 'error',
      'solid/jsx-no-script-url': 'error',
      'solid/jsx-no-undef': 'error',
      'solid/jsx-uses-solid': 'error',

      // import のルール
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

      // prettier のルール
      'prettier/prettier': ['error', prettierConfig],
    },

    // 設定（フラット設定形式では settings が外側に)
    settings: {
      'import/resolver': {
        typescript: {
          project: './tsconfig.app.json',
        },
      },
    },
  },

  // TSX ファイル向けの特殊ルール（overrides の代わりに別オブジェクトで指定）
  {
    files: ['**/*.tsx'],
    rules: {
      'solid/jsx-quotes': ['error', 'prefer-double'],
    },
  },
];
