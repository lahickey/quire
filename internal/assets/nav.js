
history.scrollRestoration = "manual";

const scrollPositions = new Map();
let currentURL = location.href;

const hardNav = (url) => { location.href = url.href; };

function routable(anchor) {
	if (!anchor || !anchor.href) return null;
	if (anchor.target && anchor.target !== "_self") return null;
	if (anchor.hasAttribute("download")) return null;
	if (anchor.rel.includes("external")) return null;

	const url = new URL(anchor.href, location.href);
	if (url.origin !== location.origin) return null;

	if (url.pathname === location.pathname && url.search === location.search && url.hash) {
		return null;
	}

	const file = url.pathname.slice(url.pathname.lastIndexOf("/") + 1);
	const dot = file.lastIndexOf(".");
	if (dot > 0 && !/^\.html?$/i.test(file.slice(dot))) return null;

	return url;
}

async function fetchDocument(url) {
	const res = await fetch(url.href, { cache: "no-cache" });
	if (!res.ok) throw new Error(`nav: ${res.status}`);
	if (!(res.headers.get("content-type") || "").includes("text/html")) {
		throw new Error("nav: not html");
	}
	return new DOMParser().parseFromString(await res.text(), "text/html");
}

function buildID(doc) {
	return [...doc.querySelectorAll('head link[rel="stylesheet"][href], head script[src]')]
		.map((el) => el.getAttribute("href") || el.getAttribute("src"))
		.sort()
		.join("|");
}

function applyScroll(url, push) {
	if (push) {
		const target = url.hash && document.querySelector(url.hash);
		target ? target.scrollIntoView() : window.scrollTo(0, 0);
	} else {
		window.scrollTo(0, scrollPositions.get(url.href) || 0);
	}
}

function stripSlash(p) {
	return p.replace(/\/+$/, "") || "/";
}

function samePage(url) {
	return stripSlash(url.pathname) === stripSlash(location.pathname) && url.search === location.search;
}

const FADE_MS = 300;

function fadeIn(main) {
	if (!FADE_MS || document.hidden ||
		window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
	main.animate({ opacity: [0, 1] }, { duration: FADE_MS, easing: "ease" });
}

function focusMain(main) {
	main.setAttribute("tabindex", "-1");
	main.focus({ preventScroll: true });
}

function syncNav(doc) {
	const incoming = doc.querySelectorAll(".nav-link");
	document.querySelectorAll(".nav-link").forEach((a, i) => {
		if (incoming[i]) a.className = incoming[i].className;
	});
}

let navSeq = 0;

async function navigate(url, push) {
	const seq = ++navSeq;
	let doc;
	try {
		doc = await fetchDocument(url);
	} catch {
		hardNav(url);
		return;
	}
	if (seq !== navSeq) return;

	const incoming = doc.querySelector("main");
	const current = document.querySelector("main");
	if (!incoming || !current || buildID(doc) !== buildID(document)) {
		hardNav(url);
		return;
	}

	scrollPositions.set(currentURL, window.scrollY);
	document.title = doc.title;
	syncNav(doc);
	current.replaceWith(incoming);
	if (push) history.pushState({ engineNav: true }, "", url.href);
	currentURL = url.href;
	applyScroll(url, push);

	fadeIn(incoming);
	focusMain(incoming);
}

document.addEventListener("click", (e) => {
	if (e.defaultPrevented || e.button !== 0) return;
	if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

	const url = routable(e.target.closest("a"));
	if (!url) return;
	e.preventDefault();
	if (samePage(url)) return;
	navigate(url, true);
});

window.addEventListener("popstate", () => {
	const url = new URL(location.href);
	if (url.href === currentURL) return;
	if (url.pathname === new URL(currentURL).pathname && url.hash) {
		currentURL = url.href;
		return;
	}
	navigate(url, false);
});

window.addEventListener("pageshow", (e) => {
	if (e.persisted) currentURL = location.href;
});
