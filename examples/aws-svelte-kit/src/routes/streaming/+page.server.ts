/** @type {import('./$types').PageServerLoad} */
export async function load() {
	// This data is available immediately
	const instant = { message: "This rendered instantly!", timestamp: Date.now() };

	// These promises are NOT awaited — SvelteKit streams them to the client
	const slow = new Promise((resolve) => {
		setTimeout(() => {
			resolve({
				message: "This was streamed after 2 seconds!",
				timestamp: Date.now(),
			});
		}, 2000);
	});

	const slower = new Promise((resolve) => {
		setTimeout(() => {
			resolve({
				message: "This was streamed after 4 seconds!",
				timestamp: Date.now(),
			});
		}, 4000);
	});

	return {
		instant,
		streamed: {
			slow,
			slower,
		},
	};
}
