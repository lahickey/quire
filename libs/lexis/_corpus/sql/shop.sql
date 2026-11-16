-- Schema and reports for a small bookshop.
-- Run with: psql -v ON_ERROR_STOP=1 -f shop.sql

BEGIN;

CREATE TABLE IF NOT EXISTS authors (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    sort_name   VARCHAR(200) NOT NULL,
    born        DATE,
    country     CHAR(2) DEFAULT 'GB',
    UNIQUE (sort_name, born)
);

CREATE TABLE IF NOT EXISTS books (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    isbn        CHAR(13) NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    author_id   BIGINT NOT NULL REFERENCES authors (id) ON DELETE RESTRICT,
    pence       INTEGER NOT NULL CHECK (pence > 0),
    published   DATE NOT NULL,
    format      VARCHAR(16) NOT NULL DEFAULT 'paperback'
                CHECK (format IN ('hardback', 'paperback', 'ebook')),
    in_print    BOOLEAN NOT NULL DEFAULT TRUE,
    rating      NUMERIC(3, 2)
);

CREATE TABLE IF NOT EXISTS sales (
    id          BIGSERIAL PRIMARY KEY,
    book_id     BIGINT NOT NULL REFERENCES books (id),
    sold_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    quantity    SMALLINT NOT NULL DEFAULT 1,
    discount    REAL NOT NULL DEFAULT 0.0,
    channel     VARCHAR(8) NOT NULL
);

CREATE INDEX IF NOT EXISTS sales_book_time ON sales (book_id, sold_at DESC);
CREATE INDEX IF NOT EXISTS books_title_lower ON books (lower(title));

/* A few rows to work with. Quotes inside strings are doubled: 'it''s'. */
INSERT INTO authors (name, sort_name, born, country) VALUES
    ('Ines Marsh', 'Marsh, Ines', '1978-04-02', 'GB'),
    ('Sam Calloway', 'Calloway, Sam', NULL, 'IE'),
    ('Dee O''Kafor', 'O''Kafor, Dee', '1985-11-19', 'NG')
ON CONFLICT DO NOTHING;

INSERT INTO books (isbn, title, author_id, pence, published, format, rating)
SELECT '9781402894626', 'The Salt Road', a.id, 2500, DATE '2026-03-05', 'hardback', 4.35
FROM authors AS a
WHERE a.sort_name = 'Marsh, Ines';

UPDATE books
SET pence = ROUND(pence * 0.9), in_print = FALSE
WHERE published < CURRENT_DATE - INTERVAL '10 years'
  AND format <> 'ebook';

DELETE FROM sales WHERE quantity <= 0 OR discount NOT BETWEEN 0 AND 1;

-- Revenue by author for the last quarter, best first.
CREATE OR REPLACE VIEW author_revenue AS
SELECT
    a.id,
    a.name,
    COUNT(DISTINCT b.id)                                        AS titles,
    COALESCE(SUM(s.quantity), 0)                                AS copies,
    COALESCE(SUM(s.quantity * b.pence * (1 - s.discount)), 0)::BIGINT AS pence,
    MAX(s.sold_at)                                              AS last_sale
FROM authors a
LEFT JOIN books b ON b.author_id = a.id
LEFT JOIN sales s ON s.book_id = b.id
                 AND s.sold_at >= date_trunc('quarter', CURRENT_DATE) - INTERVAL '3 months'
GROUP BY a.id, a.name
HAVING COUNT(b.id) > 0
ORDER BY pence DESC NULLS LAST, a.name ASC;

-- Each book's rank within its format, and a running total by month.
WITH monthly AS (
    SELECT
        b.id AS book_id,
        b.format,
        date_trunc('month', s.sold_at) AS month,
        SUM(s.quantity) AS copies
    FROM books b
    INNER JOIN sales s ON s.book_id = b.id
    WHERE s.channel IN ('web', 'shop') AND b.in_print IS TRUE
    GROUP BY b.id, b.format, date_trunc('month', s.sold_at)
),
ranked AS (
    SELECT
        m.*,
        RANK() OVER (PARTITION BY m.format, m.month ORDER BY m.copies DESC) AS place,
        SUM(m.copies) OVER (PARTITION BY m.book_id ORDER BY m.month
                            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running
    FROM monthly m
)
SELECT
    r.month,
    r.format,
    b.title,
    r.copies,
    r.running,
    CASE
        WHEN r.place = 1 THEN 'top'
        WHEN r.place <= 3 THEN 'podium'
        ELSE 'rest'
    END AS band,
    EXISTS (SELECT 1 FROM sales x WHERE x.book_id = r.book_id AND x.discount > 0.5) AS deep_discount
FROM ranked r
JOIN books b ON b.id = r.book_id
WHERE r.place <= 10
  AND b.title LIKE '%Road%' ESCAPE '\'
  AND b.rating IS NOT NULL
UNION ALL
SELECT NULL, 'all', 'TOTAL', SUM(copies), NULL, 'total', FALSE FROM monthly
ORDER BY 1 DESC, 2, 4 DESC
LIMIT 50 OFFSET 0;

ALTER TABLE books ADD COLUMN IF NOT EXISTS pages INTEGER;
GRANT SELECT ON author_revenue TO reporting;
DROP TABLE IF EXISTS scratch CASCADE;

COMMIT;
