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

Date.prototype.toLocalTimeString = function() {
	var pad = function(x) {
		return x < 10 ? '0' + x : x
	}

	return pad(this.getHours()) +
		':' + pad(this.getMinutes())
}

Date.prototype.toLocalYMDString = function() {
	var pad = function(x) {
		return x < 10 ? '0' + x : x
	}

	return this.getFullYear() +
		'-' + pad(this.getMonth()) +
		'-' + pad(this.getDate())
}

window.addEventListener('load', function() {
	var now = new Date()
	var today = new Date(now.getFullYear(), now.getMonth(), now.getDate())

	document.querySelector('input[name="timezone-offset"]').value = -new Date().getTimezoneOffset()/60

	document.querySelectorAll('input').forEach(function(input) {
		var check = function() {
			if (this.value === "") {
				this.classList.add('empty')
				this.classList.remove('not-empty')
			} else {
				this.classList.add('not-empty')
				this.classList.remove('empty')
			}
		}

		input.addEventListener('blur', check)
		input.addEventListener('keyup', check)
		check.call(input)
	})

	document.querySelectorAll('input[type="date"]').forEach(function(input) {
		var dataDefault = input.getAttribute('data-default') || ''

		if (dataDefault === 'today') {
			input.value = today.toLocalYMDString()
		}
	})

	document.querySelectorAll('input[type="time"]').forEach(function(input) {
		var dataDefault = input.getAttribute('data-default') || ''

		if (dataDefault === 'today') {
			input.value = today.toLocalTimeString()
		}
	})

	document.querySelectorAll('fieldset.collapsed legend a').forEach(function(a) {
		a.onclick = function() {
			var parent = null
			for (parent = this.parentNode; parent != null; parent = parent.parentNode) {
				if (parent.nodeName === 'FIELDSET') {
					parent.querySelector('div.fields').style.display = 'block'
				}
			}

			this.remove()
			return false
		}
	})

	document.querySelectorAll('input[type="password"]').forEach(function(input) {
		var name = input.getAttribute("id")
		var passwordPrimary
		var passwordConfirm
		if (name.indexOf("confirm-") === 0) {
			passwordPrimary = document.getElementById(name.substr(8))
			passwordConfirm = input
		} else {
			passwordPrimary = input
			passwordConfirm = document.getElementById('confirm-' + name)
		}

		if (!passwordPrimary || !passwordConfirm) {
			return
		}

		input.onblur = input.onkeyup = function() {
			if (passwordPrimary.value !== passwordConfirm.value) {
				passwordPrimary.setCustomValidity("passwords do not match")
				passwordPrimary.classList.add('passwords-no-match')
				passwordConfirm.setCustomValidity("passwords do not match")
				passwordConfirm.classList.add('passwords-no-match')
			} else {
				passwordPrimary.setCustomValidity("")
				passwordPrimary.classList.remove('passwords-no-match')
				passwordConfirm.setCustomValidity("")
				passwordConfirm.classList.remove('passwords-no-match')
			}
		}
	})
})
