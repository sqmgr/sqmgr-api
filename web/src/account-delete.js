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

import SetupForm from './setup-form'
import Vue from 'vue'
import Modal from '@/components/Modal'
import ModalController from '@/controllers/ModalController'
import AccountDeleteConfirmation from '@/components/AccountDeleteConfirmation'

window.addEventListener('load', function () {
    SetupForm()

    new Vue({
        render: h => h(Modal),
        el: '#modal'
    })

    document.querySelector('button.destructive').onclick = () => {
        ModalController.show('Are you sure?', AccountDeleteConfirmation, {
            expectedEmail: SqMGR.ExpectedEmail,
        })
    }
})
