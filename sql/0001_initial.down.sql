-- Copyright 2020 Tom Peters
-- 
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
-- 
--    http://www.apache.org/licenses/LICENSE-2.0
-- 
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.


BEGIN;

DROP TABLE pool_squares_logs;
DROP TABLE pool_squares;
DROP TABLE pools_users;
DROP TABLE grid_settings;
DROP TABLE grids;
DROP TABLE pools;
DROP TABLE users;
DROP TABLE tokens;
DROP TYPE square_states;
DROP TYPE states;
DROP TYPE stores;

COMMIT;
