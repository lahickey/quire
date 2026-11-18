const seen = new IntersectionObserver((entries) => {
	for (const e of entries) {
		if (e.isIntersecting) e.target.play().catch(() => {});
		else e.target.pause();
	}
}, { threshold: 0.25 });

function watch() {
	if (matchMedia("(prefers-reduced-motion: reduce)").matches) return;
	for (const v of document.querySelectorAll(".figure video")) {
		v.controls = false;
		seen.observe(v);
	}
}

addEventListener("DOMContentLoaded", watch);
addEventListener("quirenavigate", watch);
