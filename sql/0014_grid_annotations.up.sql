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

CREATE TABLE grid_annotations (
    id bigserial primary key,
    grid_id bigint not null references grids (id),
    square_id int not null,
    annotation text not null,
    created timestamp not null default (now() at time zone 'utc'),
    modified timestamp not null default (now() at time zone 'utc'),
    icon smallint NOT NULL DEFAULT 0,
    UNIQUE (grid_id, square_id)
);

COMMIT;
