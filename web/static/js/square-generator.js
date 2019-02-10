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

function Squared(selector) {
	this.container = document.querySelector(selector)
	var frag = document.createDocumentFragment(),
		div, span, i

	div = document.createElement('div')
	div.classList.add('spacer')
	frag.appendChild(div)

	var createTeam = function(label) {
		var div = document.createElement('div'),
			span, i
		div.classList.add('team')
		div.classList.add('team-'+label)
		frag.appendChild(div)

		for (i=0; i<10; i++) {
			div = document.createElement('div')
			div.classList.add('score')
			div.classList.add('score-'+label)
			div.classList.add('score-'+label+'-'+i)

			span = document.createElement('span')
			span.textContent = i
			div.appendChild(span)

			frag.appendChild(div)
		}
	}

	createTeam('away')
	createTeam('home')

	var names = [ "Tom", "Kaitlin", "Ellie", "Teddy", "Gilly", "Django", "Donny", "Stacey", "Frederick von Schmidt" ]
	for (i=0; i<25; i++) {
		div = document.createElement('div')
		div.classList.add('square')
		div.classList.add('square-'+i)
		div.onclick = this.squareClicked.bind(this, div)

		span = div.appendChild(document.createElement('span'))

		// claim paid
		frag.appendChild(div)
	}

	this.container.appendChild(frag)
}

Squared.prototype.squareClicked = function(div) {
	var storedName = Squared.getItem('name')
	console.log(storedName)
	var name = prompt('What is your name?', storedName || '')
	console.log(name)
	if (!name || !name.match(/\w/)) {
		return
	}

	Squared.setItem('name', name)
}

Squared.run = function() {
	new Squared('div.squares')
}

Squared.setItem = function(key, val) {
	try {
		localStorage.setItem(key, val)
	} catch (e) {
		console.log("could not set item: ", e)
	}
}

Squared.getItem = function(key) {
	var item
	try {
		item = localStorage.getItem(key)
		return item
	} catch (e) {
		console.log("could not get item: ", e)
	}

	return null
}

window.addEventListener('load', Squared.run)
