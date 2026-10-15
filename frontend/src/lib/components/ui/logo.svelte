<script lang="ts">
	let {
		href = '/',
		size = 'md',
		class: customClass = ''
	}: {
		href?: string;
		size?: 'sm' | 'md' | 'lg';
		class?: string;
	} = $props();

	const sizeClasses = {
		sm: 'text-xl sm:text-2xl',
		md: 'text-2xl sm:text-3xl',
		lg: 'text-3xl sm:text-4xl'
	};
</script>

<a
	{href}
	class="group inline-flex items-center font-sans font-extrabold tracking-tighter text-neutral-950 transition-opacity hover:opacity-85 {sizeClasses[
		size
	]} {customClass}"
>
	<span class="relative pb-1">
		chs<span class="text-neutral-900">.</span>
		<span
			class="absolute bottom-0 left-0 h-[2.5px] w-6 bg-neutral-950 transition-all duration-300 group-hover:w-full"
		></span>
	</span>
</a>
