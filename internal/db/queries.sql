-- name: ListPostingsAsc :many
SELECT
    id,
    transaction_id,
    date,
    payee,
    account,
    commodity,
    quantity,
    amount,
    original_amount,
    status,
    tag_recurring,
    tag_period,
    transaction_begin_line,
    transaction_end_line,
    file_name,
    forecast,
    note,
    transaction_note,
    transaction_hash
FROM postings
WHERE forecast = ?1
  AND (?2 = 0 OR date >= ?3)
  AND (?4 = 0 OR date <= ?5)
  AND (?6 = '' OR status = ?6)
  AND (?7 = 0 OR amount > 0)
  AND (?8 = '' OR account = ?8)
  AND (json_array_length(?9) = 0 OR commodity IN (SELECT value FROM json_each(?9)))
  AND (json_array_length(?10) = 0 OR EXISTS (
        SELECT 1 FROM json_each(?10)
        WHERE postings.account = value OR postings.account LIKE value || ':%'
    ))
  AND (json_array_length(?11) = 0 OR NOT EXISTS (
        SELECT 1 FROM json_each(?11)
        WHERE postings.account = value OR postings.account LIKE value || ':%'
    ))
  AND (json_array_length(?12) = 0 OR EXISTS (
        SELECT 1 FROM json_each(?12)
        WHERE postings.account LIKE value
    ))
  AND (json_array_length(?13) = 0 OR NOT EXISTS (
        SELECT 1 FROM json_each(?13)
        WHERE postings.account LIKE value
    ))
  AND (json_array_length(?14) = 0 OR account NOT IN (SELECT value FROM json_each(?14)))
ORDER BY date ASC, amount DESC, account ASC
LIMIT CASE WHEN ?16 > 0 THEN ?16 ELSE -1 END
OFFSET ?15;

-- name: ListPostingsDesc :many
SELECT
    id,
    transaction_id,
    date,
    payee,
    account,
    commodity,
    quantity,
    amount,
    original_amount,
    status,
    tag_recurring,
    tag_period,
    transaction_begin_line,
    transaction_end_line,
    file_name,
    forecast,
    note,
    transaction_note,
    transaction_hash
FROM postings
WHERE forecast = ?1
  AND (?2 = 0 OR date >= ?3)
  AND (?4 = 0 OR date <= ?5)
  AND (?6 = '' OR status = ?6)
  AND (?7 = 0 OR amount > 0)
  AND (?8 = '' OR account = ?8)
  AND (json_array_length(?9) = 0 OR commodity IN (SELECT value FROM json_each(?9)))
  AND (json_array_length(?10) = 0 OR EXISTS (
        SELECT 1 FROM json_each(?10)
        WHERE postings.account = value OR postings.account LIKE value || ':%'
    ))
  AND (json_array_length(?11) = 0 OR NOT EXISTS (
        SELECT 1 FROM json_each(?11)
        WHERE postings.account = value OR postings.account LIKE value || ':%'
    ))
  AND (json_array_length(?12) = 0 OR EXISTS (
        SELECT 1 FROM json_each(?12)
        WHERE postings.account LIKE value
    ))
  AND (json_array_length(?13) = 0 OR NOT EXISTS (
        SELECT 1 FROM json_each(?13)
        WHERE postings.account LIKE value
    ))
  AND (json_array_length(?14) = 0 OR account NOT IN (SELECT value FROM json_each(?14)))
ORDER BY date DESC, amount DESC, account ASC
LIMIT CASE WHEN ?16 > 0 THEN ?16 ELSE -1 END
OFFSET ?15;

-- name: GroupPostingSums :many
SELECT
    account,
    commodity,
    CAST(SUM(amount) AS TEXT) AS amount,
    CAST(SUM(quantity) AS TEXT) AS quantity
FROM postings
WHERE forecast = ?1
  AND (?2 = 0 OR date >= ?3)
  AND (?4 = 0 OR date <= ?5)
  AND (?6 = '' OR status = ?6)
  AND (?7 = 0 OR amount > 0)
  AND (?8 = '' OR account = ?8)
  AND (json_array_length(?9) = 0 OR commodity IN (SELECT value FROM json_each(?9)))
  AND (json_array_length(?10) = 0 OR EXISTS (
        SELECT 1 FROM json_each(?10)
        WHERE postings.account = value OR postings.account LIKE value || ':%'
    ))
  AND (json_array_length(?11) = 0 OR NOT EXISTS (
        SELECT 1 FROM json_each(?11)
        WHERE postings.account = value OR postings.account LIKE value || ':%'
    ))
  AND (json_array_length(?12) = 0 OR EXISTS (
        SELECT 1 FROM json_each(?12)
        WHERE postings.account LIKE value
    ))
  AND (json_array_length(?13) = 0 OR NOT EXISTS (
        SELECT 1 FROM json_each(?13)
        WHERE postings.account LIKE value
    ))
  AND (json_array_length(?14) = 0 OR account NOT IN (SELECT value FROM json_each(?14)))
GROUP BY account, commodity
ORDER BY account ASC, commodity ASC;

-- name: ListPostingTransactionHashes :many
SELECT DISTINCT transaction_id, transaction_hash
FROM postings
ORDER BY transaction_id ASC;

-- name: DeletePostingsByTransactionID :exec
DELETE FROM postings
WHERE transaction_id = ?1;

-- name: DeleteAllPostings :exec
DELETE FROM postings;

-- name: InsertPosting :exec
INSERT INTO postings (
    transaction_id,
    date,
    payee,
    account,
    commodity,
    quantity,
    amount,
    original_amount,
    status,
    tag_recurring,
    tag_period,
    transaction_begin_line,
    transaction_end_line,
    file_name,
    forecast,
    note,
    transaction_note,
    transaction_hash
) VALUES (
    sqlc.arg(transaction_id),
    sqlc.arg(date),
    sqlc.arg(payee),
    sqlc.arg(account),
    sqlc.arg(commodity),
    sqlc.arg(quantity),
    sqlc.arg(amount),
    sqlc.arg(original_amount),
    sqlc.arg(status),
    sqlc.arg(tag_recurring),
    sqlc.arg(tag_period),
    sqlc.arg(transaction_begin_line),
    sqlc.arg(transaction_end_line),
    sqlc.arg(file_name),
    sqlc.arg(forecast),
    sqlc.arg(note),
    sqlc.arg(transaction_note),
    sqlc.arg(transaction_hash)
);

-- name: ListPrices :many
SELECT id, date, commodity_type, commodity_id, commodity_name, quote_commodity, value, source
FROM prices
WHERE (?1 = '' OR commodity_name = ?1)
  AND (?2 = '' OR quote_commodity = ?2)
  AND (?3 = '' OR source = ?3)
  AND (?4 = 0 OR date >= ?5)
  AND (?6 = 0 OR date <= ?7)
ORDER BY date ASC, commodity_name ASC, quote_commodity ASC, source ASC;

-- name: ListLatestPrices :many
SELECT p1.id, p1.date, p1.commodity_type, p1.commodity_id, p1.commodity_name, p1.quote_commodity, p1.value, p1.source
FROM prices AS p1
WHERE (?1 = '' OR p1.commodity_name = ?1)
  AND (?2 = '' OR p1.quote_commodity = ?2)
  AND (?3 = '' OR p1.source = ?3)
  AND (?4 = 0 OR p1.date >= ?5)
  AND (?6 = 0 OR p1.date <= ?7)
  AND NOT EXISTS (
        SELECT 1
        FROM prices AS p2
        WHERE p2.commodity_name = p1.commodity_name
          AND (?1 = '' OR p2.commodity_name = ?1)
          AND (?2 = '' OR p2.quote_commodity = ?2)
          AND (?3 = '' OR p2.source = ?3)
          AND (?4 = 0 OR p2.date >= ?5)
          AND (?6 = 0 OR p2.date <= ?7)
          AND (
                p2.date > p1.date OR
                (p2.date = p1.date AND p2.quote_commodity < p1.quote_commodity) OR
                (p2.date = p1.date AND p2.quote_commodity = p1.quote_commodity AND p2.source < p1.source) OR
                (p2.date = p1.date AND p2.quote_commodity = p1.quote_commodity AND p2.source = p1.source AND p2.id > p1.id)
          )
    )
ORDER BY date ASC, commodity_name ASC, quote_commodity ASC, source ASC;

-- name: FindPriceByDateBaseQuote :one
SELECT id, date, commodity_type, commodity_id, commodity_name, quote_commodity, value, source
FROM prices
WHERE commodity_name = sqlc.arg(base_commodity)
  AND quote_commodity = sqlc.arg(quote_commodity)
  AND date <= sqlc.arg(at_or_before)
ORDER BY date DESC
LIMIT 1;

-- name: DeletePricesByType :exec
DELETE FROM prices
WHERE commodity_type = sqlc.arg(commodity_type);

-- name: UpsertPrice :exec
INSERT INTO prices (
    date,
    commodity_type,
    commodity_id,
    commodity_name,
    quote_commodity,
    value,
    source
) VALUES (
    sqlc.arg(date),
    sqlc.arg(commodity_type),
    sqlc.arg(commodity_id),
    sqlc.arg(commodity_name),
    sqlc.arg(quote_commodity),
    sqlc.arg(value),
    sqlc.arg(source)
)
ON CONFLICT (commodity_type, date, commodity_name, quote_commodity) DO UPDATE SET
    commodity_id = excluded.commodity_id,
    value = excluded.value,
    source = excluded.source;

-- name: ListPortfoliosByParent :many
SELECT id, commodity_type, parent_commodity_id, security_id, security_name, security_type, security_rating, security_industry, percentage
FROM portfolios
WHERE parent_commodity_id = sqlc.arg(parent_commodity_id)
ORDER BY security_name ASC, security_id ASC;

-- name: ListPortfolioParentCommodityIDs :many
SELECT DISTINCT parent_commodity_id
FROM portfolios
ORDER BY parent_commodity_id ASC;

-- name: DeletePortfoliosByTypeAndParent :exec
DELETE FROM portfolios
WHERE commodity_type = sqlc.arg(commodity_type)
  AND parent_commodity_id = sqlc.arg(parent_commodity_id);

-- name: InsertPortfolio :exec
INSERT INTO portfolios (
    commodity_type,
    parent_commodity_id,
    security_id,
    security_name,
    security_type,
    security_rating,
    security_industry,
    percentage
) VALUES (
    sqlc.arg(commodity_type),
    sqlc.arg(parent_commodity_id),
    sqlc.arg(security_id),
    sqlc.arg(security_name),
    sqlc.arg(security_type),
    sqlc.arg(security_rating),
    sqlc.arg(security_industry),
    sqlc.arg(percentage)
);
