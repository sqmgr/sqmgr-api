-- Index for time-based filtering on pools
CREATE INDEX pools_created_idx ON pools (created);

-- Composite index for user queries (store + time filter)
CREATE INDEX users_store_created_idx ON users (store, created);

-- Composite index for archived pools with time filter
CREATE INDEX pools_archived_created_idx ON pools (archived, created);

-- For GetAllPools correlated subqueries:
CREATE INDEX pool_squares_pool_id_state_idx ON pool_squares (pool_id, state);
CREATE INDEX grids_pool_id_state_active_idx ON grids (pool_id) WHERE state = 'active';
