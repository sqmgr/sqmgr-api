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

window.addEventListener('load', function () {
    SetupForm()

    const modal = document.querySelector('div.modal'),
        cover = document.createElement('div'),
        closeLink = modal.querySelector('a.close'),
        emailInput = modal.querySelector('input[name="email"]'),
        expectedEmail = modal.querySelector('input[name="expected-email"]').value,
        submitButton = modal.querySelector('input[type="submit"]')

    modal.remove()
    modal.style.display = 'block'

    cover.classList.add('cover')

    function closeModal() {
        window.removeEventListener('keyup', checkForEscape)
        modal.remove()
        cover.remove()
        emailInput.value = ''
    }

    cover.onclick = closeModal
    closeLink.onclick = closeModal

    function checkForEscape(e) {
        if (e.key === 'Escape') {
            closeModal()
        }
    }

    document.querySelector('button.destructive').onclick = function () {
        if (document.querySelector('div.modal')) {
            return
        }

        document.body.appendChild(cover)
        document.body.appendChild(modal)
        emailInput.focus()
        window.addEventListener('keyup', checkForEscape)
    }

    emailInput.onkeyup = emailInput.onblur = function () {
        if (this.value === expectedEmail) {
            submitButton.removeAttribute("disabled")
        } else {
            submitButton.setAttribute("disabled", "disabled")
        }
    }
})
