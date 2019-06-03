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
    <div class="customize-team">
        <div class="field">
            <label :for="`name-${id}`" class="optional">{{name}} Name</label>
            <input :id="`name-${id}`" :maxlength="teamNameMaxLength" :name="`name-${id}`" :placeholder="name"
                   @keydown="keydown"
                   @keyup="keyup" autocomplete="off" type="text" v-model="value.name"
                   @blur="hideSuggestions">

            <div v-show="suggestTeams.length > 0" class="suggest">
                <div class="suggest-item" v-for="(team, idx) in suggestTeams"
                     v-bind:class="{ selected: idx === suggestIdx }">
                    <a href="#" @click.prevent="selectTeam(team)" @mouseover="suggestIdx=idx">{{team.name}} ({{team.league}})</a>
                </div>
            </div>
        </div>

        <div class="field">
            <label :for="`color-1-${id}`">{{name}} Colors</label>
            <input type="color" :id="`color-1-${id}`" :name="`color-1-${id}`" placeholder="#000000"
                   v-model="value.color1">
            <input type="color" :id="`color-2-${id}`" :name="`color-2-${id}`" placeholder="#000000"
                   v-model="value.color2">
        </div>
    </div>
</template>

<script>
    import findTeam from '../models/TeamColors'

    export default {
        name: "GridCustomizeTeam",
        watch: {
            teamName(name) {
                this.suggestTeams = findTeam(name)
                this.suggestIdx = -1
            },
        },
        computed: {
            teamName() {
                return this.value.name
            }
        },
        data() {
            return {
                id: 0,
                teamNameMaxLength: 25, // TODO
                notesMaxLength: 200, // TODO
                suggestTeams: [],
                suggestIdx: -1,
            }
        },
        props: {
            name: {
                type: String,
                required: true,
            },
            value: {
                type: Object,
                required: true,
            }
        },
        mounted() {
            this.id = this._uid
        },
        methods: {
            selectTeam(team) {
                this.value.name = team.name
                this.value.color1 = '#' + team.colors.hex[0]
                this.value.color2 = '#' + team.colors.hex[1]
                setTimeout(() => this.suggestTeams = [], 1)
            },
            selectUp() {
                this.suggestIdx--

                if (this.suggestIdx < -1) {
                    this.suggestIdx = this.suggestTeams.length - 1
                }
            },
            selectDown() {
                this.suggestIdx++

                if (this.suggestIdx >= this.suggestTeams.length) {
                    this.suggestIdx = -1
                }
            },
            keyup(event) {
                if (event.key === 'Escape') {
                    if (this.suggestTeams.length > 0) {
                        this.suggestTeams = []
                        event.stopPropagation()
                    }
                }
            },
            keydown(event) {
                switch (event.key) {
                    case 'Enter':
                        if (this.suggestIdx >= 0) {
                            event.preventDefault()
                            this.selectTeam(this.suggestTeams[this.suggestIdx])
                            return
                        }
                    case 'ArrowDown':
                        this.selectDown()
                        break
                    case 'ArrowUp':
                        this.selectUp()
                        break
                    case 'n':
                        if (event.ctrlKey) {
                            this.selectDown()
                        }
                        break
                    case 'p':
                        if (event.ctrlKey) {
                            this.selectUp()
                        }
                        break
                }
            },
            hideSuggestions() {
                // do this behind a timeout just in case user clicked something
                setTimeout( () => {
                    this.suggestIdx = -1
                    this.suggestTeams = []
                }, 100)
            }
        }
    }
</script>

<style scoped lang="scss">
    div.customize-team {
        position: relative;
    }

    div.suggest {
        background-color: #fff;
        position: absolute;
        margin-top: 2px;
        z-index: 1;
        padding: var(--minimal-spacing) calc(2 * var(--minimal-spacing));
        box-shadow: 2px 4px 4px rgba(0, 0, 0, 0.2);
        border: 1px solid var(--border-color);
        width: calc(100% - 2px);

        div.suggest-item {
            a {
                border-bottom: 1px solid var(--border-color);
                color: #000;
                display: block;
                padding: var(--minimal-spacing);

                &:hover {
                    text-decoration: none;
                }
            }

            &:last-child a {
                border-bottom: none;
            }

            &.selected {
                background-color: var(--primary);

                a {
                    color: #fff;

                    &:hover {
                        text-decoration: none;
                    }
                }
            }
        }
    }
</style>