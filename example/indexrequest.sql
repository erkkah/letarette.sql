-- Index update request
--
-- bound arguments:
--  afterDocument, fromTimeNanos, documentLimit
-- returns:
--  id, updated

with
params as (
    select ? as afterDocument, ? as fromTimeNanos
),
documentState as (
    select coalesce(max(id), 0) as afterID
    from docs cross join params
    where id = afterDocument
    and updated = fromTimeNanos
)
select
	id, updated
from
	docs join documentState join params
where
    (updated = fromTimeNanos and id > afterID)
	or updated > fromTimeNanos
order by
	updated, id
limit ?
