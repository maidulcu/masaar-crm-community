import type { Config } from 'tailwindcss'

/**
 * Masaar CRM — Modern Minimalist Design System
 *
 * Palette philosophy: warm neutral surfaces, restrained indigo primary,
 * UAE-gold accent reserved for premium signals. Generous radii, soft
 * shadows, smooth easing. All scales are full 50→900 so utilities like
 * `bg-primary-50`, `text-gold-700`, `ring-primary-400` all resolve.
 */
const config: Config = {
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['var(--font-inter)', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        arabic: ['var(--font-cairo)', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      },
      colors: {
        // Primary — refined indigo/blue. Modern, calm, professional.
        primary: {
          50:  '#eef2ff',
          100: '#e0e7ff',
          200: '#c7d2fe',
          300: '#a5b4fc',
          400: '#818cf8',
          500: '#6366f1',
          600: '#4f46e5',
          700: '#4338ca',
          800: '#3730a3',
          900: '#312e81',
        },
        // Brand — alias of primary so legacy `brand-*` usages keep working.
        brand: {
          50:  '#eef2ff',
          100: '#e0e7ff',
          200: '#c7d2fe',
          300: '#a5b4fc',
          400: '#818cf8',
          500: '#6366f1',
          600: '#4f46e5',
          700: '#4338ca',
          800: '#3730a3',
          900: '#312e81',
        },
        // Gold — UAE accent, premium signals only (won deals, AED values).
        gold: {
          50:  '#fffbeb',
          100: '#fef3c7',
          200: '#fde68a',
          300: '#fcd34d',
          400: '#fbbf24',
          500: '#f59e0b',
          600: '#d97706',
          700: '#b45309',
          800: '#92400e',
          900: '#78350f',
        },
        // Surface — warm-neutral page/card backgrounds. Replaces the
        // slightly cool gray-50 default for a softer feel.
        surface: {
          50:  '#fafafa',
          100: '#f5f5f5',
          200: '#eaeaea',
          300: '#d4d4d4',
          400: '#a3a3a3',
          500: '#737373',
          600: '#525252',
          700: '#404040',
          800: '#262626',
          900: '#171717',
        },
      },
      borderRadius: {
        '4xl': '2rem',
      },
      boxShadow: {
        // Soft layered shadow for cards on light backgrounds.
        card:        '0 1px 2px 0 rgb(15 23 42 / 0.04), 0 1px 3px 0 rgb(15 23 42 / 0.06)',
        'card-hover':'0 4px 12px -2px rgb(15 23 42 / 0.08), 0 2px 4px -1px rgb(15 23 42 / 0.04)',
        pop:         '0 12px 32px -8px rgb(15 23 42 / 0.12), 0 4px 8px -2px rgb(15 23 42 / 0.06)',
        // Subtle ring used for active sidebar items, focus halos.
        ring:        '0 0 0 1px rgb(99 102 241 / 0.18)',
      },
      transitionTimingFunction: {
        soft: 'cubic-bezier(0.22, 1, 0.36, 1)',
      },
      keyframes: {
        'slide-up': {
          '0%':   { transform: 'translateY(8px)', opacity: '0' },
          '100%': { transform: 'translateY(0)',   opacity: '1' },
        },
        'fade-in': {
          '0%':   { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'scale-in': {
          '0%':   { transform: 'scale(0.96)', opacity: '0' },
          '100%': { transform: 'scale(1)',    opacity: '1' },
        },
      },
      animation: {
        'slide-up': 'slide-up 0.25s cubic-bezier(0.22, 1, 0.36, 1) forwards',
        'fade-in':  'fade-in 0.2s ease-out forwards',
        'scale-in': 'scale-in 0.18s cubic-bezier(0.22, 1, 0.36, 1) forwards',
      },
    },
  },
  plugins: [],
}

export default config
