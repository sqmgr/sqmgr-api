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
    <form class="claim-square" @submit.prevent="submit">
        <div class="field">
            <label for="name">Name</label>
            <input type="text" id="name" name="name" placeholder="Your name" v-model="name" required ref="name">
        </div>

        <div class="buttons">
            <button type="button" class="secondary" @click.prevent="Modal.close()">Cancel</button>
            <button type="submit" name="submit" value="Claim">Claim</button>
        </div>
    </form>
</template>

<script>
    import api from '../models/api'
    import Modal from '../modal'

    export default {
        name: "Claim.vue",
        props: {
            squareId: Number,
        },
        data() {
            let name = null
            if (window.hasOwnProperty('localStorage')) {
                name = localStorage.getItem('claimant-name')
            }

            return {
                name,
                Modal,
            }
        },
        mounted() {
            setTimeout(() => this.$refs.name.select(), 25)
        },
        methods: {
            submit() {
                if (!this.name) {
                    return
                }

                if (window.hasOwnProperty('localStorage')) {
                    localStorage.setItem('claimant-name', this.name)
                }

                api.claimSquare(this.squareId, this.name)
                    .then(() => Modal.closeAll())
                    .catch(err => Modal.showError(err))
            }
        }
    }
</script>
