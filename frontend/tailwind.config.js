import { heroui } from '@heroui/theme'
import colors from 'tailwindcss/colors.js'

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}', './node_modules/@heroui/theme/dist/**/*.{js,mjs}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Keep legacy utility names aligned with the HeroUI semantic palette.
        gray: colors.zinc,
        primary: {
          ...colors.emerald,
          DEFAULT: 'hsl(var(--heroui-primary) / <alpha-value>)',
          foreground: 'hsl(var(--heroui-primary-foreground) / <alpha-value>)'
        },
        accent: colors.zinc,
        dark: colors.zinc
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 4px 16px rgba(0, 0, 0, 0.05)',
        'glass-sm': '0 2px 8px rgba(0, 0, 0, 0.04)',
        glow: 'none',
        'glow-lg': 'none',
        card: '0 1px 2px rgba(0, 0, 0, 0.03)',
        'card-hover': '0 2px 6px rgba(0, 0, 0, 0.05)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(#18181b, #18181b)',
        'gradient-dark': 'linear-gradient(#18181b, #18181b)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient': 'none'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(20, 184, 166, 0.25)' },
          '100%': { boxShadow: '0 0 30px rgba(20, 184, 166, 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        xl: '8px', '2xl': '8px', '3xl': '8px', '4xl': '8px'
      },
      letterSpacing: { tighter: '0', tight: '0', normal: '0', wide: '0', wider: '0', widest: '0' }
    }
  },
  plugins: [heroui({
    defaultTheme: 'light',
    layout: {
      radius: { small: '6px', medium: '8px', large: '8px' },
      borderWidth: { small: '1px', medium: '1px', large: '2px' },
      boxShadow: { small: '0 1px 2px rgb(0 0 0 / 0.04)', medium: '0 4px 16px rgb(0 0 0 / 0.06)', large: '0 12px 40px rgb(0 0 0 / 0.12)' }
    },
    themes: {
      light: {
        colors: {
          background: '#fafafa', foreground: '#18181b',
          content1: '#ffffff', content2: '#f4f4f5', content3: '#e4e4e7', content4: '#d4d4d8',
          divider: '#e4e4e7', focus: '#059669',
          primary: { ...colors.emerald, DEFAULT: '#18181b', foreground: '#ffffff' },
          secondary: { DEFAULT: '#047857', foreground: '#ffffff' },
          success: { DEFAULT: '#047857', foreground: '#ffffff' },
          warning: { DEFAULT: '#b45309', foreground: '#ffffff' },
          danger: { DEFAULT: '#dc2626', foreground: '#ffffff' }
        }
      },
      dark: {
        colors: {
          background: '#09090b', foreground: '#f4f4f5',
          content1: '#18181b', content2: '#27272a', content3: '#3f3f46', content4: '#52525b',
          divider: '#3f3f46', focus: '#34d399',
          primary: { ...colors.emerald, DEFAULT: '#f4f4f5', foreground: '#18181b' },
          secondary: { DEFAULT: '#34d399', foreground: '#052e16' },
          success: { DEFAULT: '#34d399', foreground: '#052e16' },
          warning: { DEFAULT: '#fbbf24', foreground: '#18181b' },
          danger: { DEFAULT: '#f87171', foreground: '#18181b' }
        }
      }
    }
  })]
}
