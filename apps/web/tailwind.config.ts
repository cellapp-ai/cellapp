import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: 'var(--color-ink)',
        panel: 'var(--color-panel)',
        paper: 'var(--color-paper)',
        muted: 'var(--color-muted)',
        line: 'var(--color-line)',
        lime: 'var(--color-lime)',
        blue: 'var(--color-blue)',
      },
    },
  },
  plugins: [],
} satisfies Config;
