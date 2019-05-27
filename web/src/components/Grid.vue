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
        <div :class="`squares ${pool.gridType}`">
            <div class="spacer">&nbsp;</div>

            <div class="team home-team" ref="home-team"><span>{{ grid.settings.homeTeamName }}</span></div>
            <div v-for="n in 10" :class="`score home-score home-score-${n-1}`"></div>

            <div class="team away-team" ref="away-team"><span>{{ grid.settings.awayTeamName }}</span></div>
            <div v-for="n in 10" :class="`score away-score away-score-${n-1}`"></div>

            <template v-for="n in numSquares">
                <Square :opaque-user-id="opaqueUserID" :sq-id="n" :square-data="squares[n] || {}"></Square>
            </template>
        </div>

        <template v-if="isAdmin">
            <Logs :show-add-note="false" :logs="logs"/>
        </template>
    </div>
</template>

<script>
    import Square from './Square.vue'
    import Logs from './Logs.vue'
    import api from '../models/api.js'
    import EventBus from '../models/EventBus'

    api.token = SqMGR.gridConfig.pool.token

    const Config = {
        Squares: {
            std100: 100,
            std25: 25,
        }
    }
    export default {
        name: "Grid",
        components: {
            Square,
            Logs,
        },
        data() {
            return {
                ...SqMGR.gridConfig,
                numSquares: Config.Squares[SqMGR.gridConfig.pool.gridType],
                squares: [],
                logs: [],
            }
        },
        beforeMount() {
            this.loadData()
            EventBus.$on('data-updated', () => this.loadData())
        },
        mounted() {
            const ht = this.$refs["home-team"]
            ht.style.setProperty('--team-primary', this.grid.settings.homeTeamColor1)
            ht.style.setProperty('--team-secondary', this.grid.settings.homeTeamColor2)

            const at = this.$refs["away-team"]
            at.style.setProperty('--team-primary', this.grid.settings.awayTeamColor1)
            at.style.setProperty('--team-secondary', this.grid.settings.awayTeamColor2)
        },
        methods: {
            loadData() {
                api.getSquares()
                    .then(res => {
                        this.squares = res
                    })

                api.getLogs()
                    .then(res => {
                        this.logs = res
                    })
            }
        }
    }
</script>

<style>
</style>