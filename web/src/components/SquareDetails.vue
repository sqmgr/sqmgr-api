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
    <div class="square-details">
        <table>
            <tbody>
            <tr>
                <td>Square ID</td>
                <td class="square-id">{{square.squareID}}</td>
            </tr>
            <tr>
                <td>Claimant</td>
                <td class="claimant">
                    <template v-if="editClaimant">
                        <form @submit.prevent="saveNewClaimant" class="standalone">
                            <input type="text"
                                   v-model="newClaimant"
                                   ref="claimantInput"
                                   @keyup="onKeyup($event)"
                            >
                        </form>
                    </template>
                    <template v-else-if="isAdmin">
                        <a href="#" @click.prevent="editClaimant=true">{{square.claimant}}</a>
                    </template>
                    <template v-else>
                        <span>{{square.claimant}}</span>
                    </template>
                </td>
            </tr>
            <tr>
                <td>State</td>
                <td class="state">
                    <template v-if="!isAdmin || square.state === 'unclaimed'">
                        {{square.state}}
                    </template>
                    <template v-else>
                        <select name="state" v-model="form.state" @change="stateDidChange">
                            <option v-for="state in states" :value="state">{{state}}</option>
                        </select>
                    </template>
                </td>
            </tr>
            <tr>
                <td>Last Modified</td>
                <td class="modified">{{new Date(square.modified).toLocaleDateString('default',
                    Common.DateTimeOptions)}}
                </td>
            </tr>
            </tbody>
        </table>

        <template v-if="canClaim">
            <div class="buttons">
                <button type="button" @click.prevent="claimSquare">Claim</button>
            </div>
        </template>
        <template v-if="canUnclaim">
            <div class="buttons">
                <button type="button" class="destructive" @click.prevent="unclaimSquare">Relinquish Claim</button>
            </div>
        </template>

        <template v-if="isAdmin">
            <Logs @note-added="reloadData" :square-id="this.square.squareID" :logs="square.logs"
                  :show-add-note="true"></Logs>
        </template>
    </div>
</template>

<script>
    import Logs from './Logs.vue'
    import api from '../models/api'
    import Claim from './Claim.vue'
    import Note from './Note.vue'
    import Common from '../common'
    import ModalController from '@/controllers/ModalController'

    export default {
        name: "SquareDetails",
        components: {Logs},
        props: {
            data: Object
        },
        data() {
            return {
                Common,
                editClaimant: false,
                newClaimant: null,
                loadedData: null,
                form: {
                    state: this.loadedData ? this.loadedData.state : this.data.state,
                },
                states: SqMGR.gridConfig.gridSquareStates,
            }
        },
        computed: {
            square() {
                return this.loadedData || this.data
            },
            isAdmin() {
                return SqMGR.gridConfig.isAdmin
            },
            isLocked() {
                const locks = new Date(SqMGR.gridConfig.pool.locks)
                return locks.getFullYear() > 1 && locks.getTime() < new Date().getTime()
            },
            canClaim() {
                return this.square.state === 'unclaimed' && ( !this.isLocked || this.isAdmin )
            },
            opaqueUserID() {
                return SqMGR.gridConfig.opaqueUserID
            },
            canUnclaim() {
                if (this.square.state !== 'claimed') return false
                if (this.isLocked) return false
                return this.square.opaqueUserID === this.opaqueUserID
            }
        },
        methods: {
            claimSquare() {
                ModalController.show('Claim Square', Claim, {squareId: this.square.squareID})
            },
            unclaimSquare() {
                api.unclaimSquare(this.square.squareID)
                    .then(() => ModalController.hide())
                    .catch(err => ModalController.showError(err))
            },
            stateDidChange() {
                ModalController.show('Note', Note, {}, {
                    submit: note => {
                        api.setSquareState(this.square.squareID, this.form.state, note)
                            .then(() => {
                                ModalController.hide()
                                this.reloadData()
                            })
                            .catch(err => ModalController.showError(err.message))
                    },
                })
            },
            reloadData() {
                api.getSquare(this.square.squareID)
                    .then(res => this.loadedData = res)
            },
            onKeyup(event) {
                if (event.key === 'Escape') {
                    event.stopPropagation()
                    this.editClaimant = false
                }
            },
            saveNewClaimant() {
                if (this.newClaimant === this.square.claimant) {
                    this.editClaimant = false
                    return
                }

                if (this.newClaimant.match(/\w/)) {
                    api.renameSquare(this.square.squareID, this.newClaimant)
                        .then(res => { this.loadedData = res; this.editClaimant = false })
                        .catch(err => ModalController.showError(err))
                }
            }
        },
        watch: {
            editClaimant(newVal) {
                if (newVal) {
                    this.newClaimant = this.square.claimant
                    this.$nextTick()
                        .then(() => {
                            this.$refs.claimantInput.select()
                        })
                }
            }
        }
    }
</script>

<style scoped lang="scss">
    table {
        width: 100%;

        td:last-child {
            font-weight: bold;
            text-align: right;
        }
    }

    div.buttons {
        margin-top: var(--spacing);
        text-align: left;
    }
</style>