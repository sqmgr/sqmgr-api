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

window.addEventListener('load', function() {
	var buffer = 100
	var notes = document.getElementById('notes')
	var remainingEl = null
	var checkRemaining = function() {
		var remainder = SqMGR.NotesMaxLength - this.value.length
		if (remainder <= buffer) {
			if (!remainingEl) {
				remainingEl = document.createElement('div')
				remainingEl.classList.add('remaining')
				this.parentNode.insertBefore(remainingEl, this.nextSibling)
			}

			remainingEl.textContent = remainder
		} else {
			if (remainingEl) {
				remainingEl.remove()
				remainingEl = null
			}
		}
	}

	notes.onkeyup = notes.onpaste = checkRemaining
	checkRemaining.apply(notes)

	document.querySelector('input[name="lock-tz"]').value = new Date().getTimezoneOffset()

	const pad = function(val) {
		if (val < 10) {
			return "0" + val
		}

		return val
	}

	document.querySelector('a[class="lock-now"]').onclick = function() {
		const now = new Date()
	    document.getElementById('lock-date').value = now.getFullYear() + "-" + pad(now.getMonth()+1) + "-" + pad(now.getDate())
		document.getElementById('lock-time').value = pad(now.getHours()) + ":" + pad(now.getMinutes())
		return false
	}
})
