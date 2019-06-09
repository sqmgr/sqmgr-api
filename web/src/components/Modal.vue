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
    <div v-if="components.length > 0">
        <template v-for="component in components">
            <div v-bind:key="component.id" class="modal-container">
                <div class="cover" @click.prevent="close"></div>
                <transition appear name="modal">
                    <div class="modal">
                        <h2 v-if="component.header === 'Error'" class="error">Error</h2>
                        <h2 v-else>{{component.header}}</h2>

                        <a class="close" href="#" @click.prevent="close"><span>Close</span></a>
                        <div class="content">
                            <component :is="component.component" v-bind="component.props" v-on="component.on"></component>
                        </div>
                    </div>
                </transition>
            </div>
        </template>
    </div>
</template>

<script>
    import ModalController from '@/controllers/ModalController'

    export default {
        name: "Modal",
        data() {
            return {
                components: [],
            }
        },
        mounted() {
            ModalController.bus.$on('show', (header, component, props = {}, on = {}) => this.components.push({
                id: Math.random(),
                header,
                component,
                props,
                on,
            }))

            ModalController.bus.$on('hide', () => this.components.pop())

            ModalController.bus.$on('hideAll', () => this.components = [])

            window.addEventListener('keyup', e => this.components.length > 0 && e.key === 'Escape' && this.components.pop())
        },
        methods: {
            close() {
                ModalController.hide()
            }
        }
    }
</script>

<style scoped lang="scss">
    div.modal-enter-active {
        animation: modal 0.2s ease-out;
    }

    @keyframes modal {
        0% {
            opacity: 0;
            margin-top: -50px;
        }
        100% {
            opacity: 1;
            margin-top: 0;
        }
    }

    div.cover {
        background-color: rgba(0, 0, 0, 0.75);
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 100;
    }

    div.modal {
        z-index: 100;
        background-color: #fff;
        border: 1px solid #ddd;
        border-radius: 3px;
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
        min-width: 200px;

        h2 {
            background-color: #f3f3f3;
            font-size: 24px;
            border-bottom: 1px solid #ddd;
            line-height: 44px;
            padding-left: 20px;

            &.error {
                background-color: #f44336;
                border-bottom-color: #b71c1c;
                color: #fff;
            }
        }

        a.close {
            color: rgba(0, 0, 0, 0.25);
            text-decoration: none;
            font-weight: bold;
            font-size: 30px;
            line-height: 40px;
            position: absolute;
            top: 3px;
            right: 14px;

            span {
                display: none;
            }

            &::after {
                content: '\2716';
            }
        }

        div.content {
            overflow: auto;
            padding: 20px 20px 0;
            max-width: 75vw;
            max-height: calc(75vh - 46px);

            & > *:last-child {
                margin-bottom: 20px;
            }
        }
    }
</style>