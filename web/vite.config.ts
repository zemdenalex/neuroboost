/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  // Windows test-runs only. The repo is commonly checked out on a Windows
  // junction (E: -> C:\E_Drive); without this, Vite resolves files to their
  // real path while Vitest's globs yield the junction path, so every test file
  // fails to load ("Does the file exist?").
  // Deliberately narrow on both axes: enabling it for *builds* breaks Rollup's
  // resolution of pnpm's symlinked node_modules, and Linux CI has no junction
  // to work around — so CI runs vanilla resolution.
  resolve:
    process.env.VITEST && process.platform === 'win32'
      ? { preserveSymlinks: true }
      : {},
  server: { port: 5173 },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
