/**
 * Curated architectural and serene daily photographic backdrops.
 * Deterministic day-of-year selection ensures:
 * 1. The exact same photo is rendered all day regardless of refreshes or sessions.
 * 2. Automatic daily transition at midnight.
 * 3. Zero third-party runtime API rate-limiting or network blocking.
 * 4. High-contrast typography legibility with subtle tone overlays.
 */

export interface DailyBackdrop {
	id: string;
	url: string;
	title: string;
	location: string;
	photographer: string;
	creditUrl: string;
	overlayOpacity: number; // 0.0 to 1.0 subtle dark/warm overlay
}

export const CURATED_BACKDROPS: DailyBackdrop[] = [
	{
		id: 'minimal-concrete-facade',
		url: 'https://images.unsplash.com/photo-1513694203232-719a280e022f?auto=format&fit=crop&w=1600&q=80',
		title: 'Minimalist Monolith',
		location: 'Copenhagen, Denmark',
		photographer: 'Simone Hutsch',
		creditUrl: 'https://unsplash.com',
		overlayOpacity: 0.15
	},
	{
		id: 'brutalist-pavilion-light',
		url: 'https://images.unsplash.com/photo-1486406146926-c627a92ad1ab?auto=format&fit=crop&w=1600&q=80',
		title: 'Geometry & Horizon',
		location: 'Tokyo, Japan',
		photographer: 'Ryoji Iwata',
		creditUrl: 'https://unsplash.com',
		overlayOpacity: 0.18
	},
	{
		id: 'quiet-gallery-colonnade',
		url: 'https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=1600&q=80',
		title: 'Limestone Rhythm',
		location: 'Zurich, Switzerland',
		photographer: 'Ricardo Gomez Angel',
		creditUrl: 'https://unsplash.com',
		overlayOpacity: 0.12
	},
	{
		id: 'modernist-atrium-shadows',
		url: 'https://images.unsplash.com/photo-1517581177682-a085bb7ffb15?auto=format&fit=crop&w=1600&q=80',
		title: 'Nordic Light Axis',
		location: 'Oslo, Norway',
		photographer: 'Mikita Yo',
		creditUrl: 'https://unsplash.com',
		overlayOpacity: 0.16
	},
	{
		id: 'sculptural-cantilever',
		url: 'https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=1600&q=80',
		title: 'Sanctuary Interior',
		location: 'Basel, Switzerland',
		photographer: 'Danist Soh',
		creditUrl: 'https://unsplash.com',
		overlayOpacity: 0.15
	},
	{
		id: 'stone-courtyard-calm',
		url: 'https://images.unsplash.com/photo-1518780664697-55e3ad937233?auto=format&fit=crop&w=1600&q=80',
		title: 'Terrace at Dawn',
		location: 'Kyoto, Japan',
		photographer: 'Timo Volz',
		creditUrl: 'https://unsplash.com',
		overlayOpacity: 0.14
	},
	{
		id: 'timber-structure-zenith',
		url: 'https://images.unsplash.com/photo-1479839672679-a46483c0e7c8?auto=format&fit=crop&w=1600&q=80',
		title: 'Warm Spatial Grid',
		location: 'Stockholm, Sweden',
		photographer: 'Jonas Jacobsson',
		creditUrl: 'https://unsplash.com',
		overlayOpacity: 0.15
	}
];

export const FALLBACK_BACKDROP: DailyBackdrop = CURATED_BACKDROPS[0];

/**
 * Returns the day index (0..365) for a given date.
 */
export function getDayOfYear(date: Date = new Date()): number {
	const start = new Date(Date.UTC(date.getUTCFullYear(), 0, 0));
	const diff = date.getTime() - start.getTime();
	const oneDay = 1000 * 60 * 60 * 24;
	return Math.floor(diff / oneDay);
}

/**
 * Resolves the deterministic backdrop for a specific calendar date.
 * Same date -> identical backdrop.
 */
export function getDailyBackdrop(date: Date = new Date()): DailyBackdrop {
	if (!CURATED_BACKDROPS || CURATED_BACKDROPS.length === 0) {
		return FALLBACK_BACKDROP;
	}
	const dayOfYear = getDayOfYear(date);
	const year = date.getUTCFullYear();
	// Mathematical hash combining year and dayOfYear
	const index = Math.abs((year * 366 + dayOfYear) % CURATED_BACKDROPS.length);
	return CURATED_BACKDROPS[index] ?? FALLBACK_BACKDROP;
}
