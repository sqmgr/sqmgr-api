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

<template>
    <div>
        <h3>{{ grid.name }}</h3>

        <p class="pool-name">Squares Pool - {{ pool.name }}</p>

        <template v-if="isAdmin">
            <nav class="admin-menu">
                <h4>Admin Menu</h4>

                <button type="button" @click.prevent="customizeWasClicked">Customize</button>

                <template v-if="!numbersAreDrawn">
                    <button type="button" @click.prevent="drawNumbersWasClicked">Draw Numbers</button>
                </template>
            </nav>
        </template>

        <template v-if="grid.settings.notes">
            <div class="notes">{{ grid.settings.notes }}</div>
        </template>

        <div class="grid-metadata">
            <table>
                <tbody>
                <tr>
                    <td>ID</td>
                    <td>{{ pool.token }}</td>
                </tr>
                <tr>
                    <td>Pool Name</td>
                    <td><a :href="`/pool/${pool.token}`">{{ pool.name }}</a></td>
                </tr>
                <tr>
                    <td>Event</td>
                    <td>{{ grid.name }}</td>
                </tr>
                <tr>
                    <td>Date of Game</td>
                    <td>{{ eventDate }}</td>
                </tr>
                <tr>
                    <td>Type</td>
                    <td>{{ pool.gridType }}</td>
                </tr>
                <tr>
                    <td>Squares Locked</td>
                    <td v-if="isLocked"><i class="fas fa-lock"></i> Locked</td>
                    <td v-else><i class="fas fa-lock-open"></i> Unlocked</td>
                </tr>
                </tbody>
            </table>
        </div>

        <div :class="`squares ${pool.gridType}`">
            <div class="spacer">&nbsp;</div>

            <div class="team home-team" ref="home-team"><span>{{ grid.homeTeamName }}</span></div>
            <div v-for="n in 10" :class="`score home-score home-score-${n-1}`">{{score('home', n-1)}}</div>

            <div class="team away-team" ref="away-team"><span>{{ grid.awayTeamName }}</span></div>
            <div v-for="n in 10" :class="`score away-score away-score-${n-1}`">{{score('away', n-1)}}</div>

            <template v-for="n in numSquares">
                <Square :opaque-user-id="opaqueUserID" :sq-id="n" :square-data="squares[n] || {}"></Square>
            </template>
        </div>

        <template v-if="isAdmin">
            <Logs :show-add-note="false" :logs="logs"/>
        </template>

        <Modal/>
    </div>
</template>

<script>
    import Square from './Square.vue'
    import Logs from './Logs.vue'
    import api from '../models/api.js'
    import EventBus from '../models/EventBus'
    import GridCustomize from './GridCustomize.vue'
    import Common from '../common'

    import Modal from '@/components/Modal'
    import ModalController from '@/controllers/ModalController'

    api.token = SqMGR.gridConfig.pool.token

    const Config = {
        Squares: {
            std100: 100,
            std25: 25,
        }
    }

    export default {
        name: "Grid",
        components: {
            Square,
            Logs,
            Modal,
        },
        data() {
            return {
                ...SqMGR.gridConfig,
                numSquares: Config.Squares[SqMGR.gridConfig.pool.gridType],
                squares: {},
                logs: [],
            }
        },
        computed: {
            eventDate() {
                if (new Date(this.grid.eventDate).getFullYear() === 0) {
                    return "Not specified"
                }

                return Common.NewDateWithoutTimezone(this.grid.eventDate).toLocaleDateString("default", Common.DateOptions)
            },
            locks() {
                const locks = new Date(this.pool.locks)
                return locks.getFullYear() > 1 ? locks : null
            },
            locksFormatted() {
                return this.locks ? this.locks.toLocaleDateString('default', Common.DateTimeOptions) : null
            },
            isLocked() {
                return this.locks && this.locks.getTime() < new Date().getTime()
            },
            numbersAreDrawn() {
                return this.grid.homeNumbers || this.grid.awayNumbers
            }
        },
        beforeMount() {
            this.loadData()
            EventBus.$on('data-updated', () => this.loadData())
            EventBus.$on('grid-updated', () => this.loadGrid())
        },
        mounted() {
            this.updateTeamColors()
        },
        updated() {
            this.updateTeamColors()
        },
        methods: {
            customizeWasClicked() {
                ModalController.show('Customize Grid', GridCustomize, {
                    gridID: this.grid.id,
                }, {
                    saved() {
                        ModalController.hide()
                        ModalController.showPrompt('Changes Saved', 'Changes saved successfully.', {
                            dismissButton: 'OK',
                        })
                    }
                })
            },
            drawNumbersWasClicked() {
                let allClaimed = true
                for (const key of Object.keys(this.squares)) {
                    const square = this.squares[key]
                    if (square.state === 'unclaimed') {
                        allClaimed = false
                        break
                    }
                }

                const description = 'Do you want to draw the numbers for this game? This action cannot be undone.'
                const warning = !allClaimed && 'Not all squares have been claimed yet'

                ModalController.showPrompt('Draw the Numbers', description, {
                    warning,
                    actionButton: 'Draw',
                    action: () => {
                        api.drawNumbers(this.grid.id)
                            .then(grid => {
                                this.grid = grid
                                ModalController.hide()
                            })
                            .catch(err => ModalController.showError(err))
                    },
                })
            },
            loadData() {
                api.getSquares()
                    .then(res => {
                        this.squares = res
                    })

                if (this.isAdmin) {
                    api.getLogs()
                        .then(res => {
                            this.logs = res
                        })
                }
            },
            loadGrid() {
                api.getGrid(this.grid.id)
                    .then(grid => this.grid = grid)
                    .catch(err => ModalController.showError(err))
            },
            score(team, index) {
                const numbers = this.grid[`${team}Numbers`]
                if (numbers === null) {
                    return ''
                }

                return numbers[index]
            },
            updateTeamColors() {
                const ht = this.$refs["home-team"]
                ht.style.setProperty('--team-primary', this.grid.settings.homeTeamColor1)
                ht.style.setProperty('--team-secondary', this.grid.settings.homeTeamColor2)

                const at = this.$refs["away-team"]
                at.style.setProperty('--team-primary', this.grid.settings.awayTeamColor1)
                at.style.setProperty('--team-secondary', this.grid.settings.awayTeamColor2)
            },
        }
    }
</script>

<style lang="scss" scoped>
    .grid-metadata {
        margin-bottom: var(--spacing);

        table {
            td:first-child {
                white-space: nowrap;
            }
        }
    }

    nav.admin-menu {
        border: 1px solid var(--border-color);
        margin-bottom: var(--spacing);
        padding: var(--spacing);

        ul {
            list-style: none;
            margin: 0;

            li {
                display: inline-block;
            }
        }
    }

    h3 {
        margin-bottom: 0;
    }

    p.pool-name {
        color: var(--gray);
    }
</style>
