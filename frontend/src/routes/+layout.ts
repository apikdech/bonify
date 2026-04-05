import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async () => {
	// Layout load - auth check happens in +layout.svelte via auth store
	return {};
};
