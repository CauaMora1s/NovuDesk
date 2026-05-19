import type { Config } from 'tailwindcss';
import daisyui from 'daisyui';

export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			fontFamily: {
				sans: ['Inter', 'system-ui', 'sans-serif']
			},
			fontSize: {
				xs: ['0.75rem', { lineHeight: '1rem' }],
				sm: ['0.875rem', { lineHeight: '1.25rem' }],
				base: ['1rem', { lineHeight: '1.5rem' }],
				lg: ['1.125rem', { lineHeight: '1.75rem' }],
				xl: ['1.25rem', { lineHeight: '1.75rem' }],
				'2xl': ['1.5rem', { lineHeight: '2rem' }]
			},
			spacing: {
				18: '4.5rem',
				22: '5.5rem'
			},
			borderRadius: {
				DEFAULT: '0.5rem'
			},
			boxShadow: {
				card: '0 1px 3px 0 rgb(0 0 0 / 0.04), 0 1px 2px -1px rgb(0 0 0 / 0.04)',
				'card-hover': '0 4px 6px -1px rgb(0 0 0 / 0.06), 0 2px 4px -2px rgb(0 0 0 / 0.06)'
			}
		}
	},
	plugins: [daisyui],
	daisyui: {
		themes: [
			{
				light: {
					primary: '#6366f1',
					'primary-content': '#ffffff',
					secondary: '#8b5cf6',
					accent: '#06b6d4',
					neutral: '#374151',
					'base-100': '#ffffff',
					'base-200': '#f9fafb',
					'base-300': '#f3f4f6',
					'base-content': '#111827',
					info: '#3b82f6',
					success: '#22c55e',
					warning: '#f59e0b',
					error: '#ef4444'
				}
			},
			{
				dark: {
					primary: '#818cf8',
					'primary-content': '#ffffff',
					secondary: '#a78bfa',
					accent: '#22d3ee',
					neutral: '#374151',
					'base-100': '#111827',
					'base-200': '#1f2937',
					'base-300': '#374151',
					'base-content': '#f9fafb',
					info: '#60a5fa',
					success: '#4ade80',
					warning: '#fbbf24',
					error: '#f87171'
				}
			}
		],
		darkTheme: 'dark',
		base: true,
		styled: true,
		utils: true,
		logs: false
	}
} satisfies Config;
