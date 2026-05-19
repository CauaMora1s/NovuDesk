import { init, register, getLocaleFromNavigator } from 'svelte-i18n';

register('pt', () => import('./pt.json'));
register('en', () => import('./en.json'));

export function setupI18n() {
	init({
		fallbackLocale: 'pt',
		initialLocale: getLocaleFromNavigator() ?? 'pt'
	});
}
