import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // TODO: 本番環境では、vite.config.tsを変更する必要がある
  server: {
    host: '0.0.0.0',
    port: 3000,
  },
});
