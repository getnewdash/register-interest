# Database schema for this simple register-interest application

This creates a blank database ready for the applications to use

To load this into a fresh PostgreSQL database run these commands
from a postgres superuser:

    $ createuser -d newdash
    $ createdb -O newdash newdash_interest
    $ psql -U newdash newdash_interest < schema.sql

It should finish with no errors.

Note - This schema is created using:

    $ pg_dump -Os -U newdash newdash_interest > schema.sql

