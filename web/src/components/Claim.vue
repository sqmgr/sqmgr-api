<template>
    <form class="claim-square" @submit.prevent="submit">
        <div class="field">
            <label for="name">Name</label>
            <input type="text" id="name" name="name" placeholder="Your name" v-model="name" required ref="name">
        </div>

        <div class="buttons">
            <input type="submit" name="submit" value="Claim">
            <a href="#" class="cancel" @click.prevent="Modal.close">Cancel</a>
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
            return {
                name: null
            }
        },
        mounted() {
            setTimeout(() => this.$refs.name.focus(), 10)
        },
        methods: {
            submit() {
                if (!this.name) {
                    return
                }

                api.claimSquare(this.squareId, this.name)
                    .then(() => Modal.closeAll())
                    .catch(err => Modal.showError(err))
            }
        }
    }
</script>

<style scoped>

</style>