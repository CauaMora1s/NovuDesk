<script lang="ts">
</script>

<div class="min-h-screen bg-base-200 flex items-center justify-center p-4">
	<div class="w-full max-w-md">
		<!-- Logo -->
		<div class="text-center mb-8">
			<div class="inline-flex items-center gap-2.5">
				<div class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
					<svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
							d="M18 8h1a4 4 0 010 8h-1M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z" />
					</svg>
				</div>
				<span class="text-xl font-bold tracking-tight">NovuDesk</span>
			</div>
		</div>

		<slot />
	</div>
</div>
