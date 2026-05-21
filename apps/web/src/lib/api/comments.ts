import { api } from './client';

export type TimelineItemType = 'comment' | 'activity';

export interface TimelineItem {
	type: TimelineItemType;
	id: string;
	created_at: string;

	// comment fields
	body?: string;
	is_internal?: boolean;
	author_id?: string;
	author_name?: string;
	author_avatar?: string;

	// activity fields
	action?: string;
	actor_id?: string;
	actor_type?: string;
	actor_name?: string;
	before?: Record<string, unknown>;
	after?: Record<string, unknown>;
}

export interface CreateCommentInput {
	body: string;
	is_internal?: boolean;
}

export const commentsApi = {
	list: (ticketId: string) =>
		api.get<TimelineItem[]>(`/api/v1/tickets/${ticketId}/comments`),

	create: (ticketId: string, input: CreateCommentInput) =>
		api.post<TimelineItem>(`/api/v1/tickets/${ticketId}/comments`, input)
};
