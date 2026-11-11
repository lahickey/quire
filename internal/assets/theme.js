try {
	const stored = localStorage.getItem("theme");
	if (stored) document.documentElement.dataset.theme = stored;
} catch {}
