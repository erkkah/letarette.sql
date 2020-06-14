-- Index update request
--
-- bound parameters:
--  afterDocument, fromTimeNanos, documentLimit
-- returns:
--  id, updatedNanos
--

with
documentState as (
    select coalesce(max(id), 0) as afterID
    from docs
    where id = :afterDocument
    and updated = :fromTimeNanos
)
select
	id, updated as updatedNanos
from
	docs join documentState
where
    (updated = :fromTimeNanos and id > afterID)
	or updated > :fromTimeNanos
order by
	updated, id
limit :documentLimit
