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
	var buffer = 1 // spacing for error message

	document.querySelectorAll('input[type="password"]').forEach(function(input) {
		var id = input.getAttribute('id')
		var confirmInput, checkPasswordFn, noMatchElem

		if (id.indexOf('confirm-') === 0) {
			return
		}

		confirmInput = document.getElementById('confirm-'+id)
		if (!confirmInput) {
			return
		}

		checkPasswordFn = function() {
			var clientRect
			if (input.value === confirmInput.value) {
				if (noMatchElem) {
					noMatchElem.remove()
					noMatchElem = null
				}

				confirmInput.setCustomValidity("")
				return
			}

			confirmInput.setCustomValidity("Passwords do not match")

			if (noMatchElem) {
				return
			}

			clientRect = confirmInput.getBoundingClientRect()

			noMatchElem = document.createElement('div')
			noMatchElem.textContent = 'The passwords do not match'
			noMatchElem.style.left = clientRect.left+'px'
			noMatchElem.style.top = clientRect.top+clientRect.height+buffer+'px'
			noMatchElem.classList.add('input-error')
			document.body.appendChild(noMatchElem)
		}

		input.addEventListener('keyup', checkPasswordFn)
		confirmInput.addEventListener('keyup', checkPasswordFn)
	})

	document.querySelectorAll('input[type="time"]').forEach(function(input) {
		console.log(typeof(input.type))
		if (input.value === '') {
			input.value = '00:00'
		}
	})
})
