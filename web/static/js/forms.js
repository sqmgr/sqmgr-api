Date.prototype.toLocalDateTimeString = function() {
	var pad = function(x) {
		return x < 10 ? '0' + x : x
	}

	return this.getFullYear() +
		'-' + pad(this.getMonth()) +
		'-' + pad(this.getDate()) +
		'T' + pad(this.getHours()) +
		':' + pad(this.getMinutes()) +
		':' + pad(this.getSeconds())
}

window.addEventListener('load', function() {
	var now = new Date()
	var today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
	var todayValue = today.toLocalDateTimeString()

	document.querySelectorAll('input[type="datetime-local"]').forEach(function(input) {
		var dataDefault = input.getAttribute('data-default') || ''

		if (dataDefault === 'today') {
			input.value = todayValue
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

		input.onblur = function() {
			if (passwordPrimary.value !== passwordConfirm.value) {
				passwordConfirm.setCustomValidity("passwords do not match")
			} else {
				passwordConfirm.setCustomValidity("")
			}
		}
	})
})
