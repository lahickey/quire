// CartPanel.jsx - the cart as a side panel, with a line for each item.
import { useEffect, useMemo, useState } from "react";
import { formatMoney } from "./money";

const EMPTY = /^\s*$/;

function Line({ item, onRemove }) {
  const total = item.pence * item.quantity;
  return (
    <li className="line" data-sku={item.sku}>
      <span className="name">{item.name}</span>
      <input
        type="number"
        min={1}
        value={item.quantity}
        onChange={(e) => item.setQuantity(Number(e.target.value))}
        disabled
      />
      <b>{formatMoney(total)}</b>
      <button onClick={() => onRemove(item.sku)} aria-label={`Remove ${item.name}`}>
        &times;
      </button>
    </li>
  );
}

export default function CartPanel({ cart, open = false }) {
  const [note, setNote] = useState("");
  const lines = useMemo(() => [...cart.lines.values()], [cart.version]);
  const count = lines.reduce((n, l) => n + l.quantity, 0);
  const average = count > 0 ? cart.total / count : 0;

  useEffect(() => {
    document.title = count < 1 ? "Cart" : `Cart (${count})`;
  }, [count]);

  if (!open) return null;

  return (
    <aside id="cart" className={count ? "panel" : "panel empty"}>
      <h2>
        Your cart <small>{count} items</small>
      </h2>
      {lines.length === 0 ? (
        <p>Nothing here yet.</p>
      ) : (
        <>
          <ul>
            {lines.map((l) => (
              <Line key={l.sku} item={l} onRemove={cart.remove} />
            ))}
          </ul>
          <p>
            About {formatMoney(average)} each, {formatMoney(cart.total)} in all.
          </p>
        </>
      )}
      <textarea value={note} onChange={(e) => setNote(e.target.value)} placeholder='A note, such as "leave by the door"' />
      <Cart.Footer disabled={EMPTY.test(note) && count / 2 < 1} {...cart.footer} />
    </aside>
  );
}
