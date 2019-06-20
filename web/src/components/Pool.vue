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
    <section class="pool">
        <template v-if="pool">
            <h3>Squares Pool - {{ this.pool.name }}</h3>

            <div class="columns">
                <div class="col-3">
                    <table>
                        <thead>
                        <tr>
                            <th>Game</th>
                            <th>Date</th>
                            <th v-if="jwt.IsAdmin">&nbsp;</th>
                        </tr>
                        </thead>
                        <tbody v-if="grids">
                        <tr v-for="grid in grids">
                            <td><a :href="`/pool/${token}/game/${grid.id}`">{{ grid.name }}</a></td>
                            <td>{{ ymd(grid.eventDate) }}</td>
                            <td class="actions" v-if="jwt.IsAdmin">
                                <button type="button" @click.prevent="customizeGrid(grid)"><i class="fas fa-cog"></i></button>
                                <button type="button" class="destructive" @click.prevent="confirmDelete(grid)"><i class="fas fa-trash-alt"></i></button>
                            </td>
                        </tr>
                        </tbody>
                    </table>

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
                            <td>Squares Locked</td>
                            <td>
                                <template v-if="isLocked">
                                    {{ date(pool.locks) }}
                                    <i class="fas fa-lock"></i>
                                </template>
                                <template v-else>
                                    <button type="button" @click="lockSquares">Lock Squares</button>
                                </template>
                            </td>
                        </tr>
                        <tr>
                            <td>Created</td>
                            <td>{{ date(pool.created) }}</td>
                        </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </template>

        <Modal/>
    </section>
</template>

<script>
    import api from '@/models/api'
    import ModalController from "@/controllers/ModalController";
    import Modal from "@/components/Modal";
    import GridCustomize from '@/components/GridCustomize'
    import Common from '@/common'

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
        components: {Modal},
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
                api.createGrid()
                    .then(grid => {
                        ModalController.show('Customize Grid', GridCustomize, {
                            gridID: grid.id,
                        }, {
                            'modal-aborted': () => {
                                api.deleteGrid(grid.id)
                                    .catch(err => ModalController.showError(err))
                            },
                            'saved': grid => {
                                ModalController.hide()
                                this.grids.push(grid)
                            }
                        })
                    })
                    .catch(err => ModalController.showError(err))
            },
            customizeGrid(grid) {
                ModalController.show('Customize Grid', GridCustomize, {
                    gridID: grid.id
                }, {
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
                    return 'Not specified'
                }

                return d.toLocaleDateString(Common.DateOptions)
            },
            date(date) {
                const d = new Date(date)
                if (d.getFullYear() <= 1) {
                    return ''
                }

                return d.toLocaleString(Common.DateTimeOptions)
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
            }
        }
    }
</script>

<style lang="scss" scoped>
    section.pool table {
        width: 100%;

        td.actions {
            text-align: right;
        }
    }

    div.buttons {
        margin-top: var(--spacing);
    }
</style>