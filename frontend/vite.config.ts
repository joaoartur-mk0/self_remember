import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // Espelha o proxy /api/ do nginx do container web (docker-compose.local.yml)
    // para que `npm run dev` local também converse com a API em mesma origem —
    // AuthMiddleware só aceita o cookie access_token, nunca Authorization.
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
