/*
Copyright (C) 2019 Tom Peters

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package model

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"
	"unicode/utf8"

	"github.com/lib/pq"
)

// TeamNameMaxLength is the maximum length of the team names
const TeamNameMaxLength = 75

const (
	defaultHomeTeamName = "Home Team"
	defaultAwayTeamName = "Away Team"
)

// ErrNumbersAlreadyDrawn happens when SelectRandomNumbers() is called multiple times
var ErrNumbersAlreadyDrawn = errors.New("error: numbers have already been drawn")

// ErrNumbersAreInvalid happens when the user submits manual numbers and they are invalid
var ErrNumbersAreInvalid = errors.New("error: numbers supplied are invalid")

// ErrLastGrid happens when the user tries to delete the last remaining grid
var ErrLastGrid = errors.New("error: you cannot delete the last grid")

// ErrGridLimit happens when a user tries to create more grids in a pool than allowed
var ErrGridLimit = fmt.Errorf("you cannot create more than %d grids per pool", MaxGridsPerPool)

// Grid represents a single grid from a pool. A pool may contain more than one grid.
type Grid struct {
	model *Model

	id           int64
	poolID       int64
	ord          int
	label        *string
	homeTeamName *string
	homeNumbers  []int
	awayTeamName *string
	awayNumbers  []int
	manualDraw   bool
	eventDate    time.Time
	rollover     bool
	state        State
	created      time.Time
	modified     time.Time
	bdlEventID   *int64
	payoutConfig *NumberSetConfig

	settings    *GridSettings
	annotations map[int]*GridAnnotation
	numberSets  map[NumberSetType]*GridNumberSet
	bdlEvent    *BDLEvent
}

// GridJSON represents grid metadata that can be sent to the front-end
type GridJSON struct {
	ID             int64                                `json:"id"`
	Name           string                               `json:"name"`
	Label          string                               `json:"label"`
	HomeTeamName   string                               `json:"homeTeamName"`
	HomeNumbers    []int                                `json:"homeNumbers"`
	AwayTeamName   string                               `json:"awayTeamName"`
	AwayNumbers    []int                                `json:"awayNumbers"`
	ManualDraw     bool                                 `json:"manualDraw"`
	EventDate      time.Time                            `json:"eventDate"`
	Rollover       bool                                 `json:"rollover"`
	State          State                                `json:"state"`
	Created        time.Time                            `json:"created"`
	Modified       time.Time                            `json:"modified"`
	Settings       *GridSettings                        `json:"settings"`
	Annotations    map[int]*GridAnnotation              `json:"annotations"`
	NumberSets     map[NumberSetType]*GridNumberSetJSON `json:"numberSets,omitempty"`
	BDLEventID     *int64                               `json:"bdlEventId,omitempty"`
	BDLEvent       *BDLEventJSON                        `json:"bdlEvent,omitempty"`
	WinningSquares map[NumberSetType]int                `json:"winningSquares,omitempty"`
	PayoutConfig   *NumberSetConfig                     `json:"payoutConfig,omitempty"`
}

// JSON will marshal the JSON using a custom marshaller
func (g *Grid) JSON() *GridJSON {
	json := &GridJSON{
		ID:           g.ID(),
		Name:         g.Name(),
		Label:        g.Label(),
		HomeTeamName: g.HomeTeamName(),
		HomeNumbers:  g.HomeNumbers(),
		AwayTeamName: g.AwayTeamName(),
		AwayNumbers:  g.AwayNumbers(),
		ManualDraw:   g.manualDraw,
		EventDate:    g.EventDate(),
		Rollover:     g.Rollover(),
		State:        g.State(),
		Created:      g.Created(),
		Modified:     g.modified,
		Settings:     g.settings,
		Annotations:  g.annotations,
		BDLEventID:   g.bdlEventID,
		PayoutConfig: g.payoutConfig,
	}

	if len(g.numberSets) > 0 {
		json.NumberSets = make(map[NumberSetType]*GridNumberSetJSON)
		for k, v := range g.numberSets {
			json.NumberSets[k] = v.JSON()
		}
	}

	if g.bdlEvent != nil {
		json.BDLEvent = g.bdlEvent.JSON()
	}

	return json
}

// JSONWithWinningSquares returns JSON with winning squares calculated
func (g *Grid) JSONWithWinningSquares(poolConfig NumberSetConfig, gridType GridType) *GridJSON {
	json := g.JSON()

	if g.bdlEvent != nil {
		// Use grid's payout config if set, otherwise fall back to pool's number set config
		config := poolConfig
		if g.payoutConfig != nil {
			config = *g.payoutConfig
		}
		result := g.GetGridWinningSquares(g.bdlEvent, config, gridType)
		if len(result.Squares) > 0 {
			json.WinningSquares = result.Squares
		}
	}

	return json
}

// State is a getter for the state
func (g *Grid) State() State {
	return g.state
}

// SetState will set the state
func (g *Grid) SetState(state State) {
	g.state = state
}

// SetEventDate is a setter for the event date
func (g *Grid) SetEventDate(eventDate time.Time) {
	g.eventDate = eventDate
}

// AwayTeamName is a getter for the away team name
func (g *Grid) AwayTeamName() string {
	if g.awayTeamName == nil {
		return defaultAwayTeamName
	}

	return *g.awayTeamName
}

// Rollover is a getter
func (g *Grid) Rollover() bool {
	return g.rollover
}

// SetRollover is the setter
func (g *Grid) SetRollover(rollover bool) {
	g.rollover = rollover
}

// SetAwayTeamName is the setter for the away team name
func (g *Grid) SetAwayTeamName(awayTeamName string) {
	if awayTeamName == "" {
		g.awayTeamName = nil
		return
	}

	if utf8.RuneCountInString(awayTeamName) > TeamNameMaxLength {
		awayTeamName = string([]rune(awayTeamName)[0:TeamNameMaxLength])
	}

	g.awayTeamName = &awayTeamName
}

// HomeTeamName is a getter for the home team name
func (g *Grid) HomeTeamName() string {
	if g.homeTeamName == nil {
		return defaultHomeTeamName
	}

	return *g.homeTeamName
}

// SetHomeTeamName is a setter for the home team name
func (g *Grid) SetHomeTeamName(homeTeamName string) {
	if homeTeamName == "" {
		g.homeTeamName = nil
		return
	}

	if utf8.RuneCountInString(homeTeamName) > TeamNameMaxLength {
		homeTeamName = string([]rune(homeTeamName)[0:TeamNameMaxLength])
	}

	g.homeTeamName = &homeTeamName
}

// ID returns the grid ID
func (g *Grid) ID() int64 {
	return g.id
}

// Created returns the created timestamp
func (g *Grid) Created() time.Time {
	return g.created
}

// EventDate returns the date of the event
func (g *Grid) EventDate() time.Time {
	return g.eventDate
}

// AwayNumbers returns the numbers to be used for the away team
func (g *Grid) AwayNumbers() []int {
	return g.awayNumbers
}

// HomeNumbers returns the numbers to be used for the home team
func (g *Grid) HomeNumbers() []int {
	return g.homeNumbers
}

// BDLEventID returns the BDL event ID if linked
func (g *Grid) BDLEventID() *int64 {
	return g.bdlEventID
}

// SetBDLEventID sets the BDL event ID
func (g *Grid) SetBDLEventID(id *int64) {
	g.bdlEventID = id
}

// BDLEvent returns the loaded BDL event
func (g *Grid) BDLEvent() *BDLEvent {
	return g.bdlEvent
}

// SetBDLEvent sets the BDL event
func (g *Grid) SetBDLEvent(event *BDLEvent) {
	g.bdlEvent = event
	if event != nil {
		g.bdlEventID = &event.ID
	} else {
		g.bdlEventID = nil
	}
}

// PayoutConfig returns the grid's payout configuration
func (g *Grid) PayoutConfig() *NumberSetConfig {
	return g.payoutConfig
}

// SetPayoutConfig sets the payout configuration for the grid
func (g *Grid) SetPayoutConfig(config *NumberSetConfig) {
	g.payoutConfig = config
}

// Label returns the label of the grid.
func (g *Grid) Label() string {
	if g.label == nil {
		return ""
	}

	return *g.label
}

// SetLabel will set the label
func (g *Grid) SetLabel(label string) {
	if label == "" {
		g.label = nil
		return
	}

	g.label = &label
}

// Save will save the grid. It will also save any dependent objects
func (g *Grid) Save(ctx context.Context) error {
	tx, err := g.model.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if g.id == 0 {
		const query = `
SELECT ` + gridColumns + `
FROM
	new_grid($1, $2)
`
		row := tx.QueryRowContext(ctx, query, g.poolID, MaxGridsPerPool)
		newGrid, err := g.model.gridByRow(row.Scan)
		if err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				return fmt.Errorf("error found: %#v. Another error found when trying to rollback: %#v", err, err2)
			}

			if err.Error() == "pq: limit reached" {
				return ErrGridLimit
			}

			return err
		}

		g.id = newGrid.id
		g.state = newGrid.state
		g.created = newGrid.created
		g.ord = newGrid.ord
		g.poolID = newGrid.poolID
		if g.settings != nil {
			g.settings.gridID = g.id
		}
	}

	if g.settings != nil {
		if err := g.settings.Save(ctx, tx); err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				return fmt.Errorf("error found: %#v. Another error found when trying to rollback: %#v", err, err2)
			}

			return err
		}
	}

	var eventDate *time.Time
	if !g.eventDate.IsZero() {
		eventDate = &g.eventDate
	}

	const query = `
		UPDATE grids
		SET ord = $1,
		    home_team_name = $2,
			home_numbers = $3,
		    away_team_name = $4,
			away_numbers = $5,
		    manual_draw = $6,
			event_date = $7,
		    rollover = $8,
		    state = $9,
		    label = $10,
		    sports_event_id = $11,
		    payout_config = $12,
			modified = (now() at time zone 'utc')
		WHERE id = $13
	`

	if _, err := tx.ExecContext(ctx, query, g.ord, g.homeTeamName, pq.Array(g.homeNumbers), g.awayTeamName, pq.Array(g.awayNumbers), g.manualDraw, eventDate, g.rollover, g.state, g.label, g.bdlEventID, g.payoutConfig, g.id); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			return fmt.Errorf("error found: %#v. Another error found when trying to rollback: %#v", err, err2)
		}

		return err
	}

	return tx.Commit()
}

// Settings will return the settings
func (g *Grid) Settings() *GridSettings {
	return g.settings
}

// Name returns the name of the grid
func (g *Grid) Name() string {
	vs := fmt.Sprintf("%s vs. %s", g.AwayTeamName(), g.HomeTeamName())
	if g.label == nil {
		return vs
	}

	return fmt.Sprintf("%s: %s", *g.label, vs)
}

// SetManualNumbers will set numbers manually (user input)
func (g *Grid) SetManualNumbers(homeTeamNumbers, awayTeamNumbers []int) error {
	if g.homeNumbers != nil || g.awayNumbers != nil {
		return ErrNumbersAlreadyDrawn
	}

	if !numbersAreValid(homeTeamNumbers) || !numbersAreValid(awayTeamNumbers) {
		return ErrNumbersAreInvalid
	}

	g.manualDraw = true
	g.homeNumbers = homeTeamNumbers
	g.awayNumbers = awayTeamNumbers

	return nil
}

func numbersAreValid(nums []int) bool {
	if len(nums) != 10 {
		return false
	}

	check := make([]int, 10)
	for _, n := range nums {
		if n >= 0 && n <= 9 {
			check[n]++
			if check[n] > 1 {
				return false
			}
		}
	}

	return true
}

// SelectRandomNumbers will select random numbers for the home and away team
func (g *Grid) SelectRandomNumbers() error {
	if g.homeNumbers != nil || g.awayNumbers != nil {
		return ErrNumbersAlreadyDrawn
	}

	hNums, err := randomNumbers()
	if err != nil {
		return err
	}

	g.homeNumbers = hNums

	aNums, err := randomNumbers()
	if err != nil {
		return err
	}

	g.awayNumbers = aNums

	return nil
}

// Delete the grid. By delete, we mean set the row to 'deleted'
func (g *Grid) Delete(ctx context.Context) error {
	const query = "SELECT * FROM delete_grid($1)"
	row := g.model.DB.QueryRowContext(ctx, query, g.id)
	var ok bool
	if err := row.Scan(&ok); err != nil {
		return err
	}

	if !ok {
		return ErrLastGrid
	}

	return nil
}

// LoadSettings will load the settings
func (g *Grid) LoadSettings(ctx context.Context) error {
	row := g.model.DB.QueryRowContext(ctx, `
		SELECT grid_id,
			   home_team_color_1, home_team_color_2,
			   away_team_color_1, away_team_color_2,
			   notes, branding_image_url, branding_image_alt, modified
		FROM grid_settings
		WHERE grid_id = $1
	`, g.id)

	if g.settings == nil {
		g.settings = &GridSettings{}
	}

	return row.Scan(
		&g.settings.gridID,
		&g.settings.homeTeamColor1,
		&g.settings.homeTeamColor2,
		&g.settings.awayTeamColor1,
		&g.settings.awayTeamColor2,
		&g.settings.notes,
		&g.settings.brandingImageURL,
		&g.settings.brandingImageAlt,
		&g.settings.modified,
	)
}

// LoadAnnotations will load the annotations for the grid
func (g *Grid) LoadAnnotations(ctx context.Context) error {
	annotations, err := g.Annotations(ctx)
	if err != nil {
		return err
	}

	g.annotations = annotations
	return nil
}

// LoadNumberSets loads all number sets for the grid
func (g *Grid) LoadNumberSets(ctx context.Context) error {
	numberSets, err := g.model.GridNumberSetsByGridID(ctx, g.id)
	if err != nil {
		return err
	}

	g.numberSets = numberSets
	return nil
}

// NumberSets returns the loaded number sets
func (g *Grid) NumberSets() map[NumberSetType]*GridNumberSet {
	return g.numberSets
}

// LoadBDLEvent loads the BDL event if one is linked
func (g *Grid) LoadBDLEvent(ctx context.Context) error {
	if g.bdlEventID == nil {
		return nil
	}

	event, err := g.model.BDLEventByIDWithTeams(ctx, *g.bdlEventID)
	if err != nil {
		return err
	}

	g.bdlEvent = event
	return nil
}

// NumbersAreDrawn checks if ALL required sets have numbers for the given config
func (g *Grid) NumbersAreDrawn(config NumberSetConfig) bool {
	setTypes := GetSetTypes(config)
	if setTypes == nil {
		return false
	}

	// For "standard" config, check legacy homeNumbers/awayNumbers
	if config == NumberSetConfigStandard {
		return g.homeNumbers != nil && g.awayNumbers != nil
	}

	// For multi-set configs, check each required set
	if g.numberSets == nil {
		return false
	}

	for _, setType := range setTypes {
		ns, ok := g.numberSets[setType]
		if !ok || !ns.HasNumbers() {
			return false
		}
	}

	return true
}

// DrawAllNumbersRandom atomically draws random numbers for all required sets
func (g *Grid) DrawAllNumbersRandom(ctx context.Context, config NumberSetConfig) error {
	setTypes := GetSetTypes(config)
	if setTypes == nil {
		return fmt.Errorf("invalid number set config: %s", config)
	}

	// For "standard" config, use legacy behavior
	if config == NumberSetConfigStandard {
		return g.SelectRandomNumbers()
	}

	tx, err := g.model.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	newSets := make(map[NumberSetType]*GridNumberSet)
	for _, setType := range setTypes {
		ns := g.model.NewGridNumberSet(g.id, setType)
		if err := ns.SelectRandomNumbers(); err != nil {
			return fmt.Errorf("selecting random numbers for %s: %w", setType, err)
		}
		if err := ns.Save(ctx, tx); err != nil {
			return fmt.Errorf("saving number set %s: %w", setType, err)
		}
		newSets[setType] = ns
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	g.numberSets = newSets
	return nil
}

// NumberSetInput represents input for a single number set
type NumberSetInput struct {
	HomeNumbers []int `json:"homeTeamNumbers"`
	AwayNumbers []int `json:"awayTeamNumbers"`
}

// DrawAllNumbersManual atomically sets manual numbers for all required sets
func (g *Grid) DrawAllNumbersManual(ctx context.Context, config NumberSetConfig, numberSets map[NumberSetType]NumberSetInput) error {
	setTypes := GetSetTypes(config)
	if setTypes == nil {
		return fmt.Errorf("invalid number set config: %s", config)
	}

	// For "standard" config, use legacy behavior with "all" set
	if config == NumberSetConfigStandard {
		input, ok := numberSets[NumberSetTypeAll]
		if !ok {
			return fmt.Errorf("missing 'all' number set for 'standard' config")
		}
		return g.SetManualNumbers(input.HomeNumbers, input.AwayNumbers)
	}

	// Validate all required sets are provided
	for _, setType := range setTypes {
		if _, ok := numberSets[setType]; !ok {
			return fmt.Errorf("missing number set for %s", setType)
		}
	}

	tx, err := g.model.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	newSets := make(map[NumberSetType]*GridNumberSet)
	for _, setType := range setTypes {
		input := numberSets[setType]
		ns := g.model.NewGridNumberSet(g.id, setType)
		if err := ns.SetNumbers(input.HomeNumbers, input.AwayNumbers); err != nil {
			return fmt.Errorf("setting numbers for %s: %w", setType, err)
		}
		if err := ns.Save(ctx, tx); err != nil {
			return fmt.Errorf("saving number set %s: %w", setType, err)
		}
		newSets[setType] = ns
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	g.numberSets = newSets
	return nil
}

func randomNumbers() ([]int, error) {
	nums := make([]int, 10)
	for i := range nums {
		nums[i] = i
	}

	for i := len(nums) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, err
		}
		j := int(jBig.Int64())

		nums[i], nums[j] = nums[j], nums[i]
	}

	return nums, nil
}

func (m *Model) gridByRow(scan scanFunc) (*Grid, error) {
	grid := &Grid{model: m}

	var homeNumbers, awayNumbers []sql.NullInt64
	var eventDate *time.Time
	var payoutConfig *string

	if err := scan(&grid.id, &grid.poolID, &grid.ord, &grid.label, &grid.homeTeamName, pq.Array(&homeNumbers), &grid.awayTeamName, pq.Array(&awayNumbers), &eventDate, &grid.rollover, &grid.state, &grid.created, &grid.modified, &grid.manualDraw, &grid.bdlEventID, &payoutConfig); err != nil {
		return nil, err
	}

	if payoutConfig != nil {
		config := NumberSetConfig(*payoutConfig)
		grid.payoutConfig = &config
	}

	if homeNumbers != nil {
		grid.homeNumbers = make([]int, len(homeNumbers))
		for i, val := range homeNumbers {
			grid.homeNumbers[i] = int(val.Int64)
		}
	}

	if awayNumbers != nil {
		grid.awayNumbers = make([]int, len(awayNumbers))
		for i, val := range awayNumbers {
			grid.awayNumbers[i] = int(val.Int64)
		}
	}

	if eventDate != nil {
		grid.eventDate = *eventDate
	}

	grid.modified = grid.modified.In(locationNewYork)
	grid.created = grid.created.In(locationNewYork)

	return grid, nil
}

const gridColumns = `
	id,
	pool_id,
	ord,
	label,
	home_team_name,
	home_numbers,
	away_team_name,
	away_numbers,
	event_date,
	rollover,
	state,
	created,
	modified,
	manual_draw,
	sports_event_id,
	payout_config`
