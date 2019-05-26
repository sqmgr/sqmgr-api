<template>
    <form class="add-note" @submit.prevent="$emit('submit', form.note)">
        <div class="field">
            <label for="note">Note</label>
            <input type="text" id="note" name="note" v-model="form.note" ref="name" placeholder="Reason for change">
        </div>

        <div class="buttons">
            <input type="submit" name="submit" value="Claim">
            <a href="#" class="cancel" @click.prevent="Modal.close()">Cancel</a>
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

<style scoped>

</style>