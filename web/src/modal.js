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

export default class Modal {
    constructor(optionalParent) {
        this.parent = optionalParent // the parent modal (optional)
        this.node = null
        this.nestedModal = null
        this._keyup = this.keyup.bind(this)
    }

    nest() {
        if (this.nestedModal) {
            this.nestedModal.close()
        }

        this.nestedModal = new Modal(this)
        return this.nestedModal
    }

    nestedDidClose() {
        this.nestedModal = null
    }

    close() {
        window.removeEventListener('keyup', this._keyup)

        if (this.node) {
            this.node.dispatchEvent(new Event('modalclose'))

            this.node.remove()
            this.node = null
        }

        if (this.parent) {
            this.parent.nestedDidClose()
        }

        return false
    }

    show(childNode) {
        const node = document.createElement('div')
        node.classList.add('modal')

        const closeLink = document.createElement('a')
        closeLink.setAttribute('href', '#')
        closeLink.classList.add('close')

        const closeText = document.createElement('span')
        closeText.textContent = 'Close'

        const container = document.createElement('div')
        container.classList.add('container')

        const content = document.createElement('div')
        content.classList.add('container-content')

        closeLink.appendChild(closeText)
        container.appendChild(closeLink)
        content.appendChild(childNode)
        container.appendChild(content)
        node.appendChild(container)

        container.onclick = function (event) {
            event.cancelBubble = true
        }

        if (this.node) {
            this.close()
        }

        this.node = node

        this.node.onclick = closeLink.onclick = this.close.bind(this)

        document.body.appendChild(node)

        window.addEventListener('keyup', this._keyup)

        return node
    }

    showError(errorMsg) {
        const div = document.createElement('div')
        div.classList.add('error')
        div.textContent = errorMsg

        this.show(div)
    }

    keyup(event) {
        if (this.nestedModal) {
            return
        }

        if (event.key === 'Escape') {
            event.stopPropagation()
            this.close()
            return
        }
    }
}
