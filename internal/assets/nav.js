history.scrollRestoration = "manual";

const FADE_MS = 300;
const nonPage = /[^/]\.(?!html?$)[^./]+$/i;

let current = new URL(location.href);
let latest = 0;
let saving;

const page = (url) => url.pathname.replace(/\/+$/, "") + url.search;

const assetURLs = (doc) =>
	[...doc.querySelectorAll('head link[rel="stylesheet"][href], head script[src]')]
		.map((el) => el.getAttribute("href") || el.getAttribute("src"))
		.sort()
		.join("|");

const saveScroll = () => history.replaceState({ scroll: scrollY }, "");

function routable(anchor) {
	if (!anchor?.href) return null;
	if (anchor.target && anchor.target !== "_self") return null;
	if (anchor.hasAttribute("download") || anchor.rel.includes("external")) return null;
	const url = new URL(anchor.href);
	return url.origin === location.origin && !nonPage.test(url.pathname) ? url : null;
}

async function fetchDocument(url) {
	const res = await fetch(url.href, { cache: "no-cache" });
	if (!res.ok || !(res.headers.get("content-type") || "").includes("text/html")) {
		throw new Error(`nav: ${res.status}`);
	}
	return new DOMParser().parseFromString(await res.text(), "text/html");
}

function syncNav(doc) {
	for (const a of document.querySelectorAll(".nav-link")) {
		const twin = doc.querySelector(`.nav-link[href="${CSS.escape(a.getAttribute("href"))}"]`);
		if (twin) a.className = twin.className;
	}
}

function place(url, scroll) {
	const target = scroll === undefined && document.getElementById(decodeURIComponent(url.hash.slice(1)));
	target ? target.scrollIntoView() : scrollTo(0, scroll ?? 0);
}

async function navigate(url, push, scroll) {
	const seq = ++latest;
	const doc = await fetchDocument(url).catch(() => null);
	if (seq !== latest) return;

	const incoming = doc?.querySelector("main");
	const main = document.querySelector("main");
	if (!incoming || !main || assetURLs(doc) !== assetURLs(document)) {
		location.assign(url.href);
		return;
	}

	if (push) {
		saveScroll();
		history.pushState(null, "", url.href);
	}
	document.title = doc.title;
	syncNav(doc);
	main.replaceWith(incoming);
	current = url;
	place(url, scroll);

	if (!matchMedia("(prefers-reduced-motion: reduce)").matches) {
		incoming.animate({ opacity: [0, 1] }, { duration: FADE_MS, easing: "ease" });
	}
	incoming.tabIndex = -1;
	incoming.focus({ preventScroll: true });
}

document.addEventListener("click", (e) => {
	if (e.defaultPrevented || e.button !== 0) return;
	if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

	const url = routable(e.target.closest("a"));
	if (!url) return;
	e.preventDefault();
	if (page(url) !== page(current)) navigate(url, true);
	else if (url.hash) location.assign(current.pathname + current.search + url.hash);
});

addEventListener("popstate", (e) => {
	const url = new URL(location.href);
	if (page(url) !== page(current)) {
		navigate(url, false, e.state?.scroll);
		return;
	}
	current = url;
	place(url, e.state?.scroll);
});

addEventListener("scroll", () => {
	clearTimeout(saving);
	saving = setTimeout(saveScroll, 200);
}, { passive: true });

addEventListener("DOMContentLoaded", () => {
	if (history.state?.scroll) scrollTo(0, history.state.scroll);
});
