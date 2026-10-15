<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		variant = 'default',
		class: className = '',
		children,
		...restProps
	}: {
		variant?: 'default' | 'success' | 'warning' | 'destructive' | 'outline';
		class?: string;
		children?: Snippet;
	} & HTMLAttributes<HTMLSpanElement> = $props();

	const variants = {
		default: 'bg-neutral-900 text-white border-neutral-900',
		success: 'bg-emerald-50 text-emerald-800 border-emerald-200',
		warning: 'bg-amber-50 text-amber-800 border-amber-200',
		destructive: 'bg-rose-50 text-rose-800 border-rose-200',
		outline: 'border-neutral-300 bg-white/70 text-neutral-800'
	};
</script>

<span
	class={cn(
		'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium tracking-tight transition-colors select-none',
		variants[variant] || variants.default,
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</span>
