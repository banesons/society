<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		variant = 'primary',
		size = 'md',
		class: className = '',
		children,
		...restProps
	}: {
		variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'link' | 'destructive';
		size?: 'sm' | 'md' | 'lg';
		class?: string;
		children?: Snippet;
	} & HTMLButtonAttributes = $props();

	const variants = {
		primary:
			'bg-neutral-950 text-white hover:bg-neutral-800 active:scale-[0.98] shadow-sm tracking-tight',
		destructive:
			'bg-rose-600 text-white hover:bg-rose-700 active:scale-[0.98] shadow-sm tracking-tight',
		secondary:
			'bg-neutral-200 text-neutral-900 hover:bg-neutral-300 active:scale-[0.98] tracking-tight',
		outline:
			'border border-neutral-300 bg-white/60 text-neutral-900 hover:bg-neutral-100 active:scale-[0.98] tracking-tight',
		ghost: 'text-neutral-600 hover:text-neutral-950 hover:bg-neutral-200/60 transition-colors',
		link: 'text-neutral-950 underline underline-offset-4 decoration-1 hover:opacity-75 p-0'
	};

	const sizes = {
		sm: 'px-3 py-1.5 text-xs',
		md: 'px-4 py-2 text-sm',
		lg: 'px-6 py-3 text-base font-medium'
	};
</script>

<button
	class={cn(
		'inline-flex cursor-pointer items-center justify-center gap-2 rounded-full font-medium transition-all duration-150 focus-visible:ring-2 focus-visible:ring-neutral-900 focus-visible:outline-none disabled:pointer-events-none disabled:opacity-40',
		variants[variant],
		variant !== 'link' && sizes[size],
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</button>
