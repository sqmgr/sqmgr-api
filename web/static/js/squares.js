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

SqMGR.Config = {
	Types: {
		'std100': 100,
		'std25': 25
	}
}

SqMGR.buildSquares = function() {
	new SqMGR.SquaresBuilder()
}

SqMGR.SquaresBuilder = function() {
	var container = document.getElementById('squares-container'),
		parent = document.createElement('div'),
		i, elem, elem2, numSquares

	// this shouldn't happen
	if (typeof(SqMGR.grid) === "undefined") {
		throw new Error("grid data not found")
	}

	parent.classList.add('squares')
	parent.classList.add(SqMGR.grid.gridType)

	elem = document.createElement('div')
	elem.classList.add('spacer')
	parent.appendChild(elem)

	;["Home", "Away"].forEach(function (team) {
		elem = document.createElement('div')
		elem.classList.add('team')
		elem.classList.add(team.toLowerCase()+ '-team')
		elem.style.setProperty('--team-primary', this.getTeamValue(team, "Color1"))
		elem.style.setProperty('--team-secondary', this.getTeamValue(team, "Color2"))
		elem.style.setProperty('--team-tertiary', this.getTeamValue(team, "Color3"))
		elem2 = document.createElement('span')
		elem2.textContent = this.getTeamValue(team, "Name")
		elem.appendChild(elem2)
		parent.appendChild(elem)

		for (i=0; i<10; i++) {
			elem = document.createElement('div')
			elem.classList.add('score')
			elem.classList.add(team.toLowerCase() + '-score')
			elem.classList.add(team.toLowerCase() + '-score-'+i)
			elem2 = document.createElement('span')
			elem2.textContent = i
			elem.appendChild(elem2)
			parent.appendChild(elem)
		}
	}.bind(this))

	// FIXME
	var names = [ "Alexandria", "Brett", "Charlie", "Danny", "Eliza", "Frank", "Gary", "Harper" ]

	numSquares = SqMGR.Config.Types[SqMGR.grid.gridType]
	for (i=0; i<numSquares; i++) {
		elem = document.createElement('div')
		elem.classList.add('square')
		elem.setAttribute('data-sqid', i)
		elem2 = document.createElement('span')
		elem2.textContent = i+1
		elem2.classList.add('square-id')
		elem.appendChild(elem2)
		elem2 = document.createElement('span')
		elem2.classList.add('name')
		var r = Math.floor(Math.random() * names.length * 2) // FIXME
		var n = r > names.length ? "" : names[r] // FIXME
		elem2.textContent = n
		elem.appendChild(elem2)
		parent.appendChild(elem)
	}

	container.innerHTML = ''
	container.appendChild(parent)
}

SqMGR.SquaresBuilder.prototype.getTeamValue = function(team, prop) {
	var setting = team.toLowerCase() + "Team" + prop
	return SqMGR.grid.settings[setting]
}

window.addEventListener('load', SqMGR.buildSquares)
