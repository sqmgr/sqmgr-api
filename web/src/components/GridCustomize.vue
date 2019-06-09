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

// TODO - handle max lengths and notification

<template>
    <section class="grid-customize">
        <template v-if="saved">
            <p>Your changes have been saved</p>

            <div class="buttons">
                <button type="button" @click.prevent="ModalController.hide()">Dismiss</button>
            </div>
        </template>
        <template v-else-if="grid">
            <form @submit.prevent="submit">
                <template v-if="errors">
                    <template v-for="(errList, errKey) in errors">
                        <div class="errors">
                            <h4>Error</h4>

                            <p>Please correct the following errors:</p>

                            <ul>
                                <li>{{errKey}}
                                    <ul>
                                        <li v-for="err in errList">{{err}}</li>
                                    </ul>
                                </li>
                            </ul>
                        </div>
                    </template>
                </template>
                <fieldset>
                    <legend>General Settings</legend>

                    <div class="field">
                        <label for="event-date">Event Date</label>
                        <input type="date" id="event-date" name="event-date" v-model="form.eventDate">
                    </div>

                    <div class="field">
                        <label for="notes" class="optional">Notes</label>
                        <textarea id="notes" :maxlength="notesMaxLength" name="notes" placeholder="Notes" v-model="form.notes"></textarea>
                    </div>
                </fieldset>
                <fieldset>
                    <legend>Styling</legend>

                    <GridCustomizeTeam name="Away Team" v-model="form.awayTeam"/>
                    <GridCustomizeTeam name="Home Team" v-model="form.homeTeam"/>
                </fieldset>

                <div class="buttons">
                    <button type="button" class="secondary" @click.prevent="didClickCancel">Cancel</button>
                    <button type="submit" name="submit">Save</button>
                </div>
            </form>
        </template>
        <template v-else>
            <div class="loading-indicator"><span></span></div>
        </template>
    </section>
</template>

<script>
    import GridCustomizeTeam from './GridCustomizeTeam.vue'
    import api from '../models/api'
    import ModalController from '@/controllers/ModalController'

    export default {
        name: "GridCustomize",
        components: {GridCustomizeTeam},
        props: {
            gridID: {
                type: Number,
                required: true,
            },
        },
        beforeMount() {
            api.getGrid(this.gridID)
                .then(grid => this.grid = grid)
                .catch(err => ModalController.showError(err))
        },
        watch: {
            grid(newValue) {
                const date = newValue.eventDate.substr(0,10)
                this.form.eventDate = date === '0001-01-01' ? '' : date
                this.form.notes = newValue.settings.notes
                this.form.awayTeam.name = newValue.awayTeamName
                this.form.awayTeam.color1 = newValue.settings.awayTeamColor1
                this.form.awayTeam.color2 = newValue.settings.awayTeamColor2
                this.form.homeTeam.name = newValue.homeTeamName
                this.form.homeTeam.color1 = newValue.settings.homeTeamColor1
                this.form.homeTeam.color2 = newValue.settings.homeTeamColor2
            }
        },
        data() {
            return {
                ModalController,
                errors: null,
                saved: null,
                grid: null,
                teamNameMaxLength: 50, // TODO
                notesMaxLength: 200, // TODO
                form: {
                    eventDate: '0000-00-00',
                    notes: '',
                    awayTeam: {
                        name: '',
                        color1: '',
                        color2: '',
                    },
                    homeTeam: {
                        name: '',
                        color1: '',
                        color2: '',
                    }
                }
            }
        },
        methods: {
            submit() {
                this.errors = null

                api.saveGrid(this.gridID, {
                    eventDate: this.form.eventDate,
                    notes: this.form.notes,
                    homeTeamName: this.form.homeTeam.name,
                    homeTeamColor1: this.form.homeTeam.color1,
                    homeTeamColor2: this.form.homeTeam.color2,
                    awayTeamName: this.form.awayTeam.name,
                    awayTeamColor1: this.form.awayTeam.color1,
                    awayTeamColor2: this.form.awayTeam.color2,
                })
                    .then(grid => { this.$emit('saved', grid); this.saved = true })
                    .catch(err => {
                        if (err.response && err.response.data && err.response.data.result) {
                            this.errors = err.response.data.result
                        }

                        ModalController.showError(err)
                    })
            },
            didClickCancel() {
                this.$emit('canceled')
                ModalController.hide()
            }
        },
    }
</script>

<style scoped>
    section.grid-customize {
        position: relative;
        width: 70vw;

        div.loading-indicator {
            position: static;
            bottom: var(--minimal-spacing);
            right: var(--minimal-spacing);
        }
    }
</style>