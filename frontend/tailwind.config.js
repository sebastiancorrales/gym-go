/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,jsx}",
  ],
  theme: {
    extend: {
      colors: {
        navy: {
          DEFAULT: '#0F1C35',
          mid:     '#152744',
          hover:   '#1C2F4F',
        },
        primary: {
          DEFAULT: '#1272D6',
          hover:   '#0D5BAD',
          bg:      '#EBF3FF',
          muted:   'rgba(18,114,214,0.14)',
        },
        surface: '#FFFFFF',
        'app-bg': '#F4F7FC',
        'app-border': '#E2E8EF',
        'border-light': '#F0F4F9',
        'text-main': '#0F1C35',
        'text-sec': '#4B5778',
        'text-muted': '#94A3B8',
        'status-success': '#10B981',
        'status-success-bg': '#DCFCE7',
        'status-warn': '#F59E0B',
        'status-warn-bg': '#FEF3C7',
        'status-danger': '#EF4444',
        'status-danger-bg': '#FEE2E2',
        'accent-purple': '#6D28D9',
        'accent-purple-bg': '#EDE9FE',
        'accent-orange': '#EA580C',
        'accent-orange-bg': '#FFF7ED',
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        'card': '12px',
        'modal': '16px',
      },
      boxShadow: {
        'card': '0 1px 3px rgba(0,0,0,0.05)',
        'card-hover': '0 8px 24px rgba(0,0,0,0.10)',
        'primary': '0 4px 14px rgba(18,114,214,0.4)',
        'modal': '0 24px 60px rgba(0,0,0,0.22)',
      },
    },
  },
  plugins: [],
}
