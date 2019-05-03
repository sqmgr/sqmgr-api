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

(function() {
    const loading = function() {
       this.calls = 0

        const div = document.createElement('div')
        div.classList.add('loading-indicator')
        div.appendChild(document.createElement('span'))

        this.loadingDiv = div
    }

    loading.prototype.show = function() {
        if (this.calls === 0) {
            document.body.appendChild(this.loadingDiv)
        }

        this.calls++
    }

    loading.prototype.hide = function() {
        if (this.calls <= 0) {
            throw new Error('hide() called too many times')
        }

        this.calls--

        if (this.calls === 0) {
            this.loadingDiv.remove()
        }
    }

    SqMGR.Loading = new loading()
})()
