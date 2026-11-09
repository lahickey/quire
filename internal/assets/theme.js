
var root = document.documentElement;

function set(theme) {
	root.setAttribute("data-theme", theme);
	root.style.colorScheme = theme;
}
function preferred() {
	try {
		if (window.matchMedia) {
			if (matchMedia("(prefers-color-scheme: dark)").matches) return "dark";
			if (matchMedia("(prefers-color-scheme: light)").matches) return "light";
		}
	} catch (e) {}
	return root.getAttribute("data-theme-default") === "dark" ? "dark" : "light";
}

var stored = null;
try {
	stored = localStorage.getItem("theme");
} catch (e) {
}
set(stored === "light" || stored === "dark" ? stored : preferred());

document.addEventListener("click", function (e) {
	if (!e.target.closest("[data-theme-toggle]")) return;
	var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
	var apply = function () { set(next); };
	try {
		if (document.startViewTransition && !matchMedia("(prefers-reduced-motion: reduce)").matches) {
			document.startViewTransition(apply);
		} else {
			apply();
		}
	} catch (err) {
		apply();
	}
	try {
		localStorage.setItem("theme", next);
	} catch (e) {}
});
