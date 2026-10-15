<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		class: className = '',
		children,
		...restProps
	}: {
		class?: string;
		children?: Snippet;
	} & HTMLAttributes<HTMLDivElement> = $props();
</script>

<div
	class={cn(
		'rounded-2xl border border-neutral-200/90 bg-white/90 p-6 text-neutral-900 shadow-sm backdrop-blur-xs transition-colors',
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</div>
