try {
	const os = ["dark", "light"].find((t) => matchMedia(`(prefers-color-scheme: ${t})`).matches);
	const at = document.documentElement.dataset.root || "";
	const theme = localStorage.getItem("theme" + at) || os;
	if (theme) document.documentElement.dataset.theme = theme;
	const palette = localStorage.getItem("palette" + at);
	if (palette) document.documentElement.dataset.palette = palette;
} catch {}
