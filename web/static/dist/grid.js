/*!
 * Copyright 2019 Tom Peters
 * 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * 
 *    http://www.apache.org/licenses/LICENSE-2.0
 * 
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * 
 */
/******/ (function(modules) { // webpackBootstrap
/******/ 	// The module cache
/******/ 	var installedModules = {};
/******/
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/
/******/ 		// Check if module is in cache
/******/ 		if(installedModules[moduleId]) {
/******/ 			return installedModules[moduleId].exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = installedModules[moduleId] = {
/******/ 			i: moduleId,
/******/ 			l: false,
/******/ 			exports: {}
/******/ 		};
/******/
/******/ 		// Execute the module function
/******/ 		modules[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/
/******/ 		// Flag the module as loaded
/******/ 		module.l = true;
/******/
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/
/******/
/******/ 	// expose the modules object (__webpack_modules__)
/******/ 	__webpack_require__.m = modules;
/******/
/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = installedModules;
/******/
/******/ 	// define getter function for harmony exports
/******/ 	__webpack_require__.d = function(exports, name, getter) {
/******/ 		if(!__webpack_require__.o(exports, name)) {
/******/ 			Object.defineProperty(exports, name, { enumerable: true, get: getter });
/******/ 		}
/******/ 	};
/******/
/******/ 	// define __esModule on exports
/******/ 	__webpack_require__.r = function(exports) {
/******/ 		if(typeof Symbol !== 'undefined' && Symbol.toStringTag) {
/******/ 			Object.defineProperty(exports, Symbol.toStringTag, { value: 'Module' });
/******/ 		}
/******/ 		Object.defineProperty(exports, '__esModule', { value: true });
/******/ 	};
/******/
/******/ 	// create a fake namespace object
/******/ 	// mode & 1: value is a module id, require it
/******/ 	// mode & 2: merge all properties of value into the ns
/******/ 	// mode & 4: return value when already ns object
/******/ 	// mode & 8|1: behave like require
/******/ 	__webpack_require__.t = function(value, mode) {
/******/ 		if(mode & 1) value = __webpack_require__(value);
/******/ 		if(mode & 8) return value;
/******/ 		if((mode & 4) && typeof value === 'object' && value && value.__esModule) return value;
/******/ 		var ns = Object.create(null);
/******/ 		__webpack_require__.r(ns);
/******/ 		Object.defineProperty(ns, 'default', { enumerable: true, value: value });
/******/ 		if(mode & 2 && typeof value != 'string') for(var key in value) __webpack_require__.d(ns, key, function(key) { return value[key]; }.bind(null, key));
/******/ 		return ns;
/******/ 	};
/******/
/******/ 	// getDefaultExport function for compatibility with non-harmony modules
/******/ 	__webpack_require__.n = function(module) {
/******/ 		var getter = module && module.__esModule ?
/******/ 			function getDefault() { return module['default']; } :
/******/ 			function getModuleExports() { return module; };
/******/ 		__webpack_require__.d(getter, 'a', getter);
/******/ 		return getter;
/******/ 	};
/******/
/******/ 	// Object.prototype.hasOwnProperty.call
/******/ 	__webpack_require__.o = function(object, property) { return Object.prototype.hasOwnProperty.call(object, property); };
/******/
/******/ 	// __webpack_public_path__
/******/ 	__webpack_require__.p = "";
/******/
/******/
/******/ 	// Load entry module and return exports
/******/ 	return __webpack_require__(__webpack_require__.s = "./src/grid.js");
/******/ })
/************************************************************************/
/******/ ({

/***/ "./src/datetime.js":
/*!*************************!*\
  !*** ./src/datetime.js ***!
  \*************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, \"default\", function() { return FormatDateTimes; });\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nfunction FormatDateTimes(node) {\n  node.querySelectorAll('*[data-datetime]').forEach(function (node) {\n    const datetimeStr = node.getAttribute(\"data-datetime\");\n    node.removeAttribute(\"data-datetime\");\n    const datetime = new Date(datetimeStr);\n\n    if (isNaN(datetime.getTime())) {\n      return;\n    }\n\n    node.textContent = datetime.toLocaleDateString('default', {\n      year: '2-digit',\n      month: 'numeric',\n      day: 'numeric',\n      hour: 'numeric',\n      minute: 'numeric'\n    });\n  });\n}\n\n//# sourceURL=webpack:///./src/datetime.js?");

/***/ }),

/***/ "./src/grid.js":
/*!*********************!*\
  !*** ./src/grid.js ***!
  \*********************/
/*! no exports provided */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony import */ var _datetime__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./datetime */ \"./src/datetime.js\");\n/* harmony import */ var _modal__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ./modal */ \"./src/modal.js\");\n/* harmony import */ var _loading__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ./loading */ \"./src/loading.js\");\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\n\n\n\nSqMGR.Config = {\n  Types: {\n    'std100': 100,\n    'std25': 25\n  }\n};\n\nSqMGR.buildSquares = function () {\n  new SqMGR.GridBuilder(SqMGR.gridConfig);\n};\n\nSqMGR.GridBuilder = function (config) {\n  this.jwt = config.jwt;\n  this.grid = config.grid;\n  this.pool = config.pool;\n  this.isAdmin = config.isAdmin;\n  this.opaqueUserID = config.opaqueUserID;\n  this.gridSquareStates = config.gridSquareStates;\n  this.modal = new _modal__WEBPACK_IMPORTED_MODULE_1__[\"default\"]();\n  this.templates = document.querySelector('section.templates');\n  this.templates.remove();\n  this.draw(null);\n  this.loadSquares();\n};\n\nSqMGR.GridBuilder.prototype.draw = function (squares) {\n  // XXX refactor this!!!\n  let container = document.getElementById('grid-container'),\n      parent = document.createElement('div'),\n      i,\n      elem,\n      elem2,\n      numSquares,\n      square;\n  parent.classList.add('squares');\n  parent.classList.add(this.pool.gridType);\n  elem = document.createElement('div');\n  elem.classList.add('spacer');\n  parent.appendChild(elem);\n  [\"Home\", \"Away\"].forEach(function (team) {\n    elem = document.createElement('div');\n    elem.classList.add('team');\n    elem.classList.add(team.toLowerCase() + '-team');\n    elem.style.setProperty('--team-primary', this.getTeamValue(team, \"Color1\"));\n    elem.style.setProperty('--team-secondary', this.getTeamValue(team, \"Color2\"));\n    elem.style.setProperty('--team-tertiary', this.getTeamValue(team, \"Color3\"));\n    elem2 = document.createElement('span');\n    elem2.textContent = this.getTeamValue(team, \"Name\");\n    elem.appendChild(elem2);\n    parent.appendChild(elem);\n\n    for (i = 0; i < 10; i++) {\n      elem = document.createElement('div');\n      elem.classList.add('score');\n      elem.classList.add(team.toLowerCase() + '-score');\n      elem.classList.add(team.toLowerCase() + '-score-' + i);\n      elem2 = document.createElement('span'); // FIXME: will need to figure out how to handle scores\n\n      elem2.textContent = '';\n      elem.appendChild(elem2);\n      parent.appendChild(elem);\n    }\n  }.bind(this));\n  numSquares = SqMGR.Config.Types[this.pool.gridType];\n\n  for (i = 1; i <= numSquares; i++) {\n    square = squares ? squares[i] : null;\n    const squareDiv = document.createElement('div');\n    squareDiv.onclick = this.showSquareDetails.bind(this, i);\n    squareDiv.classList.add('square');\n\n    if (square) {\n      squareDiv.classList.add(square.state);\n    }\n\n    squareDiv.setAttribute('data-sqid', i); // add the square id\n\n    const squareIDSpan = document.createElement('span');\n    squareIDSpan.textContent = i;\n    squareIDSpan.classList.add('square-id');\n    squareDiv.appendChild(squareIDSpan); // add the name\n\n    const nameSpan = document.createElement('span');\n    nameSpan.classList.add('name');\n    squareDiv.appendChild(nameSpan);\n\n    if (square) {\n      nameSpan.textContent = square.claimant;\n\n      if (square.opaqueUserID === this.opaqueUserID) {\n        const ownedSpan = document.createElement('span');\n        ownedSpan.classList.add('owned');\n        squareDiv.appendChild(ownedSpan);\n      }\n    }\n\n    parent.appendChild(squareDiv);\n  }\n\n  container.innerHTML = '';\n  container.appendChild(parent);\n};\n\nSqMGR.GridBuilder.prototype.loadSquares = function () {\n  this.get(\"/api/pool/\" + this.pool.token + \"/squares\", function (data) {\n    this.draw(data);\n  }.bind(this));\n  this.loadLogs();\n};\n\nSqMGR.GridBuilder.prototype.loadLogs = function () {\n  if (!this.isAdmin) {\n    return;\n  }\n\n  this.get(\"/api/pool/\" + this.pool.token + \"/logs\", function (data) {\n    let section;\n    const auditLog = this.templates.querySelector('section.audit-log').cloneNode(true);\n    const gridMetadata = document.querySelector('div.grid-metadata');\n    auditLog.querySelector('p.add-note').remove(); // not needed for all logs\n\n    this.buildLogs(auditLog, data);\n\n    if (section = gridMetadata.querySelector('section.audit-log')) {\n      section.replaceWith(auditLog);\n    } else {\n      gridMetadata.appendChild(auditLog);\n    }\n\n    Object(_datetime__WEBPACK_IMPORTED_MODULE_0__[\"default\"])(auditLog);\n    document.querySelector('div.grid-metadata').appendChild(auditLog);\n  }.bind(this));\n};\n\nSqMGR.GridBuilder.prototype.getTeamValue = function (team, prop) {\n  const setting = team.toLowerCase() + \"Team\" + prop;\n  return this.grid.settings[setting];\n};\n\nSqMGR.GridBuilder.prototype.showSquareDetails = function (squareID) {\n  const path = \"/api/pool/\" + this.pool.token + \"/squares/\" + squareID;\n\n  const drawDetails = function (data) {\n    const squareDetails = this.templates.querySelector('div.square-details').cloneNode(true);\n\n    if (data.state === 'unclaimed' || !this.isAdmin) {\n      squareDetails.querySelector('td.state').textContent = data.state;\n    } else {\n      const select = document.createElement('select');\n      let option;\n      this.gridSquareStates.forEach(function (state) {\n        option = document.createElement('option');\n        option.value = state;\n        option.textContent = state;\n\n        if (state === data.state) {\n          option.setAttribute('selected', 'selected');\n        }\n\n        select.appendChild(option);\n      });\n\n      select.onchange = function () {\n        this.promptAndSubmitSquareData(squareID, {\n          state: select.value\n        });\n      }.bind(this);\n\n      squareDetails.querySelector('td.state').appendChild(select);\n    }\n\n    squareDetails.classList.add(data.state);\n    squareDetails.querySelector('td.square-id').textContent = '#' + data.squareID;\n    squareDetails.querySelector('td.claimant').textContent = data.claimant;\n    squareDetails.querySelector('td.modified').setAttribute('data-datetime', data.modified);\n    const claimP = squareDetails.querySelector('p.claim');\n\n    if (data.state !== 'unclaimed' || !this.isAdmin && this.pool.isLocked) {\n      claimP.remove();\n    } else {\n      claimP.querySelector('a').onclick = function () {\n        this.claimSquare(squareID);\n        return false;\n      }.bind(this);\n    }\n\n    const unclaimP = squareDetails.querySelector('p.unclaim');\n\n    if (!this.isAdmin && this.pool.isLocked || data.state !== 'claimed' || data.opaqueUserID !== this.opaqueUserID) {\n      unclaimP.remove();\n    } else {\n      unclaimP.querySelector('a').onclick = function () {\n        this.unclaimSquare(squareID);\n        return false;\n      }.bind(this);\n    }\n\n    const auditLog = squareDetails.querySelector('section.audit-log');\n\n    if (data.logs) {\n      this.buildLogs(auditLog, data.logs, squareID);\n    } // auditLog is only available for admins\n\n\n    if (auditLog) {\n      const addNote = auditLog.querySelector('a.add-note');\n\n      if (addNote) {\n        addNote.onclick = function () {\n          this.promptAndSubmitSquareData(squareID);\n          return false;\n        }.bind(this);\n      }\n    }\n\n    Object(_datetime__WEBPACK_IMPORTED_MODULE_0__[\"default\"])(squareDetails);\n    this.modal.show(squareDetails).addEventListener('modalclose', function () {\n      this.loadSquares();\n    }.bind(this));\n  }.bind(this);\n\n  this.get(path, drawDetails);\n};\n\nSqMGR.GridBuilder.prototype.buildLogs = function (auditLog, logs, squareID) {\n  const auditLogTbody = auditLog.querySelector('tbody');\n  const auditLogRowTpl = auditLog.querySelector('tr.template');\n  auditLogRowTpl.remove();\n  logs.forEach(function (log) {\n    const row = auditLogRowTpl.cloneNode(true);\n    row.querySelector('td.square-id').textContent = '#' + log.squareID;\n    row.querySelector('td.created').setAttribute('data-datetime', log.created);\n    row.querySelector('td.state').textContent = log.state;\n    row.querySelector('td.claimant').textContent = log.claimant;\n    row.querySelector('td.remote-addr').textContent = log.remoteAddr;\n    row.querySelector('td.note').textContent = log.note;\n    auditLogTbody.appendChild(row);\n  }.bind(this));\n};\n\nSqMGR.GridBuilder.prototype.promptAndSubmitSquareData = function (squareID, options) {\n  const form = this.templates.querySelector('form.add-note').cloneNode(true),\n        modal = this.modal.nest();\n\n  form.querySelector('a.cancel').onclick = function () {\n    modal.close();\n    return false;\n  };\n\n  form.onsubmit = function () {\n    const path = \"/api/pool/\" + this.pool.token + \"/squares/\" + squareID;\n    const body = JSON.stringify(Object.assign({\n      note: note.value\n    }, options));\n\n    const success = function (data) {\n      modal.close();\n    }.bind(this);\n\n    const error = function (data) {\n      modal.nest().showError(data.error);\n    }.bind(this);\n\n    this.request(\"POST\", path, body, success, error);\n    return false;\n  }.bind(this);\n\n  modal.show(form).addEventListener('modalclose', function () {\n    this.showSquareDetails(squareID);\n  }.bind(this));\n  form.querySelector('input').select();\n};\n\nSqMGR.GridBuilder.prototype.unclaimSquare = function (squareID) {\n  const path = \"/api/pool/\" + this.pool.token + \"/squares/\" + squareID;\n  const body = JSON.stringify({\n    \"unclaim\": true\n  });\n\n  const success = function () {\n    this.modal.close();\n  }.bind(this);\n\n  const failure = function (data) {\n    this.modal.nest().showError(data.error);\n  }.bind(this);\n\n  this.request(\"POST\", path, body, success, failure);\n};\n\nSqMGR.GridBuilder.prototype.claimSquare = function (squareID) {\n  const modal = this.modal.nest(),\n        form = this.templates.querySelector('form.claim-square').cloneNode(true),\n        input = form.querySelector('input'),\n        storageKey = \"name\";\n\n  if (window.localStorage) {\n    const name = localStorage.getItem(storageKey);\n\n    if (name) {\n      input.value = name;\n    }\n  }\n\n  form.onsubmit = function () {\n    if (input.value === '') {\n      return false;\n    }\n\n    if (window.localStorage) {\n      localStorage.setItem(storageKey, input.value);\n    }\n\n    const path = \"/api/pool/\" + this.pool.token + \"/squares/\" + squareID;\n    const body = JSON.stringify({\n      \"claimant\": input.value\n    });\n\n    const success = function (data) {\n      modal.close();\n    }.bind(this);\n\n    const failure = function (data) {\n      modal.nest().showError(data.error);\n    }.bind(this);\n\n    this.request(\"POST\", path, body, success, failure);\n    return false;\n  }.bind(this);\n\n  modal.show(form).addEventListener('modalclose', function () {\n    this.showSquareDetails(squareID);\n  }.bind(this));\n  input.select();\n};\n\nSqMGR.GridBuilder.prototype.get = function (path, callback, errorCallback) {\n  this.request(\"GET\", path, null, callback, errorCallback);\n};\n\nSqMGR.GridBuilder.prototype.refreshJWT = function (retryFunc) {\n  console.log('refreshing JWT');\n  const xhr = new XMLHttpRequest();\n  xhr.open(\"GET\", \"/pool/\" + this.pool.token + \"/jwt\");\n\n  xhr.onload = function () {\n    let data = null;\n\n    try {\n      data = JSON.parse(xhr.response);\n    } catch (err) {\n      console.log(\"could not parse JSON\", err);\n    }\n\n    if (data.status === \"OK\" && data.result) {\n      this.jwt = data.result;\n      retryFunc();\n      return;\n    }\n\n    throw new Error('could not refresh JWT');\n  }.bind(this);\n\n  xhr.send();\n};\n\nSqMGR.GridBuilder.prototype.request = function (method, path, body, callback, errorCallback, _attempt = 0) {\n  const xhr = new XMLHttpRequest();\n  xhr.open(method, path);\n\n  xhr.onloadend = function () {\n    _loading__WEBPACK_IMPORTED_MODULE_2__[\"default\"].hide();\n  };\n\n  xhr.onload = function () {\n    let data;\n\n    try {\n      if (xhr.status === 401) {\n        if (_attempt > 0) {\n          throw new Error(\"unauthorized\");\n        }\n\n        this.refreshJWT(function () {\n          this.request(method, path, body, callback, errorCallback, _attempt + 1);\n        }.bind(this));\n        return;\n      }\n\n      data = JSON.parse(xhr.response);\n    } catch (err) {\n      console.log(\"could not parse JSON\", err);\n      return;\n    }\n\n    if (data.status === \"OK\") {\n      callback(data.result);\n    } else if (typeof errorCallback === \"function\") {\n      errorCallback(data);\n    }\n  }.bind(this);\n\n  xhr.setRequestHeader(\"Content-Type\", \"application/json\");\n  xhr.setRequestHeader(\"Authorization\", \"Bearer \" + this.jwt);\n  _loading__WEBPACK_IMPORTED_MODULE_2__[\"default\"].show();\n  xhr.send(body);\n};\n\nwindow.addEventListener('load', () => {\n  Object(_datetime__WEBPACK_IMPORTED_MODULE_0__[\"default\"])(document.body);\n  SqMGR.buildSquares();\n});\n\n//# sourceURL=webpack:///./src/grid.js?");

/***/ }),

/***/ "./src/loading.js":
/*!************************!*\
  !*** ./src/loading.js ***!
  \************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nclass Loading {\n  constructor() {\n    this.calls = 0;\n    const div = document.createElement('div');\n    div.classList.add('loading-indicator');\n    div.appendChild(document.createElement('span'));\n    this.loadingDiv = div;\n  }\n\n  show() {\n    if (this.calls++ === 0) {\n      document.body.appendChild(this.loadingDiv);\n    }\n  }\n\n  hide() {\n    if (this.calls <= 0) {\n      throw new Error('hide() called too many times');\n    }\n\n    if (--this.calls === 0) {\n      this.loadingDiv.remove();\n    }\n  }\n\n}\n\n/* harmony default export */ __webpack_exports__[\"default\"] = (new Loading());\n\n//# sourceURL=webpack:///./src/loading.js?");

/***/ }),

/***/ "./src/modal.js":
/*!**********************!*\
  !*** ./src/modal.js ***!
  \**********************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, \"default\", function() { return Modal; });\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nclass Modal {\n  constructor(optionalParent) {\n    this.parent = optionalParent; // the parent modal (optional)\n\n    this.node = null;\n    this.nestedModal = null;\n    this._keyup = this.keyup.bind(this);\n  }\n\n  nest() {\n    if (this.nestedModal) {\n      this.nestedModal.close();\n    }\n\n    this.nestedModal = new Modal(this);\n    return this.nestedModal;\n  }\n\n  nestedDidClose() {\n    this.nestedModal = null;\n  }\n\n  close() {\n    window.removeEventListener('keyup', this._keyup);\n\n    if (this.node) {\n      this.node.dispatchEvent(new Event('modalclose'));\n      this.node.remove();\n      this.node = null;\n    }\n\n    if (this.parent) {\n      this.parent.nestedDidClose();\n    }\n\n    return false;\n  }\n\n  show(childNode) {\n    const node = document.createElement('div');\n    node.classList.add('modal');\n    const closeLink = document.createElement('a');\n    closeLink.setAttribute('href', '#');\n    closeLink.classList.add('close');\n    const closeText = document.createElement('span');\n    closeText.textContent = 'Close';\n    const container = document.createElement('div');\n    container.classList.add('container');\n    const content = document.createElement('div');\n    content.classList.add('container-content');\n    closeLink.appendChild(closeText);\n    container.appendChild(closeLink);\n    content.appendChild(childNode);\n    container.appendChild(content);\n    node.appendChild(container);\n\n    container.onclick = function (event) {\n      event.cancelBubble = true;\n    };\n\n    if (this.node) {\n      this.close();\n    }\n\n    this.node = node;\n    this.node.onclick = closeLink.onclick = this.close.bind(this);\n    document.body.appendChild(node);\n    window.addEventListener('keyup', this._keyup);\n    return node;\n  }\n\n  showError(errorMsg) {\n    const div = document.createElement('div');\n    div.classList.add('error');\n    div.textContent = errorMsg;\n    this.show(div);\n  }\n\n  keyup(event) {\n    if (this.nestedModal) {\n      return;\n    }\n\n    if (event.key === 'Escape') {\n      event.stopPropagation();\n      this.close();\n      return;\n    }\n  }\n\n}\n\n//# sourceURL=webpack:///./src/modal.js?");

/***/ })

/******/ });