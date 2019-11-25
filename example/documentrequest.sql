-- Document update request
--
-- bound arguments:
--  wantedIDs to be used in an "in" statement
-- returns:
--  id, updated, title, text, alive

select
    id, updated, title, txt, 1
from
    docs
where
    id in (?)
