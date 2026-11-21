history.scrollRestoration = "manual";

const FADE_MS = 300;
const SETTLE_MS = 300;
const nonPage = /[^/]\.(?!html?$)[^./]+$/i;

let current = new URL(location.href);
let pending;
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

async function fetchDocument(url, signal) {
	const res = await fetch(url.href, { cache: "no-cache", signal });
	if (!res.ok || !(res.headers.get("content-type") || "").includes("text/html")) {
		throw new Error(`nav: ${res.status}`);
	}
	return new DOMParser().parseFromString(await res.text(), "text/html");
}

function settle() {
	const root = document.documentElement;
	const reached = new DOMMatrix(getComputedStyle(root, "::before").transform).a;
	if (!(reached > 0)) {
		delete root.dataset.navigating;
		return;
	}
	root.style.setProperty("--qr-navigated", reached);
	root.dataset.navigating = "done";
	setTimeout(() => {
		if (root.dataset.navigating === "done") delete root.dataset.navigating;
	}, SETTLE_MS);
}

function swap(doc) {
	for (const from of document.querySelectorAll("main [data-region]")) {
		const to = doc.querySelector(`main [data-region="${CSS.escape(from.dataset.region)}"]`);
		if (to) return [from, to];
	}
	return [document.querySelector("main"), doc.querySelector("main")];
}

function syncLinks(doc, from, to) {
	const twins = Map.groupBy(
		[...doc.querySelectorAll("a[href]")].filter((a) => !to.contains(a)),
		(a) => a.getAttribute("href"),
	);
	for (const a of document.querySelectorAll("a[href]")) {
		const twin = !from.contains(a) && twins.get(a.getAttribute("href"))?.shift();
		if (!twin) continue;
		a.className = twin.className;
		a.toggleAttribute("aria-current", twin.hasAttribute("aria-current"));
		if (twin.hasAttribute("aria-current")) a.setAttribute("aria-current", twin.getAttribute("aria-current"));
	}
}

function place(url, scroll) {
	const target = scroll === undefined && document.getElementById(decodeURIComponent(url.hash.slice(1)));
	target ? target.scrollIntoView() : scrollTo(0, scroll ?? 0);
}

async function navigate(url, push, scroll) {
	pending?.abort.abort();
	const abort = new AbortController();
	pending = { page: page(url), abort };
	document.documentElement.dataset.navigating = "";
	const doc = await fetchDocument(url, abort.signal).catch(() => null);
	if (abort.signal.aborted) return;
	pending = null;
	settle();

	const [outgoing, incoming] = doc ? swap(doc) : [];
	if (!incoming || !outgoing || assetURLs(doc) !== assetURLs(document)) {
		location.assign(url.href);
		return;
	}

	if (push) {
		saveScroll();
		history.pushState(null, "", url.href);
	}
	document.title = doc.title;
	syncLinks(doc, outgoing, incoming);
	outgoing.replaceWith(incoming);
	current = url;
	place(url, scroll);

	if (!matchMedia("(prefers-reduced-motion: reduce)").matches) {
		incoming.animate({ opacity: [0, 1] }, { duration: FADE_MS, easing: "ease" });
	}
	incoming.tabIndex = -1;
	incoming.focus({ preventScroll: true });
	dispatchEvent(new Event("quirenavigate"));
}

document.addEventListener("click", (e) => {
	if (e.defaultPrevented || e.button !== 0) return;
	if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

	const url = routable(e.target.closest("a"));
	if (!url) return;
	e.preventDefault();
	if (page(url) === pending?.page) return;
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
