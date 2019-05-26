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

export class Modal {
    constructor(optionalParent) {
        this.modals = []
    }

    element(childNode) {
        const modal = document.createElement('div')
        modal.classList.add('modal')

        const closeLink = document.createElement('a')
        closeLink.setAttribute('href', '#')
        closeLink.classList.add('close')

        const closeSpan = document.createElement('span')
        closeSpan.classList.add('close')
        closeSpan.textContent = 'Close'

        const container = document.createElement('div')
        container.classList.add('container')

        const containerContent = document.createElement('div')
        containerContent.classList.add('container-content')

        closeLink.appendChild(closeSpan)
        modal.appendChild(closeLink)

        containerContent.appendChild(childNode)
        container.appendChild(containerContent)
        modal.appendChild(container)

        modal.onclick = e => {
            e.preventDefault()
            this.close()
        }

        closeLink.onclick = e => {
            e.preventDefault()
            this.close()
        }

        container.onclick  = e => e.stopPropagation()

        return modal
    }

    show(childNode) {
        const modal = this.element(childNode)
        this.modals.push(modal)
        document.body.appendChild(modal)

        if (this.modals.length === 1) {
            this._keyup = event => {
                if (event.key === 'Escape') {
                    event.stopPropagation()
                    this.close()
                }
            }

            window.addEventListener('keyup', this._keyup)
        }
    }

    close() {
        const modal = this.modals.pop()
        if (this.modals.length === 0) {
            window.removeEventListener('keyup', this._keyup)
        }

        if (modal) {
            modal.remove()
            return true
        }

        return false
    }

    closeAll() {
        while (this.close())
            ;
    }

    showError(errorMsg) {
        const div = document.createElement('div')
        div.classList.add('error')
        div.textContent = errorMsg

        this.show(div)
    }
}

export default new Modal()
