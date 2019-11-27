-- Document update request
--
-- bound arguments:
--  wantedIDs to be used in an "in" statement
-- returns:
--  id, updated, title, txt, alive

select
    id, updated, title, txt, 1 as alive
from
    docs
where
    id in (?)
