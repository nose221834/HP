import js from '@eslint/js';
import eslintConfigPrettier from 'eslint-config-prettier';
import prettier from 'eslint-plugin-prettier';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['dist'] },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
      eslintConfigPrettier,
    ],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      prettier: prettier,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      'prettier/prettier': [
        'error',
        {
          // Prettierの設定をここに追加
          singleQuote: true,
          semi: true,
          tabWidth: 2,
          trailingComma: 'es5',
          bracketSpacing: true,
          importOrder: [
            '^(react/(.*)$)|^(react$)',
            '^(next/(.*)$)|^(next$)',
            '<THIRD_PARTY_MODULES>',
            '^types$',
            '^@/config/(.*)$',
            '^@/lib/(.*)$',
            '^@/components/(.*)$',
            '^@/hooks/(.*)$',
            '^@/styles/(.*)$',
            '^[./]',
          ],
          importOrderSeparation: false,
          importOrderSortSpecifiers: true,
        },
      ],
    },
  }
);
