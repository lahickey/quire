try {
	const os = ["dark", "light"].find((t) => matchMedia(`(prefers-color-scheme: ${t})`).matches);
	const theme = localStorage.getItem("theme") || os;
	if (theme) document.documentElement.dataset.theme = theme;
} catch {}
