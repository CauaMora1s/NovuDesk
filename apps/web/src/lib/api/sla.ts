import { api } from './client';

export type SLAUnit = 'hours' | 'days' | 'weeks';

export interface CategorySLAStat {
	category_id: string;
	category_name: string;
	sla_id: string | null;
	resolution_value: number | null;
	resolution_unit: SLAUnit | null;
	resolution_hours: number | null;
	avg_resolution_hours: number | null;
}

export interface UpsertSLAInput {
	resolution_value: number;
	resolution_unit: SLAUnit;
	response_hours?: number;
}

export const slaApi = {
	listWithStats: () => api.get<CategorySLAStat[]>('/api/v1/sla-policies'),

	upsert: (categoryId: string, input: UpsertSLAInput) =>
		api.put<unknown>(`/api/v1/sla-policies/category/${categoryId}`, input),

	delete: (id: string) => api.delete<void>(`/api/v1/sla-policies/${id}`)
};

export function formatAvgDuration(hours: number | null): string {
	if (hours === null || hours === undefined) return '—';
	if (hours < 1) return `${Math.round(hours * 60)} min`;
	if (hours < 24) return `${hours.toFixed(1)}h`;
	const days = Math.floor(hours / 24);
	const rem = Math.round(hours % 24);
	return rem > 0 ? `${days}d ${rem}h` : `${days}d`;
}

export function formatSLAValue(value: number | null, unit: SLAUnit | null): string {
	if (!value || !unit) return '—';
	const labels: Record<SLAUnit, string> = { hours: 'h', days: 'd', weeks: 'w' };
	return `${value} ${labels[unit]}`;
}
