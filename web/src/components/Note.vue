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
    <form class="add-note" @submit.prevent="$emit('submit', form.note)">
        <div class="field">
            <label for="note">Note</label>
            <input type="text" id="note" name="note" v-model="form.note" ref="name" placeholder="Reason for change">
        </div>

        <div class="buttons">
            <button type="button" class="secondary" @click.prevent="Modal.close()">Cancel</button>
            <button type="submit" name="submit">Make Note</button>
        </div>
    </form>
</template>

<script>
    import Modal from '../modal'
    export default {
        name: "Note.vue",
        data() {
            return {
                form: {
                    note: null
                }
            }
        },
        mounted() {
            setTimeout(() => this.$refs.name.focus(), 1)
        },
        computed: {
            Modal: () => Modal,
        }
    }
</script>
