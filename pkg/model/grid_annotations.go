/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"context"
	"database/sql"
	"time"
)

// GridAnnotation represents an annotation on a grid square
type GridAnnotation struct {
	model      *Model
	ID         int64     `json:"id"`
	GridID     int64     `json:"grid_id"`
	SquareID   int       `json:"square_id"`
	Annotation string    `json:"annotation"`
	Created    time.Time `json:"created"`
	Modified   time.Time `json:"modified"`
}

const gridAnnotationColumns = `id, grid_id, square_id, annotation, created, modified`

// DeleteAnnotationBySquareID will delete the annotation
func (g *Grid) DeleteAnnotationBySquareID(ctx context.Context, squareID int) error {
	_, err := g.model.DB.ExecContext(ctx, "DELETE FROM grid_annotations WHERE grid_id = $1 AND square_id = $2", g.ID(), squareID)
	return err
}

// AnnotationBySquareID will always return a GridAnnotation object, unless an error occurs.
// In the event an existing annotation cannot be found, a new object is created and returned
func (g *Grid) AnnotationBySquareID(ctx context.Context, squareID int) (*GridAnnotation, error) {
	const query = `SELECT ` + gridAnnotationColumns + ` FROM grid_annotations WHERE grid_id = $1 AND square_id = $2`
	row := g.model.DB.QueryRowContext(ctx, query, g.id, squareID)
	ga, err := g.model.gridAnnotationByRow(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return &GridAnnotation{
				model:    g.model,
				GridID:   g.id,
				SquareID: squareID,
			}, nil
		}

		return nil, err
	}

	return ga, nil
}

// Save will save the data to the database
func (a *GridAnnotation) Save(ctx context.Context) error {
	// insert
	if a.ID == 0 {
		const query = `
INSERT INTO grid_annotations
	(grid_id, square_id, annotation)
VALUES
	($1, $2, $3)
RETURNING ` + gridAnnotationColumns

		model := a.model
		row := model.DB.QueryRowContext(ctx, query, a.GridID, a.SquareID, a.Annotation)
		a2, err := model.gridAnnotationByRow(row.Scan)
		if err != nil {
			return err
		}

		*a = *a2
		a.model = model
		return nil
	}

	const query = `
UPDATE
	grid_annotations
SET
	annotation = $1,
	modified = (NOW() AT TIME ZONE 'UTC')
WHERE
	id = $2
`

	_, err := a.model.DB.ExecContext(ctx, a.Annotation, a.ID)
	return err
}

// Annotations returns a map of square IDs to GridAnnotation objects, or an error
func (g *Grid) Annotations(ctx context.Context) (map[int]*GridAnnotation, error) {
	const query = `
SELECT ` + gridAnnotationColumns + `
FROM
	grid_annotations
WHERE
	grid_id = $1	
`

	rows, err := g.model.DB.QueryContext(ctx, query, g.ID())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	annotations := make(map[int]*GridAnnotation)
	for rows.Next() {
		a, err := g.model.gridAnnotationByRow(rows.Scan)
		if err != nil {
			return nil, err
		}

		annotations[a.SquareID] = a
	}

	return annotations, nil
}

func (m *Model) gridAnnotationByRow(scan scanFunc) (*GridAnnotation, error) {
	ga := GridAnnotation{}
	if err := scan(&ga.ID, &ga.GridID, &ga.SquareID, &ga.Annotation, &ga.Created, &ga.Modified); err != nil {
		return nil, err
	}

	ga.model = m
	return &ga, nil
}
