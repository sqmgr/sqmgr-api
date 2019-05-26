<template>
    <section class="audit-log">
        <h4>Audit Log</h4>

        <template v-if="addNote">
            <p class="add-note"><a href="#" class="add-note">Add Note</a></p>
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
    export default {
        name: "Logs",
        props: {
            logs: Array,
            addNote: Boolean,
        },
        methods: {
            datetime(dt) {
                return new Date(dt).toLocaleDateString('default', {year: '2-digit', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric'})
            }
        }
    }
</script>

<style scoped>

</style>