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
        <h4>Square Details</h4>

        <template v-if="canClaim">
            <p class="claim"><a href="#" @click.prevent="claimSquare">Claim this square!</a></p>
        </template>
        <template v-if="canUnclaim">
            <p class="unclaim"><a href="#" @click.prevent="unclaimSquare">Relinquish claim</a></p>
        </template>

        <table>
            <tbody>
            <tr>
                <td>Square ID</td>
                <td class="square-id">{{square.squareID}}</td>
            </tr>
            <tr>
                <td>Claimant</td>
                <td class="claimant">{{square.claimant}}</td>
            </tr>
            <tr>
                <td>State</td>
                <td class="state">
                    <template v-if="square.state === 'unclaimed'">
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
                <td class="modified">{{square.modified}}</td>
            </tr>
            </tbody>
        </table>

        <template v-if="isAdmin">
            <Logs @note-added="reloadData" :square-id="this.square.squareID" :logs="square.logs" :show-add-note="true"></Logs>
        </template>
    </div>
</template>

<script>
    import Logs from './Logs.vue'
    import api from '../models/api'
    import Modal from '../modal'
    import Claim from './Claim.vue'
    import Note from './Note.vue'
    import Vue from 'vue'

    export default {
        name: "SquareDetails",
        components: {Logs},
        props: {
            data: Object
        },
        data() {
            return {
                loadedData: null,
                form: {
                    state: this.data.state,
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
                return SqMGR.gridConfig.pool.isLocked
            },
            canClaim() {
                return this.data.state === 'unclaimed' && !this.isLocked
            },
            opaqueUserID() {
                return SqMGR.gridConfig.opaqueUserID
            },
            canUnclaim() {
                if (this.data.state !== 'claimed') return false
                if (this.isAdmin) return true
                if (this.isLocked) return false
                return this.data.opaqueUserID === this.opaqueUserID
            }
        },
        methods: {
            claimSquare() {
                const Component = Vue.extend(Claim)
                const vm = new Component({
                    propsData: {
                        squareId: this.data.squareID,
                    }
                })

                Modal.show(vm.$mount().$el)
            },
            unclaimSquare() {
                api.unclaimSquare(this.data.squareID)
                    .then(() => Modal.close())
                    .catch(err => Modal.showError(err))
            },
            stateDidChange() {
                const vm = new (Vue.extend(Note))
                vm.$on('submit', note => {
                    api.setSquareState(this.data.squareID, this.form.state, note)
                        .then(() => {
                            Modal.close()
                            this.reloadData()
                        })
                        .catch(err => Modal.show(err))
                })

                Modal.show(vm.$mount().$el)
            },
            reloadData() {
                api.getSquare(this.square.squareID)
                    .then(res => this.loadedData = res)
            }
        }
    }
</script>
