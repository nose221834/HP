import js from '@eslint/js';
import eslintConfigPrettier from 'eslint-config-prettier';
import prettier from 'eslint-plugin-prettier';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import tseslint from 'typescript-eslint';

const prettierConfig = {
  singleQuote: true,
  semi: true,
  tabWidth: 2,
  trailingComma: 'es5',
  bracketSpacing: true,
  // 注意: importOrderを使用するには@ianvs/prettier-plugin-sort-importsが必要
  importOrder: [
    '^(react/(.*)$)|^(react$)',
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

export default tseslint.config(
  // 無視するディレクトリ
  { ignores: ['node_modules', '.vite', '.pnpm-store'] },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
      // 型チェック用の設定
      ...tseslint.configs.recommendedTypeChecked,
      // Prettierと競合するルールを無効化（必ず最後に配置）
      eslintConfigPrettier,
    ],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        // ソースファイルを直接含むtsconfig.app.jsonを指定
        project: './tsconfig.app.json',
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      prettier: prettier,
    },
    rules: {
      // React Hooksのルール
      ...reactHooks.configs.recommended.rules,

      // React Refreshのルール
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],

      // Prettierのルール - コードスタイルの自動整形
      'prettier/prettier': ['error', prettierConfig],
    },
  }
);
