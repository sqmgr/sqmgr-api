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

SqMGR.Modal = function() {
    this.modal = null
    this._keyup = this.keyup.bind(this)
}

SqMGR.Modal.prototype.close = function() {
    window.removeEventListener('keyup', this._keyup)

    if (this.modal) {
        this.modal.remove()
        this.modal = null
    }
}

SqMGR.Modal.prototype.show = function(node) {
    const modal = document.createElement('div')
    modal.classList.add('modal')

    const closeLink = document.createElement('a')
    closeLink.setAttribute('href', '#')
    closeLink.classList.add('close')

    const closeText = document.createElement('span')
    closeText.textContent = 'Close'

    const container = document.createElement('div')
    container.classList.add('container')

    closeLink.appendChild(closeText)
    container.appendChild(closeLink)
    container.appendChild(node)
    modal.appendChild(container)

    container.onclick = function(event) {
        event.cancelBubble = true
    }

    if (this.modal) {
        this.modal.close()
    }

    this.modal = modal

    this.modal.onclick = closeLink.onclick = this.close.bind(this)

    document.body.appendChild(modal)

    window.addEventListener('keyup', this._keyup)
}


SqMGR.Modal.prototype.keyup = function(event) {
    if (event.key === 'Escape') {
        this.close()
        return
    }
}
