import { api } from './client';

export type PlanTier = 'free' | 'pro' | 'business' | 'enterprise';

export interface PlanLimits {
	seats: number;
	tickets_per_month: number;
	storage_bytes: number;
	teams: number;
	categories: number;
	api_keys: number;
}

export interface Plan {
	tier: PlanTier;
	name: string;
	price_cents: number;
	currency: string;
	limits: PlanLimits;
}

export interface Usage {
	members: number;
	tickets_this_month: number;
	storage_bytes: number;
	teams: number;
	categories: number;
	api_keys: number;
}

export interface Organization {
	id: string;
	name: string;
	slug: string;
	logo_url: string | null;
	plan_tier: PlanTier;
	created_at: string;
	updated_at: string;
	plan_renews_at: string | null;
	billing_status: string;
	billing_provider: string;
	billing_customer_ref: string | null;
	payment_method_brand: string | null;
	payment_method_last4: string | null;
}

export type SessionStatus = 'pending' | 'completed' | 'cancelled' | 'failed' | 'expired';

export interface PaymentSession {
	id: string;
	org_id: string;
	from_tier: string | null;
	to_tier: PlanTier;
	status: SessionStatus;
	amount_cents: number;
	proration_cents: number;
	currency: string;
	provider: string;
	provider_ref: string | null;
	created_by: string | null;
	created_at: string;
	completed_at: string | null;
	expires_at: string | null;
}

export interface OrgOverview {
	organization: Organization;
	plan: Plan;
	usage: Usage;
	pending_session: PaymentSession | null;
}

export const organizationApi = {
	getOverview: () => api.get<OrgOverview>('/api/v1/organization'),

	listPlans: () => api.get<Plan[]>('/api/v1/organization/plans'),

	update: (input: { name: string }) => api.patch<Organization>('/api/v1/organization', input),

	initiatePlanChange: (toTier: PlanTier) =>
		api.post<PaymentSession>('/api/v1/organization/plan/sessions', { to_tier: toTier }),

	confirmSession: (id: string) =>
		api.post<PaymentSession>(`/api/v1/organization/plan/sessions/${id}/confirm`, {}),

	cancelSession: (id: string) =>
		api.post<void>(`/api/v1/organization/plan/sessions/${id}/cancel`, {}),

	sessionHistory: () => api.get<PaymentSession[]>('/api/v1/organization/plan/sessions')
};

const UNLIMITED = -1;

export function isUnlimited(value: number): boolean {
	return value < 0;
}

export function formatBytes(bytes: number): string {
	if (bytes < 0) return '∞';
	if (bytes === 0) return '0 B';
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(1024));
	const value = bytes / Math.pow(1024, i);
	return `${value.toFixed(value >= 10 || i === 0 ? 0 : 1)} ${units[i]}`;
}

export function formatLimit(value: number, isBytes = false): string {
	if (isUnlimited(value)) return '∞';
	return isBytes ? formatBytes(value) : value.toLocaleString();
}

export function formatPrice(cents: number, currency = 'BRL'): string {
	return new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(cents / 100);
}

export function usagePercent(current: number, limit: number): number {
	if (isUnlimited(limit) || limit === 0) return 0;
	return Math.min(100, Math.round((current / limit) * 100));
}

export { UNLIMITED };
