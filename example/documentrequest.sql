-- Document update request
--
-- bound arguments:
--  wantedIDs to be used in an "in" statement
-- returns:
--  id, updatedNanos, title, txt, alive

select
    id, updated as updatedNanos, title, txt, 1 as alive
from
    docs
where
    id in (:wantedIDs)
