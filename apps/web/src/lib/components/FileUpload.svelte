<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { attachmentsApi, type Attachment } from '$lib/api/attachments';

	export let ticketId: string;
	export let commentId: string | undefined = undefined;
	export let onUploaded: (a: Attachment) => void = () => {};

	let files: File[] = [];
	let uploading = false;
	let error = '';

	const ALLOWED_TYPES = [
		'image/jpeg', 'image/png', 'image/gif', 'image/webp',
		'application/pdf', 'text/plain', 'text/csv',
		'application/zip',
		'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
		'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
	];

	const MAX_BYTES = 25 * 1024 * 1024;

	function onFileChange(e: Event) {
		error = '';
		const input = e.target as HTMLInputElement;
		const selected = Array.from(input.files ?? []);
		const invalid = selected.find((f) => !ALLOWED_TYPES.includes(f.type));
		if (invalid) {
			error = $_('tickets.attachments.error');
			input.value = '';
			return;
		}
		const oversized = selected.find((f) => f.size > MAX_BYTES);
		if (oversized) {
			error = $_('tickets.attachments.maxSize');
			input.value = '';
			return;
		}
		files = [...files, ...selected];
		input.value = '';
	}

	function removeFile(index: number) {
		files = files.filter((_, i) => i !== index);
	}

	export async function uploadAll(overrideTicketId?: string): Promise<Attachment[]> {
		const targetId = overrideTicketId ?? ticketId;
		if (files.length === 0 || !targetId) return [];
		uploading = true;
		error = '';
		const uploaded: Attachment[] = [];
		try {
			for (const file of files) {
				const a = await attachmentsApi.upload(targetId, file, commentId);
				uploaded.push(a);
				onUploaded(a);
			}
			files = [];
		} catch {
			error = $_('tickets.attachments.error');
		} finally {
			uploading = false;
		}
		return uploaded;
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

<div class="space-y-2">
	<div class="flex items-center gap-2">
		<label class="btn btn-ghost btn-xs gap-1 cursor-pointer">
			<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
					d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
			</svg>
			{$_('tickets.attachments.add')}
			<input
				type="file"
				multiple
				accept={ALLOWED_TYPES.join(',')}
				class="hidden"
				on:change={onFileChange}
			/>
		</label>
		<span class="text-xs text-base-content/40">{$_('tickets.attachments.maxSize')}</span>
	</div>

	{#if error}
		<p class="text-xs text-error">{error}</p>
	{/if}

	{#if files.length > 0}
		<ul class="space-y-1">
			{#each files as file, i}
				<li class="flex items-center gap-2 text-xs bg-base-200 rounded px-2 py-1">
					<svg class="w-3.5 h-3.5 shrink-0 text-base-content/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
							d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
					<span class="flex-1 truncate">{file.name}</span>
					<span class="text-base-content/40 shrink-0">{formatSize(file.size)}</span>
					{#if !uploading}
						<button
							type="button"
							class="text-base-content/40 hover:text-error transition-colors"
							on:click={() => removeFile(i)}
							aria-label={$_('tickets.attachments.remove')}
						>
							<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if uploading}
		<p class="text-xs text-base-content/50 flex items-center gap-1">
			<span class="loading loading-spinner loading-xs"></span>
			{$_('tickets.attachments.uploading')}
		</p>
	{/if}
</div>
