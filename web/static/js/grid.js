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

		const squareDiv = document.createElement('div')
		squareDiv.onclick = this.showSquareDetails.bind(this, i)
		squareDiv.classList.add('square')
		if (square) {
            squareDiv.classList.add(square.state)
        }
		squareDiv.setAttribute('data-sqid', i)

		// add the square id
		const squareIDSpan = document.createElement('span')
		squareIDSpan.textContent = i
		squareIDSpan.classList.add('square-id')
		squareDiv.appendChild(squareIDSpan)

		// add the name
		const nameSpan = document.createElement('span')
		nameSpan.classList.add('name')
		squareDiv.appendChild(nameSpan)

		if (square) {
			nameSpan.textContent = square.claimant

			if (square.opaqueUserID === SqMGR.ouid) {
			    const ownedSpan = document.createElement('span')
				ownedSpan.classList.add('owned')
                squareDiv.appendChild(ownedSpan)
			}
		}

		parent.appendChild(squareDiv)
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
    if (!SqMGR.isAdmin) {
    	return
	}

	SqMGR.get("/grid/" + this.grid.token + "/logs", function(data) {
		let section
	    const auditLog = this.templates.querySelector('section.audit-log').cloneNode(true)
		const gridMetadata = document.querySelector('div.grid-metadata')
        auditLog.querySelector('p.add-note').remove() // not needed for all logs

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

SqMGR.GridBuilder.prototype.showSquareDetails = function(squareID) {
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
            	this.promptAndSubmitSquareData(squareID, { state: select.value })
            }.bind(this)

			squareDetails.querySelector('td.state').appendChild(select)
		}

		squareDetails.classList.add(data.state)
		squareDetails.querySelector('td.square-id').textContent = '#' + data.squareID
		squareDetails.querySelector('td.claimant').textContent = data.claimant
		squareDetails.querySelector('td.modified').setAttribute('data-datetime', data.modified)

		const claimP = squareDetails.querySelector('p.claim')
		if (data.state !== 'unclaimed') {
		    claimP.remove()
		} else {
			claimP.querySelector('a').onclick = function() {
				this.claimSquare(squareID)
				return false
			}.bind(this)
		}

		const unclaimP = squareDetails.querySelector('p.unclaim')
		if (data.state === 'claimed' && data.opaqueUserID === SqMGR.ouid) {
			unclaimP.querySelector('a').onclick = function() {
				this.unclaimSquare(squareID)
				return false
			}.bind(this)
		} else {
			unclaimP.remove()
		}

		const auditLog = squareDetails.querySelector('section.audit-log')

		if (data.logs) {
		    this.buildLogs(auditLog, data.logs, squareID)
		}

		SqMGR.DateTime.format(squareDetails)

		this.modal.show(squareDetails).addEventListener('modalclose', function() {
			this.loadSquares()
		}.bind(this))
	}.bind(this)

	SqMGR.get(path, drawDetails)
}

SqMGR.GridBuilder.prototype.buildLogs = function(auditLog, logs, squareID) {
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

	const addNote = auditLog.querySelector('a.add-note')
	if (addNote) {
		addNote.onclick = function() {
			this.promptAndSubmitSquareData(squareID)
			return false
		}.bind(this)
	}
}

SqMGR.GridBuilder.prototype.promptAndSubmitSquareData = function(squareID, options) {
	const form = this.templates.querySelector('form.add-note').cloneNode(true),
		modal = this.modal.nest()

	form.querySelector('a.cancel').onclick = function() {
		modal.close()
		return false
	}

	form.onsubmit = function() {
		const path = "/grid/" + this.grid.token + "/squares/" + squareID
		const body = JSON.stringify(Object.assign({
			note: note.value,
		}, options))
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
        this.showSquareDetails(squareID)
	}.bind(this))

	form.querySelector('input').select()
}

SqMGR.GridBuilder.prototype.unclaimSquare = function(squareID) {
	const path = "/grid/"+this.grid.token+"/squares/"+squareID
	const body = JSON.stringify({"unclaim": true})

	const success = function() {
		this.modal.close()
	}.bind(this)

	const failure = function(data) {
		this.modal.nest().showError(data.error)
	}.bind(this)

	SqMGR.request("POST", path, body, success, failure)
}

SqMGR.GridBuilder.prototype.claimSquare = function(squareID) {
	const modal = this.modal.nest(),
        form = this.templates.querySelector('form.claim-square').cloneNode(true),
		input = form.querySelector('input')

	form.onsubmit = function() {
	    if (input.value === '') {
	    	return	false
		}

	    const path = "/grid/"+this.grid.token+"/squares/"+squareID
		const body = JSON.stringify({"claimant": input.value})

	    const success = function(data) {
	    	modal.close()
		}.bind(this)

		const failure = function(data) {
	    	modal.nest().showError(data.error)
		}.bind(this)

		SqMGR.request("POST", path, body, success, failure)

	    return false
	}.bind(this)

    modal.show(form).addEventListener('modalclose', function() {
    	this.showSquareDetails(squareID)
	}.bind(this))

	input.select()
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
