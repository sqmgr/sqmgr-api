# General Requirements

* Always write unit tests for any new or modified code
* Always run go fmt at the end
* Ensure all unit tests pass

# Database Migrations

* Migrations live in `sql/` and follow the pattern `NNNN_description.{up,down}.sql`
* Before creating a new migration, glob `sql/0*.sql` to find the current highest sequence number and use the next one
* Never hardcode or guess the migration number from a plan — always check the filesystem
