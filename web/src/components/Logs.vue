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
    <section class="audit-log">
        <h4>Audit Log</h4>

        <template v-if="showAddNote">
            <p class="add-note"><a href="#" class="add-note" @click.prevent="addNote">Add Note</a></p>
        </template>

        <table>
            <thead>
            <tr>
                <th>ID</th>
                <th>Time</th>
                <th>State</th>
                <th>Claimant</th>
                <th>Remote Addr</th>
                <th>Note</th>
            </tr>
            </thead>
            <tbody>
            <tr v-for="log in logs">
                <td class="square-id">#{{ log.squareID }}</td>
                <td class="created">{{ datetime(log.created) }}</td>
                <td class="state">{{ log.state }}</td>
                <td class="claimant">{{ log.claimant }}</td>
                <td class="remote-addr">{{ log.remoteAddr }}</td>
                <td class="note">{{ log.note }}</td>
            </tr>
            </tbody>
        </table>
    </section>
</template>

<script>
    import Note from './Note.vue'
    import Vue from 'vue'
    import Modal from '../modal'
    import api from '../models/api'
    export default {
        name: "Logs",
        props: {
            squareId: Number,
            logs: Array,
            showAddNote: Boolean,
        },
        methods: {
            addNote() {
                const vm = new (Vue.extend(Note))
                vm.$on('submit', note => {
                    if (note) {
                        api.updateSquare(this.squareId, {note})
                            .then(() => {
                                Modal.close()
                                this.$emit('note-added')
                            })
                            .catch(err => Modal.showError(err))
                        return
                    }

                    Modal.close()
                })

                Modal.show(vm.$mount().$el)
            },
            datetime(dt) {
                return new Date(dt).toLocaleDateString('default', {year: '2-digit', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric'})
            }
        }
    }
</script>
