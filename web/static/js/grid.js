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
	const grid = SqMGR.grid

	new SqMGR.GridBuilder(grid)
}

SqMGR.GridBuilder = function(grid) {
	this.modal = new SqMGR.Modal()
	this.grid = grid
	this.templates = document.querySelector('section.templates')
	this.templates.remove()

	this.draw(null)
	this.loadSquares()
}


SqMGR.GridBuilder.prototype.draw = function(squares) {
	let container = document.getElementById('grid-container'),
		parent = document.createElement('div'),
		i, elem, elem2, numSquares,
		square

	parent.classList.add('squares')
	parent.classList.add(this.grid.gridType)

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
			// FIXME: will need to figure out how to handle scores
			elem2.textContent = ''
			elem.appendChild(elem2)
			parent.appendChild(elem)
		}
	}.bind(this))

	numSquares = SqMGR.Config.Types[this.grid.gridType]
	for (i=0; i<numSquares; i++) {
		square = squares ? squares[i] : null

		elem = document.createElement('div')
		elem.onclick = this.clickSquare.bind(this, i)
		elem.classList.add('square')
		if (square) {
            elem.classList.add(square.state)
        }
		elem.setAttribute('data-sqid', i)

		// add the square id
		elem2 = document.createElement('span')
		elem2.textContent = i
		elem2.classList.add('square-id')
		elem.appendChild(elem2)

		// add the name
		elem2 = document.createElement('span')
		elem2.classList.add('name')

		if (square) {
			elem2.textContent = square.claimant
		}

		elem.appendChild(elem2)
		parent.appendChild(elem)
	}

	container.innerHTML = ''
	container.appendChild(parent)
}

SqMGR.GridBuilder.prototype.loadSquares = function() {
	const container = document.getElementById('grid-container')

	SqMGR.get( "/grid/" + this.grid.token + "/squares", function (data) {
		this.draw(data)
	}.bind(this))

	this.loadLogs()
}

SqMGR.GridBuilder.prototype.loadLogs = function() {
	SqMGR.get("/grid/" + this.grid.token + "/logs", function(data) {
		let section
	    const auditLog = this.templates.querySelector('section.audit-log').cloneNode(true)
		const gridMetadata = document.querySelector('div.grid-metadata')
        this.buildLogs(auditLog, data)

		if (section = gridMetadata.querySelector('section.audit-log')) {
			section.replaceWith(auditLog)
		} else {
			gridMetadata.appendChild(auditLog)
		}

		SqMGR.DateTime.format(auditLog)

		document.querySelector('div.grid-metadata').appendChild(auditLog)
	}.bind(this))
}

SqMGR.GridBuilder.prototype.getTeamValue = function(team, prop) {
	const setting = team.toLowerCase() + "Team" + prop
	return this.grid.settings[setting]
}

SqMGR.GridBuilder.prototype.clickSquare = function(squareID) {
    const path = "/grid/" + this.grid.token + "/squares/" + squareID
    const drawDetails = function(data) {
		const squareDetails = this.templates.querySelector('div.square-details').cloneNode(true)

		if (data.state === 'unclaimed' || !SqMGR.isAdmin) {
			squareDetails.querySelector('td.state').textContent = data.state
		} else {
            const select = document.createElement('select')
            let option
            SqMGR.gridSquareStates.forEach(function (state) {
                option = document.createElement('option')
                option.value = state
                option.textContent = state

                if (state === data.state) {
                    option.setAttribute('selected', 'selected')
                }

                select.appendChild(option)
            })

            select.onchange = function () {
                this.changeSquareState(squareID, select.value)
            }.bind(this)

			squareDetails.querySelector('td.state').appendChild(select)
		}

		squareDetails.classList.add(data.state)
		squareDetails.querySelector('td.square-id').textContent = '#' + data.squareID
		squareDetails.querySelector('td.claimant').textContent = data.claimant
		squareDetails.querySelector('td.modified').setAttribute('data-datetime', data.modified)

		const auditLog = squareDetails.querySelector('section.audit-log')

		if (data.logs) {
		    this.buildLogs(auditLog, data.logs)
		} else {
			auditLog.remove()
		}

		SqMGR.DateTime.format(squareDetails)

		this.modal.show(squareDetails).addEventListener('modalclose', function() {
			this.loadSquares()
		}.bind(this))
	}.bind(this)

	SqMGR.get(path, drawDetails)
}

SqMGR.GridBuilder.prototype.buildLogs = function(auditLog, logs) {
	const auditLogTbody = auditLog.querySelector('tbody')
	const auditLogRowTpl = auditLog.querySelector('tr.template')
	auditLogRowTpl.remove()

	logs.forEach(function (log) {
		const row = auditLogRowTpl.cloneNode(true)
		row.querySelector('td.square-id').textContent = '#' + log.squareID
		row.querySelector('td.created').setAttribute('data-datetime', log.created)
		row.querySelector('td.state').textContent = log.state
		row.querySelector('td.claimant').textContent = log.claimant
		row.querySelector('td.remote-addr').textContent = log.remoteAddr
		row.querySelector('td.note').textContent = log.note

		auditLogTbody.appendChild(row)
	}.bind(this))
}

SqMGR.GridBuilder.prototype.changeSquareState = function(squareID, newState) {
	const form = document.createElement('form'),
		field = document.createElement('div'),
		label = document.createElement('label'),
		note = document.createElement('input'),
        buttons = document.createElement('div'),
		button = document.createElement('input'),
		cancelLink = document.createElement('a'),
		modal = this.modal.nest()

	field.classList.add('field')
	
	label.setAttribute('for', 'note')
	label.textContent = 'Reason for change'
	
	note.id = 'note'
    note.type = 'text'
	note.placeholder = 'Reason for change'
    note.name = 'note'
    
	field.appendChild(label)
	field.appendChild(note)
	
	buttons.classList.add('buttons')
    
	button.type = 'submit'
	button.name = 'submit'
	button.value = 'Save'
    
	cancelLink.setAttribute('href', '#')
    cancelLink.classList.add('cancel')
	cancelLink.textContent = 'Cancel'
	cancelLink.onclick = function() {
		modal.close()
		return false
	}
	
	buttons.appendChild(button)
	buttons.appendChild(cancelLink)
	
	form.appendChild(field)
	form.appendChild(buttons)

	form.onsubmit = function() {
		const path = "/grid/" + this.grid.token + "/squares/" + squareID
		const body = JSON.stringify({
			note: note.value,
			state: newState,
		})
		const success = function(data) {
		    modal.close()
		}.bind(this)
		const error = function(data) {
		    modal.nest().showError(data.error)
		}.bind(this)
	    SqMGR.request("POST", path, body, success, error)
		return false
	}.bind(this)

	modal.show(form).addEventListener('modalclose', function() {
        this.clickSquare(squareID)
	}.bind(this))

	note.select()
}

SqMGR.get = function(path, callback, errorCallback) {
	SqMGR.request("GET", path, null, callback, errorCallback)
}

SqMGR.request = function(method, path, body, callback, errorCallback) {
	const xhr = new XMLHttpRequest()
	xhr.open(method, path)
    xhr.onloadend = function() {
	    SqMGR.Loading.hide()
	}
	xhr.onload = function() {
	    let data
		try {
	   		data = JSON.parse(xhr.response)
		} catch (err) {
	    	console.log("could not parse JSON", err)
			return
		}

		if (data.status === "OK") {
			callback(data.result)
		} else if (typeof(errorCallback) === "function") {
			errorCallback(data)
		}
	}

	xhr.setRequestHeader("Content-Type", "application/json")

	SqMGR.Loading.show()
	xhr.send(body)
}

window.addEventListener('load', SqMGR.buildSquares)
