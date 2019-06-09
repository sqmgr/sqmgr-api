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
        <p>{{description}}</p>

        <template v-if="warning">
            <div class="warning">
                <p><strong>Warning!</strong><br>{{ warning }}</p>
            </div>
        </template>

        <div class="buttons">
            <template v-if="dismissButton">
                <button type="button" @click.prevent="ModalController.hide()">{{dismissButton}}</button>
            </template>
            <template v-else>
                <button type="button" class="secondary" @click.prevent="$emit('cancel-was-clicked')">Cancel</button>
                <button type="button" @click.prevent="$emit('action-was-clicked')">{{actionButton}}</button>
            </template>
        </div>
    </div>
</template>

<script>
    import ModalController from '@/controllers/ModalController'

    export default {
        name: "Prompt.vue",
        props: {
            description: {
                type: String,
                required: true,
            },
            actionButton: String,
            dismissButton: String,
            warning: String
        },
        data() {
            return {
                ModalController
            }
        }
    }
</script>

<style lang="scss" scoped>
    div.warning {
        background-color: var(--warning);
        padding: var(--minimal-spacing);
        margin-bottom: var(--spacing);

        p {
            margin: 0;
            &::before {
                content: '\26a0';
                padding-right: var(--minimal-spacing);
            }
        }
    }
</style>