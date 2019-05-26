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