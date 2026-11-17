// Table.tsx - a typed table with sortable columns.
import { useState, type ReactNode } from "react";

export interface Column<Row> {
  key: keyof Row & string;
  title: string;
  width?: number;
  render?: (row: Row) => ReactNode;
}

type Order = "asc" | "desc";

interface Props<Row extends { id: string }> {
  rows: readonly Row[];
  columns: Column<Row>[];
  caption?: string;
  onOpen?(row: Row): void;
}

const ARROW: Record<Order, string> = { asc: "▲", desc: "▼" };

function compare<Row>(a: Row, b: Row, key: keyof Row): number {
  return a[key] < b[key] ? -1 : a[key] > b[key] ? 1 : 0;
}

const first = <T,>(items: readonly T[]): T | undefined => items[0];
const pick = <Row extends object>(row: Row, key: keyof Row) => row[key];

export function Table<Row extends { id: string }>({ rows, columns, caption, onOpen }: Props<Row>) {
  const [sort, setSort] = useState<{ key: keyof Row; order: Order } | null>(null);
  const sorted = sort ? [...rows].sort((a, b) => compare(a, b, sort.key) * (sort.order === "asc" ? 1 : -1)) : rows;

  const toggle = (key: keyof Row) =>
    setSort((s) => (s?.key === key && s.order === "asc" ? { key, order: "desc" } : { key, order: "asc" }));

  return (
    <table className="table">
      {caption && <caption>{caption}</caption>}
      <thead>
        <tr>
          {columns.map((c) => (
            <th key={c.key} style={{ width: c.width ?? 120 }} onClick={() => toggle(c.key)}>
              {c.title} {sort?.key === c.key ? ARROW[sort.order] : null}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {sorted.map((row) => (
          <tr key={row.id} onDoubleClick={() => onOpen?.(row)}>
            {columns.map((c) => (
              <td key={c.key}>{c.render ? c.render(row) : String(row[c.key])}</td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
