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
    <section class="pool" :class="{ admin: jwt.IsAdmin }">
        <template v-if="pool">
            <h3>Squares Pool - {{ this.pool.name }}</h3>

            <div class="columns">
                <div class="col-3">
                    <h4>Games in Pool</h4>

                    <div class="grids">
                        <div class="grid-row header">
                            <div>Game</div>
                            <div>Event Date</div>
                        </div>

                        <draggable v-model="grids" @start="drag=true" @end="drag=false" :disabled="!jwt.IsAdmin"
                                   handle=".handle" @change="change">
                            <div class="grid-row" v-for="grid in grids" :key="grid.id">
                                <span v-if="jwt.IsAdmin" class="handle"><i class="fas fa-grip-lines"></i> <span>=</span></span>

                                <a :href="`/pool/${token}/game/${grid.id}`">{{ grid.name }}</a>

                                <div class="event-date">
                                    <span v-if="ymd(grid.eventDate)">{{ ymd(grid.eventDate) }}</span>
                                    <span v-else class="unknown">0/0/0000</span>
                                </div>

                                <div v-if="jwt.IsAdmin" class="actions">
                                    <button type="button" class="icon" @click.prevent="customizeGrid(grid)"><i
                                            class="fas fa-cog"></i><span>Customize</span></button>
                                    <button type="button" class="icon destructive" @click.prevent="confirmDelete(grid)">
                                        <span>Delete</span><i
                                            class="fas fa-trash-alt"></i></button>
                                </div>
                            </div>
                        </draggable>
                    </div>

                    <div class="buttons" v-if="jwt.IsAdmin">
                        <button type="button" @click.prevent="createGrid">Create Game</button>
                    </div>
                </div>

                <div class="col-1">
                    <h4>Pool Settings</h4>

                    <table>
                        <tbody>
                        <tr>
                            <td>Token</td>
                            <td>{{ token }}</td>
                        </tr>
                        <tr>
                            <td>Grid Type</td>
                            <td>{{ pool.gridType }}</td>
                        </tr>
                        <tr>
                            <td>State</td>
                            <td>
                                <template v-if="isLocked">
                                    <i class="fas fa-lock"></i> Locked ({{ date(pool.locks, false) }})<br>

                                </template>
                                <template v-else>
                                    <i class="fas fa-lock-open"></i> Open<br>
                                </template>
                            </td>
                        </tr>
                        <tr>
                            <td>Created</td>
                            <td>{{ date(pool.created, true) }}</td>
                        </tr>
                        </tbody>
                    </table>

                    <div class="buttons">
                        <button v-if="isLocked" type="button" @click="unlockSquares">Open Squares</button>
                        <button v-else type="button" @click="lockSquares">Lock Squares</button>
                    </div>
                </div>
            </div>
        </template>

        <h4>Help</h4>

        <p>SqMGR allows you to create multiple games or events within a single squares pool. People will claim a square
            and then use that same square for all games in the pool. Each game will draw unique numbers.</p>

        <p>For example, Ted might claim square 5 for an entire football season, but each week he'll have a different set
            of numbers for that square (e.g., 0 and 7 for week 1, 8 and 8 for week 2, etc.).</p>
        <Modal/>
    </section>
</template>

<script>
    import api from '@/models/api'
    import ModalController from "@/controllers/ModalController";
    import Modal from "@/components/Modal";
    import GridCustomize from '@/components/GridCustomize'
    import Common from '@/common'
    import draggable from 'vuedraggable'

    export default {
        name: "Pool",
        data() {
            return {
                token: window.location.pathname.substr(6, 8),
                pool: {},
                grids: [],
                jwt: {},
            }
        },
        components: {Modal, draggable},
        mounted() {
            api.token = this.token

            api.decodedJWT()
                .then(jwt => this.jwt = jwt)

            api.getPool()
                .then(res => this.pool = res)
                .catch(err => ModalController.showError(err))

            api.getPoolGrids()
                .then(res => this.grids = res)
                .catch(err => ModalController.showError(err))
        },
        computed: {
            isLocked() {
                const locks = new Date(this.pool.locks)
                return locks.getFullYear() > 1 && locks.getTime() < new Date().getTime()
            }
        },
        methods: {
            createGrid() {
                ModalController.show('Customize Grid', GridCustomize, {}, {
                    'saved': grid => {
                        ModalController.hide()
                        this.grids.push(grid)
                    }
                })
            },
            customizeGrid(grid) {
                api.getGrid(grid.id)
                    .then(grid => {
                        ModalController.show('Customize Grid', GridCustomize, {grid}, {
                            'saved': grid => {
                                ModalController.hide()
                                let index = -1
                                for (let i = 0; i < this.grids.length; i++) {
                                    if (this.grids[i].id === grid.id) {
                                        index = i
                                    }
                                }

                                if (index >= 0) {
                                    this.grids.splice(index, 1, grid)
                                }
                            }
                        })

                    })
            },
            confirmDelete(grid) {
                ModalController.showPrompt('Are you sure?', `Do you really want to delete "${grid.name}"`, {
                    actionButton: 'Delete It',
                    action: () => {
                        api.deleteGrid(grid.id)
                            .then(() => {
                                const index = this.grids.indexOf(grid)
                                if (index >= 0) {
                                    this.grids.splice(index, 1)
                                }
                                ModalController.hide()
                            })
                            .catch(err => ModalController.showError(err))
                    }
                })

                return false
            },
            ymd(eventDate) {
                const d = Common.NewDateWithoutTimezone(eventDate)
                if (d.getFullYear() <= 1) {
                    return ''
                }

                return d.toLocaleDateString(Common.DateOptions)
            },
            date(date, includeTime) {
                const d = new Date(date)
                if (d.getFullYear() <= 1) {
                    return ''
                }

                return includeTime ? d.toLocaleString(Common.DateTimeOptions) : d.toLocaleDateString(Common.DateTimeOptions)
            },
            unlockSquares() {
                const promptOpts = {
                    actionButton: "Unlock Squares",
                    action: () => {
                        ModalController.hide()

                        api.unlockPool()
                            .then(pool => this.pool = pool)
                            .catch(err => ModalController.showError(err))
                    }
                }

                ModalController.showPrompt("Unlock the squares?", "Are you sure you want to open the squares back up again for users to claim?", promptOpts)
            },
            lockSquares() {
                api.getSquares()
                    .then(squares => {
                        const promptOpts = {
                            actionButton: "Lock Squares",
                            action: () => {
                                ModalController.hide()

                                api.lockPool()
                                    .then(pool => this.pool = pool)
                                    .catch(err => ModalController.showError(err))
                            }
                        }

                        const unclaimedSquares = Object.values(squares).filter(s => s.state === 'unclaimed')
                        if (unclaimedSquares.length > 0) {
                            promptOpts.warning = "There are still unclaimed squares."
                        }

                        ModalController.showPrompt("Lock the squares?", "Are you sure you want to lock the squares? Users will no longer be allowed to claim any open squares.", promptOpts)
                    })
                    .catch(err => ModalController.showError(err))
            },
            change() {
                api.reorderGrids(this.grids.map(g => g.id))
                    .catch(err => ModalController.showError(err))
            }
        }
    }
</script>

<style lang="scss" scoped>
    div.buttons {
        margin-top: var(--spacing);
    }

    button.icon span {
        display: none;
    }

    .handle {
        color: #aaa;
        cursor: move;

        span {
            display: none;
        }
    }

    div.grid-row {
        align-items: center;
        display: grid;
        grid-template-columns: 1fr 100px;
        padding: calc(2 * var(--minimal-spacing));

        &:not(.header) {
            border-bottom: 1px solid var(--border-color);
        }

        &:nth-child(odd) {
            background-color: var(--light-gray);
        }

        &.header {
            font-weight: bold;
            background-color: var(--midnight-gray);
            color: #fff;

            & > div {
                justify-self: stretch;
            }

        }

        span.unknown {
            color: var(--gray);
        }
    }

    .admin {
        div.grid-row {
            grid-template-columns: 40px 1fr 100px 130px;

            & > :first-child {
                justify-self: center;
            }

            div.actions {
                text-align: right;
            }

            @media(max-width: 600px) {
                & > :nth-child(4) {
                    grid-column: 1 / 5;
                    padding-top: var(--minimal-spacing);
                    text-align: right;
                }
            }

            &.header {
                & > div {
                    justify-self: stretch;
                }

                & > :first-child {
                    grid-column: 2 / 3;
                }

                & > :nth-child(2) {
                    grid-column: 3 / 5;
                }
            }
        }
    }
</style>