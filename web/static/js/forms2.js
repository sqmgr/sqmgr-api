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
})
