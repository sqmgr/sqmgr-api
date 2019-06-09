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
    import api from '../models/api'
    import Common from '../common'
    import ModalController from '@/controllers/ModalController'

    export default {
        name: "Logs",
        props: {
            squareId: Number,
            logs: Array,
            showAddNote: Boolean,
        },
        methods: {
            addNote() {
                ModalController.show('Add Note', Note, {}, {
                    submit: note => {
                        if (note) {
                            api.updateSquare(this.squareId, {note})
                                .then(() => {
                                    ModalController.hide()
                                    this.$emit('note-added')
                                })
                                .catch(err => ModalController.showError(err))
                            return
                        }

                        ModalController.hide()
                    }
                })
            },
            datetime(dt) {
                return new Date(dt).toLocaleDateString('default', Common.DateTimeOptions)
            }
        }
    }
</script>

<style lang="scss" scoped>
section.audit-log {
    border-top: 1px solid var(--hr-color);
    padding-top: var(--spacing);
    overflow: auto;
}
</style>
