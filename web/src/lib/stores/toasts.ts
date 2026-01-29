import { writable, derived } from 'svelte/store';

interface Toast {
	id: string;
	message: string;
	type: 'success' | 'error' | 'info' | 'loading';
}

function createToastStore() {
	const { subscribe, update } = writable<Toast[]>([]);

	function remove(id: string) {
		update((toasts) => toasts.filter((t) => t.id !== id));
	}

	function add(message: string, type: Toast['type'] = 'info', duration = 5000): string {
		const id = crypto.randomUUID();
		update((toasts) => [...toasts, { id, message, type }]);

		if (type !== 'loading' && duration > 0) {
			setTimeout(() => remove(id), duration);
		}

		return id;
	}

	return {
		subscribe,
		success: (message: string) => add(message, 'success'),
		error: (message: string) => add(message, 'error', 8000),
		info: (message: string) => add(message, 'info'),
		loading: (message: string) => add(message, 'loading', 0),
		remove,
		clear: () => update(() => [])
	};
}

export const toasts = createToastStore();
