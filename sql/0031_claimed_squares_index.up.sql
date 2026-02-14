-- Partial index for counting claimed squares with optional time filter on modified
CREATE INDEX pool_squares_claimed_modified_idx ON pool_squares (modified) WHERE state != 'unclaimed';
