<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth';
	import { ticketsApi, type Ticket, type TicketStatus } from '$lib/api/tickets';
	import { commentsApi, type TimelineItem } from '$lib/api/comments';
	import { categoriesApi, type Category } from '$lib/api/categories';
	import { membersApi, type Member } from '$lib/api/members';
	import { attachmentsApi, type Attachment } from '$lib/api/attachments';
	import { can } from '$lib/permissions';
	import { pollingInterval } from '$lib/stores/polling';
	import PollingControl from '$lib/components/PollingControl.svelte';
	import FileUpload from '$lib/components/FileUpload.svelte';

	let ticket: Ticket | null = null;
	let timeline: TimelineItem[] = [];
	let categories: Category[] = [];
	let orgMembers: Member[] = [];
	let attachments: Attachment[] = [];

	let loading = true;
	let error = '';
	let commentBody = '';
	let isInternal = false;
	let submittingComment = false;

	let fileUploadRef: FileUpload;

	$: ticketId = $page.params.id;
	$: currentUserId = $authStore.user?.id ?? '';
	$: isTeamMember = $authStore.teamIds.length > 0;
	$: isAdmin = $authStore.role === 'owner' || $authStore.role === 'admin';
	$: canManageTicket = isAdmin || isTeamMember;

	// Reload when ticket ID changes (navigation between tickets) — same as original
	$: if (ticketId) loadAll();

	async function loadAll() {
		loading = true;
		error = '';
		ticket = null;
		timeline = [];
		try {
			[ticket, timeline, attachments] = await Promise.all([
				ticketsApi.get(ticketId),
				commentsApi.list(ticketId),
				attachmentsApi.list(ticketId)
			]);
			if (canManageTicket) {
				[categories, orgMembers] = await Promise.all([
					categoriesApi.list(),
					membersApi.list()
				]);
			}
		} catch {
			error = $_('tickets.notFound');
		} finally {
			loading = false;
		}
	}

	let timer: ReturnType<typeof setInterval> | null = null;

	$: {
		if (timer) clearInterval(timer);
		timer = $pollingInterval > 0 ? setInterval(refreshData, $pollingInterval) : null;
	}

	onMount(() => () => { if (timer) clearInterval(timer); });


	async function changeStatus(status: TicketStatus) {
		if (!ticket) return;
		try {
			ticket = await ticketsApi.update(ticket.id, { status });
		} catch { /* handle silently */ }
	}

	async function changeCategory(categoryId: string) {
		if (!ticket) return;
		try {
			ticket = await ticketsApi.update(ticket.id, { category_id: categoryId || undefined });
		} catch { /* handle silently */ }
	}

	async function changeAssignee(assigneeId: string) {
		if (!ticket) return;
		try {
			ticket = await ticketsApi.update(ticket.id, { assignee_id: assigneeId || undefined });
		} catch { /* handle silently */ }
	}

	async function assignToMe() {
		if (!ticket || !currentUserId) return;
		try {
			ticket = await ticketsApi.update(ticket.id, { assignee_id: currentUserId });
		} catch { /* handle silently */ }
	}

	async function refreshData() {
		if (!ticketId || loading) return;
		try {
			const [newTicket, newTimeline, newAttachments] = await Promise.all([
				ticketsApi.get(ticketId),
				commentsApi.list(ticketId),
				attachmentsApi.list(ticketId)
			]);
			if (ticket && JSON.stringify(newTicket) !== JSON.stringify(ticket)) ticket = newTicket;
			const existingIds = new Set(timeline.map(i => i.id));
			const added = newTimeline.filter(i => !existingIds.has(i.id));
			if (added.length > 0) timeline = [...timeline, ...added];
			const existingAttIds = new Set(attachments.map(a => a.id));
			const addedAtts = newAttachments.filter(a => !existingAttIds.has(a.id));
			if (addedAtts.length > 0) attachments = [...attachments, ...addedAtts];
		} catch { /* silent */ }
	}

	async function sendComment() {
		if (!ticket || !commentBody.trim()) return;
		submittingComment = true;
		try {
			const item = await commentsApi.create(ticket.id, { body: commentBody.trim(), is_internal: isInternal });
			timeline = [...timeline, item];
			commentBody = '';

			if (fileUploadRef) {
				const uploaded = await fileUploadRef.uploadAll();
				attachments = [...attachments, ...uploaded];
			}

			setTimeout(() => {
				const el = document.getElementById('timeline-end');
				el?.scrollIntoView({ behavior: 'smooth' });
			}, 50);
		} catch { /* handle silently */ }
		submittingComment = false;
	}

	function formatDate(iso: string) {
		return new Intl.DateTimeFormat('pt-BR', {
			year: 'numeric', month: 'short', day: 'numeric',
			hour: '2-digit', minute: '2-digit'
		}).format(new Date(iso));
	}

	function timeAgo(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return $_('tickets.timeAgo.now');
		if (mins < 60) return $_('tickets.timeAgo.mins', { values: { count: mins } });
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return $_('tickets.timeAgo.hours', { values: { count: hrs } });
		const days = Math.floor(hrs / 24);
		return $_('tickets.timeAgo.days', { values: { count: days } });
	}

	function openDuration(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const hours = Math.floor(diff / 3600000);
		if (hours < 24) return $_('tickets.openDuration.hours', { values: { count: hours } });
		return $_('tickets.openDuration.days', { values: { count: Math.floor(hours / 24) } });
	}

	type ActivityDetail = {
		label: string;
		badge?: { text: string; css: string };
		value?: string;
	};

	function describeActivity(item: TimelineItem): ActivityDetail {
		const before = item.before as Record<string, unknown> | undefined;
		const after  = item.after  as Record<string, unknown> | undefined;

		if (item.action === 'ticket.created') {
			return { label: $_('tickets.activity.created') };
		}

		if (before && after) {
			if (before.status !== after.status) {
				const s = after.status as TicketStatus;
				return {
					label: $_('tickets.activity.statusChangedTo'),
					badge: { text: $_(  `tickets.status.${s}`), css: statusColors[s] ?? 'badge badge-sm' }
				};
			}
			if (before.priority !== after.priority) {
				const p = after.priority as string;
				return {
					label: $_('tickets.activity.priorityChangedTo'),
					badge: { text: $_(  `tickets.priority.${p}`), css: `status-badge priority-${p}` }
				};
			}
			if (before.assignee_id !== after.assignee_id) {
				if (after.assignee_id) {
					return {
						label: $_('tickets.activity.assignedTo'),
						value: lookupMemberName(after.assignee_id)
					};
				}
				return { label: $_('tickets.activity.unassignedTicket') };
			}
			if (before.category_id !== after.category_id) {
				if (after.category_id) {
					return {
						label: $_('tickets.activity.categoryChangedTo'),
						value: lookupCategoryName(after.category_id)
					};
				}
				return { label: $_('tickets.activity.categoryRemoved') };
			}
		}

		const fallbackKey = item.action ?? '';
		const i18nKeys: Record<string, string> = {
			'ticket.status_changed':   'tickets.activity.statusChanged',
			'ticket.assigned':         'tickets.activity.assigned',
			'ticket.category_changed': 'tickets.activity.categoryChanged',
			'ticket.priority_changed': 'tickets.activity.priorityChanged',
			'ticket.updated':          'tickets.activity.updated'
		};
		return { label: i18nKeys[fallbackKey] ? $_(i18nKeys[fallbackKey]) : fallbackKey };
	}

	function lookupMemberName(id: unknown): string {
		if (!id) return '';
		const sid = String(id);
		const name = orgMembers.find(m => m.id === sid)?.full_name
			?? (ticket?.assignee_id === sid ? (ticket?.assignee_name ?? '') : '');
		return name || sid;
	}

	function lookupCategoryName(id: unknown): string {
		if (!id) return '';
		const sid = String(id);
		const name = categories.find(c => c.id === sid)?.name
			?? (ticket?.category_id === sid ? (ticket?.category_name ?? '') : '');
		return name || sid;
	}

	function initials(name: string | undefined | null): string {
		if (!name) return '?';
		return name.split(' ').slice(0, 2).map((n) => n[0]).join('').toUpperCase();
	}

	const statusColors: Record<TicketStatus, string> = {
		open:     'status-badge status-open',
		pending:  'status-badge status-pending',
		on_hold:  'status-badge status-on-hold',
		resolved: 'status-badge status-resolved',
		closed:   'status-badge status-closed'
	};

	const priorityColors = {
		low:    'priority-low',
		normal: 'priority-normal',
		high:   'priority-high',
		urgent: 'priority-urgent'
	};

	function formatBytes(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

<svelte:head>
	<title>{ticket ? `#${ticket.number} ${ticket.title}` : $_('tickets.title')} — NovuDesk</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	<!-- Back + polling -->
	<div class="flex items-center justify-between mb-5">
		<a href="/tickets" class="inline-flex items-center gap-1.5 text-sm text-base-content/50 hover:text-base-content transition-colors">
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
			</svg>
			{$_('tickets.backToAll')}
		</a>
		<PollingControl />
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<span class="loading loading-spinner loading-lg text-primary"></span>
		</div>
	{:else if error || !ticket}
		<div class="alert alert-error">{error || $_('tickets.notFound')}</div>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-5">

			<!-- Main column -->
			<div class="lg:col-span-2 space-y-4">

				<!-- Title card -->
				<div class="card bg-base-100 shadow-card">
					<div class="card-body p-6">
						<div class="flex items-start justify-between gap-4">
							<div class="flex-1 min-w-0">
								<div class="flex items-center flex-wrap gap-2 mb-2">
									<span class="text-xs text-base-content/40 font-mono">#{ticket.number}</span>
									<span class="{statusColors[ticket.status]}">
										{$_(`tickets.status.${ticket.status}`)}
									</span>
									{#if ticket.sla_breached}
										<span class="badge badge-sm badge-error">{$_('tickets.slaBreached')}</span>
									{/if}
									{#if ticket.category_name}
										<span class="badge badge-sm badge-outline">{ticket.category_name}</span>
									{/if}
								</div>
								<h1 class="text-xl font-bold">{ticket.title}</h1>
							</div>

							{#if canManageTicket && can('tickets:change_status')}
								<div class="dropdown dropdown-end shrink-0">
									<button class="btn btn-sm btn-outline gap-1">
										{$_('tickets.changeStatus')}
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
										</svg>
									</button>
									<ul class="dropdown-content menu menu-sm bg-base-100 rounded-box shadow-lg border border-base-200 z-10 w-44 mt-1">
										{#each ['open','pending','on_hold','resolved','closed'] as s}
											<li>
												<button
													on:click={() => changeStatus(s as TicketStatus)}
													class:font-medium={ticket.status === s}
												>
													{$_(`tickets.status.${s}`)}
												</button>
											</li>
										{/each}
									</ul>
								</div>
							{/if}
						</div>

						{#if ticket.description}
							<div class="prose prose-sm max-w-none mt-4 text-base-content/80">
								<p>{ticket.description}</p>
							</div>
						{/if}

						<div class="flex items-center gap-3 text-xs text-base-content/40 mt-4">
							<span>{$_('tickets.createdMsg', { values: { time: timeAgo(ticket.created_at) } })}</span>
							<span>·</span>
							<span>{$_('tickets.updatedMsg', { values: { time: timeAgo(ticket.updated_at) } })}</span>
						</div>
					</div>
				</div>

				<!-- Timeline -->
				{#if timeline.length > 0}
					<div class="space-y-3">
						{#each timeline as item}
							{#if item.type === 'comment'}
								<!-- Comment bubble -->
								<div class="card bg-base-100 shadow-card"
									class:border-l-4={item.is_internal}
									class:border-warning={item.is_internal}
								>
									<div class="card-body p-4">
										<div class="flex items-start gap-3">
											<div class="avatar placeholder shrink-0">
												<div class="w-8 h-8 rounded-full bg-primary/10 text-primary text-xs font-semibold flex items-center justify-center">
													{initials(item.author_name)}
												</div>
											</div>
											<div class="flex-1 min-w-0">
												<div class="flex items-center gap-2 mb-1">
													<span class="text-sm font-medium">{item.author_name ?? $_('tickets.systemActor')}</span>
													{#if item.is_internal}
														<span class="badge badge-warning badge-xs gap-1">
															<svg class="w-2.5 h-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
																	d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
															</svg>
															{$_('tickets.internalNote')}
														</span>
													{/if}
													<span class="text-xs text-base-content/40">{timeAgo(item.created_at)}</span>
												</div>
												<p class="text-sm text-base-content/80 whitespace-pre-wrap">{item.body}</p>
											</div>
										</div>
									</div>
								</div>
							{:else}
								<!-- Activity event -->
								{@const detail = describeActivity(item)}
								<div class="flex items-center gap-3 px-2 py-1">
									<div class="w-px h-4 bg-base-300 ml-4"></div>
									<div class="flex-1 flex items-center flex-wrap gap-1.5 text-xs text-base-content/50">
										<span class="font-medium text-base-content/70">
											{item.actor_type === 'system'
												? $_('tickets.systemActor')
												: (item.actor_name ?? $_('tickets.systemActor'))}
										</span>
										<span>{detail.label}</span>
										{#if detail.badge}
											<span class="{detail.badge.css} !text-xs">{detail.badge.text}</span>
										{:else if detail.value}
											<span class="font-medium text-base-content/70">{detail.value}</span>
										{/if}
										<span>·</span>
										<span>{timeAgo(item.created_at)}</span>
									</div>
								</div>
							{/if}
						{/each}
					</div>
				{/if}
				<div id="timeline-end"></div>

				<!-- Reply box -->
				{#if can('comments:create_public') || can('comments:create_internal')}
					<div class="card bg-base-100 shadow-card">
						<div class="card-body p-5">
							<div class="tabs tabs-bordered mb-3">
								{#if can('comments:create_public')}
									<button
										class="tab tab-sm"
										class:tab-active={!isInternal}
										on:click={() => (isInternal = false)}
									>
										{$_('tickets.comments.public')}
									</button>
								{/if}
								{#if can('comments:create_internal')}
									<button
										class="tab tab-sm"
										class:tab-active={isInternal}
										on:click={() => (isInternal = true)}
									>
										{$_('tickets.comments.internal')}
									</button>
								{/if}
							</div>

							{#if isInternal}
								<div class="alert alert-warning py-2 text-xs mb-2">
									{$_('tickets.internalNoteWarning')}
								</div>
							{/if}

							<textarea
								bind:value={commentBody}
								placeholder={isInternal
									? $_('tickets.comments.internalPlaceholder')
									: $_('tickets.comments.placeholder')}
								class="textarea textarea-bordered w-full min-h-24 resize-none text-sm"
								class:border-warning={isInternal}
							></textarea>

							<div class="mt-3">
								<FileUpload bind:this={fileUploadRef} {ticketId} />
							</div>

							<div class="flex justify-end mt-3">
								<button
									class="btn btn-primary btn-sm"
									disabled={submittingComment || !commentBody.trim()}
									on:click={sendComment}
								>
									{#if submittingComment}
										<span class="loading loading-spinner loading-xs"></span>
									{/if}
									{$_('tickets.comments.send')}
								</button>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Sidebar column -->
			<div class="space-y-4">

				<!-- Details card -->
				<div class="card bg-base-100 shadow-card">
					<div class="card-body p-5">
						<h3 class="text-xs font-semibold text-base-content/50 uppercase tracking-wide mb-4">{$_('tickets.detailsSection')}</h3>

						<dl class="space-y-4 text-sm">

							<!-- Priority -->
							<div>
								<dt class="text-xs text-base-content/40 mb-1">{$_('tickets.priorityLabel')}</dt>
								<dd>
									<span class="status-badge {priorityColors[ticket.priority]}">
										{$_(`tickets.priority.${ticket.priority}`)}
									</span>
								</dd>
							</div>

							<!-- Category -->
							<div>
								<dt class="text-xs text-base-content/40 mb-1">{$_('tickets.categoryLabel')}</dt>
								<dd>
									{#if canManageTicket && categories.length > 0}
										<select
											class="select select-bordered select-sm w-full text-sm"
											value={ticket.category_id ?? ''}
											on:change={(e) => changeCategory((e.target as HTMLSelectElement).value)}
										>
											<option value="">{$_('tickets.noCategory')}</option>
											{#each categories as cat}
												<option value={cat.id}>{cat.name}</option>
											{/each}
										</select>
									{:else if ticket.category_name}
										<span class="badge badge-sm badge-outline">{ticket.category_name}</span>
									{:else}
										<span class="text-base-content/40 text-xs">{$_('tickets.categoryUndefined')}</span>
									{/if}
								</dd>
							</div>

							<!-- Assignee -->
							<div>
								<dt class="text-xs text-base-content/40 mb-1">{$_('tickets.assigneeLabel')}</dt>
								<dd>
									{#if canManageTicket && can('tickets:assign')}
										<div class="space-y-1.5">
											<select
												class="select select-bordered select-sm w-full text-sm"
												value={ticket.assignee_id ?? ''}
												on:change={(e) => changeAssignee((e.target as HTMLSelectElement).value)}
											>
												<option value="">{$_('tickets.unassigned')}</option>
												{#each orgMembers as m}
													<option value={m.id}>{m.full_name}</option>
												{/each}
											</select>
											{#if !ticket.assignee_id}
												<button class="btn btn-outline btn-xs w-full" on:click={assignToMe}>
													{$_('tickets.assignToMe')}
												</button>
											{/if}
										</div>
									{:else if ticket.assignee_name}
										<div class="flex items-center gap-2">
											<div class="avatar placeholder">
												<div class="w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-semibold flex items-center justify-center">
													{initials(ticket.assignee_name)}
												</div>
											</div>
											<span class="text-sm">{ticket.assignee_name}</span>
										</div>
									{:else}
										<span class="text-base-content/40 text-xs">{$_('tickets.unassigned')}</span>
									{/if}
								</dd>
							</div>

							<!-- Team -->
							{#if ticket.team_name}
								<div>
									<dt class="text-xs text-base-content/40 mb-1">{$_('tickets.teamLabel')}</dt>
									<dd class="text-sm">{ticket.team_name}</dd>
								</div>
							{/if}

							<!-- Tags -->
							{#if ticket.tags && ticket.tags.length > 0}
								<div>
									<dt class="text-xs text-base-content/40 mb-1">{$_('tickets.tagsLabel')}</dt>
									<dd class="flex flex-wrap gap-1">
										{#each ticket.tags as tag}
											<span class="badge badge-ghost badge-sm">{tag}</span>
										{/each}
									</dd>
								</div>
							{/if}

							<!-- SLA -->
							{#if ticket.sla_resolution_due_at}
								<div>
									<dt class="text-xs text-base-content/40 mb-1">{$_('tickets.slaDueLabel')}</dt>
									<dd class="text-sm" class:text-error={ticket.sla_breached}>
										{formatDate(ticket.sla_resolution_due_at)}
									</dd>
								</div>
							{/if}
						</dl>
					</div>
				</div>

				<!-- Attachments card -->
				<div class="card bg-base-100 shadow-card">
					<div class="card-body p-5">
						<h3 class="text-xs font-semibold text-base-content/50 uppercase tracking-wide mb-3">{$_('tickets.attachments.title')}</h3>

						{#if attachments.length > 0}
							<ul class="space-y-2 mb-3">
								{#each attachments as att}
									<li class="flex items-center gap-2 text-xs">
										<svg class="w-3.5 h-3.5 shrink-0 text-base-content/40" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
												d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
										</svg>
										<a
											href={att.url}
											target="_blank"
											rel="noopener noreferrer"
											class="flex-1 truncate hover:text-primary transition-colors"
											title={att.filename}
										>
											{att.filename}
										</a>
										<span class="text-base-content/40 shrink-0">{formatBytes(att.size_bytes)}</span>
									</li>
								{/each}
							</ul>
						{/if}

						{#if can('tickets:create') || can('comments:create_public')}
							<FileUpload
								{ticketId}
								onUploaded={(a) => { attachments = [...attachments, a]; }}
							/>
						{/if}
					</div>
				</div>

				<!-- Info card -->
				<div class="card bg-base-100 shadow-card">
					<div class="card-body p-5">
						<h3 class="text-xs font-semibold text-base-content/50 uppercase tracking-wide mb-4">{$_('tickets.infoSection')}</h3>
						<dl class="space-y-3 text-sm">
							{#if ticket.requester_name}
								<div>
									<dt class="text-xs text-base-content/40 mb-0.5">{$_('tickets.createdBy')}</dt>
									<dd>{ticket.requester_name}</dd>
								</div>
							{/if}
							<div>
								<dt class="text-xs text-base-content/40 mb-0.5">{$_('tickets.openFor')}</dt>
								<dd>{openDuration(ticket.created_at)}</dd>
							</div>
							<div>
								<dt class="text-xs text-base-content/40 mb-0.5">{$_('tickets.lastUpdated')}</dt>
								<dd>{timeAgo(ticket.updated_at)}</dd>
							</div>
							<div>
								<dt class="text-xs text-base-content/40 mb-0.5">{$_('tickets.createdAt')}</dt>
								<dd class="text-xs text-base-content/60">{formatDate(ticket.created_at)}</dd>
							</div>
						</dl>
					</div>
				</div>

			</div>
		</div>
	{/if}
</div>
