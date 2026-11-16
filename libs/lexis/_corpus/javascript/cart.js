// cart.js - a shopping cart kept in localStorage, with a small event bus.
"use strict";

import { formatMoney } from "./money.js";
import * as api from "./api.js";

const STORAGE_KEY = "cart:v2";
const TAX_RATE = 0.2;
const MAX_QUANTITY = 99;
const SKU_PATTERN = /^[A-Z]{2,4}-\d{3,}$/i;

/**
 * @typedef {Object} Line
 * @property {string} sku
 * @property {number} quantity
 * @property {number} pence
 */

class EventBus {
  #handlers = new Map();

  on(name, handler) {
    const list = this.#handlers.get(name) ?? [];
    this.#handlers.set(name, [...list, handler]);
    return () => this.off(name, handler);
  }

  off(name, handler) {
    const list = this.#handlers.get(name);
    if (!list) return;
    this.#handlers.set(name, list.filter((h) => h !== handler));
  }

  emit(name, detail = {}) {
    for (const handler of this.#handlers.get(name) || []) {
      try {
        handler(detail);
      } catch (err) {
        console.error(`handler for "${name}" failed:`, err);
      }
    }
  }
}

export class Cart extends EventBus {
  static #instance = null;

  /** @type {Map<string, Line>} */
  lines = new Map();

  static shared() {
    return (Cart.#instance ??= new Cart().load());
  }

  load() {
    try {
      const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]");
      for (const line of saved) this.lines.set(line.sku, line);
    } catch {
      localStorage.removeItem(STORAGE_KEY);
    }
    return this;
  }

  save() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify([...this.lines.values()]));
    this.emit("change", { count: this.count });
  }

  add(sku, pence, quantity = 1) {
    if (!SKU_PATTERN.test(sku)) throw new TypeError("bad sku: " + sku);
    const line = this.lines.get(sku) ?? { sku, pence, quantity: 0 };
    line.quantity = Math.min(line.quantity + quantity, MAX_QUANTITY);
    this.lines.set(sku, line);
    this.save();
  }

  remove(sku) {
    if (this.lines.delete(sku)) this.save();
  }

  get count() {
    let n = 0;
    for (const { quantity } of this.lines.values()) n += quantity;
    return n;
  }

  get subtotal() {
    return [...this.lines.values()].reduce((sum, l) => sum + l.pence * l.quantity, 0);
  }

  totals() {
    const subtotal = this.subtotal;
    const tax = Math.round(subtotal * TAX_RATE);
    return { subtotal, tax, total: subtotal + tax };
  }

  async checkout({ retries = 2, signal } = {}) {
    for (let attempt = 0; attempt <= retries; attempt++) {
      try {
        const order = await api.post("/orders", { lines: [...this.lines.values()] }, { signal });
        this.lines.clear();
        this.save();
        return order;
      } catch (err) {
        if (err.name === "AbortError" || attempt === retries) throw err;
        await new Promise((resolve) => setTimeout(resolve, 250 * 2 ** attempt));
      }
    }
  }
}

function* chunks(items, size) {
  for (let i = 0; i < items.length; i += size) yield items.slice(i, i + size);
}

export function render(cart, root = document.querySelector("#cart")) {
  const { subtotal, tax, total } = cart.totals();
  const rows = [...cart.lines.values()]
    .map((l) => `<li data-sku="${l.sku}">${l.quantity} × ${l.sku} <b>${formatMoney(l.pence * l.quantity)}</b></li>`)
    .join("\n");
  root.innerHTML = `
    <ul>${rows}</ul>
    <p>Subtotal ${formatMoney(subtotal)}, tax ${formatMoney(tax)}</p>
    <p class="${total > 100_00 ? "big" : ""}">Total ${formatMoney(total)}</p>`;
  root.hidden = cart.count === 0;
}

const cart = Cart.shared();
cart.on("change", () => render(cart));
document.addEventListener("click", (event) => {
  const item = event.target.closest("[data-sku]");
  if (item && event.altKey) cart.remove(item.dataset.sku);
});
if (typeof window !== "undefined" && window.location.hash === "#debug") {
  console.table([...chunks([...cart.lines.keys()], 3)]);
  console.log(0xff, 1e3, 0b1010, 12n, null, undefined, NaN);
}
