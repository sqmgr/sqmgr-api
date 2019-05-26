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
    <div :class="`square ${squareData.state}`" @click.prevent="didClickSquare">
        <span class="square-id">{{ sqId }}</span>
        <span class="name">{{ squareData.claimant }}</span>

        <template v-if="squareData.opaqueUserID === opaqueUserId">
            <span class="owned"></span>
        </template>
    </div>
</template>

<script>
    import api from '../models/api'
    import SquareDetails from './SquareDetails.vue'
    import Vue from 'vue'
    import Modal from '../modal'

    export default {
        name: "Square.vue",
        props: {
            'sqId': Number,
            'squareData': Object,
            'opaqueUserId': String,
        },
        methods: {
            didClickSquare() {
                api.getSquare(this.sqId)
                    .then(data => {
                        const Component = Vue.extend(SquareDetails)
                        const vm = new Component({
                            propsData: {
                                data,
                            }
                        })

                        Modal.show(vm.$mount().$el)
                    })
            }
        }
    }
</script>

<style scoped>

</style>