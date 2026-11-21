document.addEventListener("keydown", (e) => {
	const box = e.target.closest?.(".search-box");
	if (!box) return;

	if (e.key === "Escape") {
		document.activeElement.blur();
		return;
	}

	const first = box.querySelector("a");
	if (e.key === "Enter" && e.target === box.elements.q && first) {
		e.preventDefault();
		first.click();
		return;
	}

	const step = { ArrowDown: 1, ArrowUp: -1 }[e.key];
	if (!step) return;
	e.preventDefault();
	const stops = [box.elements.q, ...box.querySelectorAll("a")];
	stops[(stops.indexOf(document.activeElement) + step + stops.length) % stops.length].focus();
});
