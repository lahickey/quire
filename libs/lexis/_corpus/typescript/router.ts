// router.ts - a typed client-side router with guards.
import type { Component } from "./component";
import { createHistory, type History } from "./history";

export type Params = Record<string, string>;

export interface Route<P extends Params = Params> {
  readonly path: string;
  component: () => Promise<Component<P>>;
  guard?: Guard;
  children?: Route[];
  meta?: { title?: string; requiresAuth: boolean };
}

type Guard = (to: Match, from: Match | null) => boolean | string | Promise<boolean | string>;

interface Match<P extends Params = Params> {
  route: Route<P>;
  params: P;
  query: URLSearchParams;
}

enum Outcome {
  Allowed,
  Redirected,
  Blocked = -1,
}

const PARAM = /:([A-Za-z_]\w*)(\?)?/g;
const DEFAULT_TITLE = "Untitled";

/** Turn "/users/:id/posts/:slug?" into a matcher. */
function compile(path: string): { regex: RegExp; keys: string[] } {
  const keys: string[] = [];
  const source = path.replace(PARAM, (_whole, key: string, optional?: string) => {
    keys.push(key);
    return optional ? "([^/]*)?" : "([^/]+)";
  });
  return { regex: new RegExp(`^${source}/?$`), keys };
}

export class NavigationError extends Error {
  constructor(
    public readonly outcome: Outcome,
    public readonly target: string,
  ) {
    super(`navigation to ${target} was ${Outcome[outcome].toLowerCase()}`);
    this.name = "NavigationError";
  }
}

export class Router {
  private readonly compiled = new Map<Route, ReturnType<typeof compile>>();
  private current: Match | null = null;
  private listeners: Array<(match: Match) => void> = [];

  constructor(
    private readonly routes: readonly Route[],
    private readonly history: History = createHistory(),
  ) {
    for (const route of this.flatten(routes)) {
      this.compiled.set(route, compile(route.path));
    }
    this.history.listen((url) => void this.navigate(url, { replace: true }));
  }

  private *flatten(routes: readonly Route[], prefix = ""): Generator<Route> {
    for (const route of routes) {
      const full = { ...route, path: prefix + route.path };
      yield full;
      if (route.children?.length) yield* this.flatten(route.children, full.path);
    }
  }

  resolve(url: string): Match | undefined {
    const { pathname, searchParams } = new URL(url, "http://local");
    for (const [route, { regex, keys }] of this.compiled) {
      const found = regex.exec(pathname);
      if (!found) continue;
      const params = Object.fromEntries(keys.map((k, i) => [k, decodeURIComponent(found[i + 1] ?? "")]));
      return { route, params, query: searchParams };
    }
    return undefined;
  }

  async navigate(url: string, options: { replace?: boolean } = {}): Promise<Outcome> {
    const match = this.resolve(url);
    if (match === undefined) throw new NavigationError(Outcome.Blocked, url);

    const verdict = (await match.route.guard?.(match, this.current)) ?? true;
    if (typeof verdict === "string") {
      await this.navigate(verdict, options);
      return Outcome.Redirected;
    }
    if (!verdict) return Outcome.Blocked;

    const component = await match.route.component();
    component.mount(match.params as never);
    document.title = match.route.meta?.title ?? DEFAULT_TITLE;
    options.replace ? this.history.replace(url) : this.history.push(url);
    this.current = match;
    this.listeners.forEach((fn) => fn(match));
    return Outcome.Allowed;
  }

  onChange(fn: (match: Match) => void): () => void {
    this.listeners.push(fn);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== fn);
    };
  }
}

export function defineRoutes<const T extends readonly Route[]>(routes: T): T {
  return routes;
}

const isSignedIn = (): boolean => Boolean(sessionStorage.getItem("token"));

export const routes = defineRoutes([
  { path: "/", component: () => import("./pages/home"), meta: { title: "Home", requiresAuth: false } },
  {
    path: "/users/:id",
    component: () => import("./pages/user"),
    guard: (to) => (isSignedIn() ? true : `/login?next=${encodeURIComponent(to.route.path)}`),
    children: [{ path: "/posts/:slug?", component: () => import("./pages/post") }],
  },
] as const satisfies readonly Route[]);

declare global {
  interface Window {
    __router?: Router;
  }
}

window.__router ??= new Router(routes);
let retries: number = 0x10 >>> 2;
let nothing: null | undefined = retries > 3.5e0 ? null : undefined;
