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
/******/ 	return __webpack_require__(__webpack_require__.s = "./src/account-delete.js");
/******/ })
/************************************************************************/
/******/ ({

/***/ "./src/account-delete.js":
/*!*******************************!*\
  !*** ./src/account-delete.js ***!
  \*******************************/
/*! no exports provided */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony import */ var _setup_form__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./setup-form */ \"./src/setup-form.js\");\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\n\nwindow.addEventListener('load', function () {\n  Object(_setup_form__WEBPACK_IMPORTED_MODULE_0__[\"default\"])();\n  const modal = document.querySelector('div.modal'),\n        cover = document.createElement('div'),\n        closeLink = modal.querySelector('a.close'),\n        emailInput = modal.querySelector('input[name=\"email\"]'),\n        expectedEmail = modal.querySelector('input[name=\"expected-email\"]').value,\n        submitButton = modal.querySelector('input[type=\"submit\"]');\n  modal.remove();\n  modal.style.display = 'block';\n  cover.classList.add('cover');\n\n  function closeModal() {\n    window.removeEventListener('keyup', checkForEscape);\n    modal.remove();\n    cover.remove();\n    emailInput.value = '';\n  }\n\n  cover.onclick = closeModal;\n  closeLink.onclick = closeModal;\n\n  function checkForEscape(e) {\n    if (e.key === 'Escape') {\n      closeModal();\n    }\n  }\n\n  document.querySelector('button.destructive').onclick = function () {\n    if (document.querySelector('div.modal')) {\n      return;\n    }\n\n    document.body.appendChild(cover);\n    document.body.appendChild(modal);\n    emailInput.focus();\n    window.addEventListener('keyup', checkForEscape);\n  };\n\n  emailInput.onkeyup = emailInput.onblur = function () {\n    if (this.value === expectedEmail) {\n      submitButton.removeAttribute(\"disabled\");\n    } else {\n      submitButton.setAttribute(\"disabled\", \"disabled\");\n    }\n  };\n});//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvYWNjb3VudC1kZWxldGUuanM/YTE5MCJdLCJuYW1lcyI6WyJ3aW5kb3ciLCJhZGRFdmVudExpc3RlbmVyIiwiU2V0dXBGb3JtIiwibW9kYWwiLCJkb2N1bWVudCIsInF1ZXJ5U2VsZWN0b3IiLCJjb3ZlciIsImNyZWF0ZUVsZW1lbnQiLCJjbG9zZUxpbmsiLCJlbWFpbElucHV0IiwiZXhwZWN0ZWRFbWFpbCIsInZhbHVlIiwic3VibWl0QnV0dG9uIiwicmVtb3ZlIiwic3R5bGUiLCJkaXNwbGF5IiwiY2xhc3NMaXN0IiwiYWRkIiwiY2xvc2VNb2RhbCIsInJlbW92ZUV2ZW50TGlzdGVuZXIiLCJjaGVja0ZvckVzY2FwZSIsIm9uY2xpY2siLCJlIiwia2V5IiwiYm9keSIsImFwcGVuZENoaWxkIiwiZm9jdXMiLCJvbmtleXVwIiwib25ibHVyIiwicmVtb3ZlQXR0cmlidXRlIiwic2V0QXR0cmlidXRlIl0sIm1hcHBpbmdzIjoiQUFBQTtBQUFBO0FBQUE7Ozs7Ozs7Ozs7Ozs7OztBQWdCQTtBQUVBQSxNQUFNLENBQUNDLGdCQUFQLENBQXdCLE1BQXhCLEVBQWdDLFlBQVk7QUFDeENDLDZEQUFTO0FBRVQsUUFBTUMsS0FBSyxHQUFHQyxRQUFRLENBQUNDLGFBQVQsQ0FBdUIsV0FBdkIsQ0FBZDtBQUFBLFFBQ0lDLEtBQUssR0FBR0YsUUFBUSxDQUFDRyxhQUFULENBQXVCLEtBQXZCLENBRFo7QUFBQSxRQUVJQyxTQUFTLEdBQUdMLEtBQUssQ0FBQ0UsYUFBTixDQUFvQixTQUFwQixDQUZoQjtBQUFBLFFBR0lJLFVBQVUsR0FBR04sS0FBSyxDQUFDRSxhQUFOLENBQW9CLHFCQUFwQixDQUhqQjtBQUFBLFFBSUlLLGFBQWEsR0FBR1AsS0FBSyxDQUFDRSxhQUFOLENBQW9CLDhCQUFwQixFQUFvRE0sS0FKeEU7QUFBQSxRQUtJQyxZQUFZLEdBQUdULEtBQUssQ0FBQ0UsYUFBTixDQUFvQixzQkFBcEIsQ0FMbkI7QUFPQUYsT0FBSyxDQUFDVSxNQUFOO0FBQ0FWLE9BQUssQ0FBQ1csS0FBTixDQUFZQyxPQUFaLEdBQXNCLE9BQXRCO0FBRUFULE9BQUssQ0FBQ1UsU0FBTixDQUFnQkMsR0FBaEIsQ0FBb0IsT0FBcEI7O0FBRUEsV0FBU0MsVUFBVCxHQUFzQjtBQUNsQmxCLFVBQU0sQ0FBQ21CLG1CQUFQLENBQTJCLE9BQTNCLEVBQW9DQyxjQUFwQztBQUNBakIsU0FBSyxDQUFDVSxNQUFOO0FBQ0FQLFNBQUssQ0FBQ08sTUFBTjtBQUNBSixjQUFVLENBQUNFLEtBQVgsR0FBbUIsRUFBbkI7QUFDSDs7QUFFREwsT0FBSyxDQUFDZSxPQUFOLEdBQWdCSCxVQUFoQjtBQUNBVixXQUFTLENBQUNhLE9BQVYsR0FBb0JILFVBQXBCOztBQUVBLFdBQVNFLGNBQVQsQ0FBd0JFLENBQXhCLEVBQTJCO0FBQ3ZCLFFBQUlBLENBQUMsQ0FBQ0MsR0FBRixLQUFVLFFBQWQsRUFBd0I7QUFDcEJMLGdCQUFVO0FBQ2I7QUFDSjs7QUFFRGQsVUFBUSxDQUFDQyxhQUFULENBQXVCLG9CQUF2QixFQUE2Q2dCLE9BQTdDLEdBQXVELFlBQVk7QUFDL0QsUUFBSWpCLFFBQVEsQ0FBQ0MsYUFBVCxDQUF1QixXQUF2QixDQUFKLEVBQXlDO0FBQ3JDO0FBQ0g7O0FBRURELFlBQVEsQ0FBQ29CLElBQVQsQ0FBY0MsV0FBZCxDQUEwQm5CLEtBQTFCO0FBQ0FGLFlBQVEsQ0FBQ29CLElBQVQsQ0FBY0MsV0FBZCxDQUEwQnRCLEtBQTFCO0FBQ0FNLGNBQVUsQ0FBQ2lCLEtBQVg7QUFDQTFCLFVBQU0sQ0FBQ0MsZ0JBQVAsQ0FBd0IsT0FBeEIsRUFBaUNtQixjQUFqQztBQUNILEdBVEQ7O0FBV0FYLFlBQVUsQ0FBQ2tCLE9BQVgsR0FBcUJsQixVQUFVLENBQUNtQixNQUFYLEdBQW9CLFlBQVk7QUFDakQsUUFBSSxLQUFLakIsS0FBTCxLQUFlRCxhQUFuQixFQUFrQztBQUM5QkUsa0JBQVksQ0FBQ2lCLGVBQWIsQ0FBNkIsVUFBN0I7QUFDSCxLQUZELE1BRU87QUFDSGpCLGtCQUFZLENBQUNrQixZQUFiLENBQTBCLFVBQTFCLEVBQXNDLFVBQXRDO0FBQ0g7QUFDSixHQU5EO0FBT0gsQ0FqREQiLCJmaWxlIjoiLi9zcmMvYWNjb3VudC1kZWxldGUuanMuanMiLCJzb3VyY2VzQ29udGVudCI6WyIvKlxuQ29weXJpZ2h0IDIwMTkgVG9tIFBldGVyc1xuXG5MaWNlbnNlZCB1bmRlciB0aGUgQXBhY2hlIExpY2Vuc2UsIFZlcnNpb24gMi4wICh0aGUgXCJMaWNlbnNlXCIpO1xueW91IG1heSBub3QgdXNlIHRoaXMgZmlsZSBleGNlcHQgaW4gY29tcGxpYW5jZSB3aXRoIHRoZSBMaWNlbnNlLlxuWW91IG1heSBvYnRhaW4gYSBjb3B5IG9mIHRoZSBMaWNlbnNlIGF0XG5cbiAgIGh0dHA6Ly93d3cuYXBhY2hlLm9yZy9saWNlbnNlcy9MSUNFTlNFLTIuMFxuXG5Vbmxlc3MgcmVxdWlyZWQgYnkgYXBwbGljYWJsZSBsYXcgb3IgYWdyZWVkIHRvIGluIHdyaXRpbmcsIHNvZnR3YXJlXG5kaXN0cmlidXRlZCB1bmRlciB0aGUgTGljZW5zZSBpcyBkaXN0cmlidXRlZCBvbiBhbiBcIkFTIElTXCIgQkFTSVMsXG5XSVRIT1VUIFdBUlJBTlRJRVMgT1IgQ09ORElUSU9OUyBPRiBBTlkgS0lORCwgZWl0aGVyIGV4cHJlc3Mgb3IgaW1wbGllZC5cblNlZSB0aGUgTGljZW5zZSBmb3IgdGhlIHNwZWNpZmljIGxhbmd1YWdlIGdvdmVybmluZyBwZXJtaXNzaW9ucyBhbmRcbmxpbWl0YXRpb25zIHVuZGVyIHRoZSBMaWNlbnNlLlxuKi9cblxuaW1wb3J0IFNldHVwRm9ybSBmcm9tICcuL3NldHVwLWZvcm0nXG5cbndpbmRvdy5hZGRFdmVudExpc3RlbmVyKCdsb2FkJywgZnVuY3Rpb24gKCkge1xuICAgIFNldHVwRm9ybSgpXG5cbiAgICBjb25zdCBtb2RhbCA9IGRvY3VtZW50LnF1ZXJ5U2VsZWN0b3IoJ2Rpdi5tb2RhbCcpLFxuICAgICAgICBjb3ZlciA9IGRvY3VtZW50LmNyZWF0ZUVsZW1lbnQoJ2RpdicpLFxuICAgICAgICBjbG9zZUxpbmsgPSBtb2RhbC5xdWVyeVNlbGVjdG9yKCdhLmNsb3NlJyksXG4gICAgICAgIGVtYWlsSW5wdXQgPSBtb2RhbC5xdWVyeVNlbGVjdG9yKCdpbnB1dFtuYW1lPVwiZW1haWxcIl0nKSxcbiAgICAgICAgZXhwZWN0ZWRFbWFpbCA9IG1vZGFsLnF1ZXJ5U2VsZWN0b3IoJ2lucHV0W25hbWU9XCJleHBlY3RlZC1lbWFpbFwiXScpLnZhbHVlLFxuICAgICAgICBzdWJtaXRCdXR0b24gPSBtb2RhbC5xdWVyeVNlbGVjdG9yKCdpbnB1dFt0eXBlPVwic3VibWl0XCJdJylcblxuICAgIG1vZGFsLnJlbW92ZSgpXG4gICAgbW9kYWwuc3R5bGUuZGlzcGxheSA9ICdibG9jaydcblxuICAgIGNvdmVyLmNsYXNzTGlzdC5hZGQoJ2NvdmVyJylcblxuICAgIGZ1bmN0aW9uIGNsb3NlTW9kYWwoKSB7XG4gICAgICAgIHdpbmRvdy5yZW1vdmVFdmVudExpc3RlbmVyKCdrZXl1cCcsIGNoZWNrRm9yRXNjYXBlKVxuICAgICAgICBtb2RhbC5yZW1vdmUoKVxuICAgICAgICBjb3Zlci5yZW1vdmUoKVxuICAgICAgICBlbWFpbElucHV0LnZhbHVlID0gJydcbiAgICB9XG5cbiAgICBjb3Zlci5vbmNsaWNrID0gY2xvc2VNb2RhbFxuICAgIGNsb3NlTGluay5vbmNsaWNrID0gY2xvc2VNb2RhbFxuXG4gICAgZnVuY3Rpb24gY2hlY2tGb3JFc2NhcGUoZSkge1xuICAgICAgICBpZiAoZS5rZXkgPT09ICdFc2NhcGUnKSB7XG4gICAgICAgICAgICBjbG9zZU1vZGFsKClcbiAgICAgICAgfVxuICAgIH1cblxuICAgIGRvY3VtZW50LnF1ZXJ5U2VsZWN0b3IoJ2J1dHRvbi5kZXN0cnVjdGl2ZScpLm9uY2xpY2sgPSBmdW5jdGlvbiAoKSB7XG4gICAgICAgIGlmIChkb2N1bWVudC5xdWVyeVNlbGVjdG9yKCdkaXYubW9kYWwnKSkge1xuICAgICAgICAgICAgcmV0dXJuXG4gICAgICAgIH1cblxuICAgICAgICBkb2N1bWVudC5ib2R5LmFwcGVuZENoaWxkKGNvdmVyKVxuICAgICAgICBkb2N1bWVudC5ib2R5LmFwcGVuZENoaWxkKG1vZGFsKVxuICAgICAgICBlbWFpbElucHV0LmZvY3VzKClcbiAgICAgICAgd2luZG93LmFkZEV2ZW50TGlzdGVuZXIoJ2tleXVwJywgY2hlY2tGb3JFc2NhcGUpXG4gICAgfVxuXG4gICAgZW1haWxJbnB1dC5vbmtleXVwID0gZW1haWxJbnB1dC5vbmJsdXIgPSBmdW5jdGlvbiAoKSB7XG4gICAgICAgIGlmICh0aGlzLnZhbHVlID09PSBleHBlY3RlZEVtYWlsKSB7XG4gICAgICAgICAgICBzdWJtaXRCdXR0b24ucmVtb3ZlQXR0cmlidXRlKFwiZGlzYWJsZWRcIilcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICAgIHN1Ym1pdEJ1dHRvbi5zZXRBdHRyaWJ1dGUoXCJkaXNhYmxlZFwiLCBcImRpc2FibGVkXCIpXG4gICAgICAgIH1cbiAgICB9XG59KVxuIl0sInNvdXJjZVJvb3QiOiIifQ==\n//# sourceURL=webpack-internal:///./src/account-delete.js\n");

/***/ }),

/***/ "./src/setup-form.js":
/*!***************************!*\
  !*** ./src/setup-form.js ***!
  \***************************/
/*! exports provided: setupPasswordInput, setupTimeInput, default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, \"setupPasswordInput\", function() { return setupPasswordInput; });\n/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, \"setupTimeInput\", function() { return setupTimeInput; });\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nfunction setupPasswordInput() {\n  const buffer = 1; // spacing for error message\n\n  document.querySelectorAll('input[type=\"password\"]').forEach(function (input) {\n    const id = input.getAttribute('id');\n    let confirmInput, checkPasswordFn, noMatchElem;\n\n    if (id.indexOf('confirm-') === 0) {\n      // if a confirm- is present, that means that we are expecting user to input\n      // a brand-new password. Do not let Firefox auto fill this\n      document.getElementById(id.substr('confirm-'.length)).value = '';\n      input.value = '';\n      return;\n    }\n\n    confirmInput = document.getElementById('confirm-' + id);\n\n    if (!confirmInput) {\n      return;\n    }\n\n    checkPasswordFn = function () {\n      if (input.value === confirmInput.value) {\n        if (noMatchElem) {\n          noMatchElem.remove();\n          noMatchElem = null;\n        }\n\n        confirmInput.setCustomValidity(\"\");\n        return;\n      }\n\n      confirmInput.setCustomValidity(\"Passwords do not match\");\n\n      if (noMatchElem) {\n        return;\n      }\n\n      const clientRect = confirmInput.getBoundingClientRect();\n      noMatchElem = document.createElement('div');\n      noMatchElem.textContent = 'The passwords do not match';\n      noMatchElem.style.left = clientRect.left + 'px';\n      noMatchElem.style.top = clientRect.top + clientRect.height + buffer + 'px';\n      noMatchElem.classList.add('input-error');\n      document.body.appendChild(noMatchElem);\n    };\n\n    input.addEventListener('keyup', checkPasswordFn);\n    confirmInput.addEventListener('keyup', checkPasswordFn);\n  });\n}\nfunction setupTimeInput() {\n  document.querySelectorAll('input[type=\"time\"]').forEach(function (input) {\n    if (input.value === '') {\n      input.value = '00:00';\n    }\n  });\n}\n/* harmony default export */ __webpack_exports__[\"default\"] = (function () {\n  setupPasswordInput();\n  setupTimeInput();\n});//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvc2V0dXAtZm9ybS5qcz8yY2Y3Il0sIm5hbWVzIjpbInNldHVwUGFzc3dvcmRJbnB1dCIsImJ1ZmZlciIsImRvY3VtZW50IiwicXVlcnlTZWxlY3RvckFsbCIsImZvckVhY2giLCJpbnB1dCIsImlkIiwiZ2V0QXR0cmlidXRlIiwiY29uZmlybUlucHV0IiwiY2hlY2tQYXNzd29yZEZuIiwibm9NYXRjaEVsZW0iLCJpbmRleE9mIiwiZ2V0RWxlbWVudEJ5SWQiLCJzdWJzdHIiLCJsZW5ndGgiLCJ2YWx1ZSIsInJlbW92ZSIsInNldEN1c3RvbVZhbGlkaXR5IiwiY2xpZW50UmVjdCIsImdldEJvdW5kaW5nQ2xpZW50UmVjdCIsImNyZWF0ZUVsZW1lbnQiLCJ0ZXh0Q29udGVudCIsInN0eWxlIiwibGVmdCIsInRvcCIsImhlaWdodCIsImNsYXNzTGlzdCIsImFkZCIsImJvZHkiLCJhcHBlbmRDaGlsZCIsImFkZEV2ZW50TGlzdGVuZXIiLCJzZXR1cFRpbWVJbnB1dCJdLCJtYXBwaW5ncyI6IkFBQUE7QUFBQTtBQUFBO0FBQUE7Ozs7Ozs7Ozs7Ozs7OztBQWdCTyxTQUFTQSxrQkFBVCxHQUE4QjtBQUNwQyxRQUFNQyxNQUFNLEdBQUcsQ0FBZixDQURvQyxDQUNuQjs7QUFFakJDLFVBQVEsQ0FBQ0MsZ0JBQVQsQ0FBMEIsd0JBQTFCLEVBQW9EQyxPQUFwRCxDQUE0RCxVQUFTQyxLQUFULEVBQWdCO0FBQzNFLFVBQU1DLEVBQUUsR0FBR0QsS0FBSyxDQUFDRSxZQUFOLENBQW1CLElBQW5CLENBQVg7QUFDQSxRQUFJQyxZQUFKLEVBQWtCQyxlQUFsQixFQUFtQ0MsV0FBbkM7O0FBRUEsUUFBSUosRUFBRSxDQUFDSyxPQUFILENBQVcsVUFBWCxNQUEyQixDQUEvQixFQUFrQztBQUNqQztBQUNBO0FBQ0FULGNBQVEsQ0FBQ1UsY0FBVCxDQUF3Qk4sRUFBRSxDQUFDTyxNQUFILENBQVUsV0FBV0MsTUFBckIsQ0FBeEIsRUFBc0RDLEtBQXRELEdBQThELEVBQTlEO0FBQ0FWLFdBQUssQ0FBQ1UsS0FBTixHQUFjLEVBQWQ7QUFDQTtBQUNBOztBQUVEUCxnQkFBWSxHQUFHTixRQUFRLENBQUNVLGNBQVQsQ0FBd0IsYUFBV04sRUFBbkMsQ0FBZjs7QUFDQSxRQUFJLENBQUNFLFlBQUwsRUFBbUI7QUFDbEI7QUFDQTs7QUFFREMsbUJBQWUsR0FBRyxZQUFXO0FBQzVCLFVBQUlKLEtBQUssQ0FBQ1UsS0FBTixLQUFnQlAsWUFBWSxDQUFDTyxLQUFqQyxFQUF3QztBQUN2QyxZQUFJTCxXQUFKLEVBQWlCO0FBQ2hCQSxxQkFBVyxDQUFDTSxNQUFaO0FBQ0FOLHFCQUFXLEdBQUcsSUFBZDtBQUNBOztBQUVERixvQkFBWSxDQUFDUyxpQkFBYixDQUErQixFQUEvQjtBQUNBO0FBQ0E7O0FBRURULGtCQUFZLENBQUNTLGlCQUFiLENBQStCLHdCQUEvQjs7QUFFQSxVQUFJUCxXQUFKLEVBQWlCO0FBQ2hCO0FBQ0E7O0FBRUQsWUFBTVEsVUFBVSxHQUFHVixZQUFZLENBQUNXLHFCQUFiLEVBQW5CO0FBRUFULGlCQUFXLEdBQUdSLFFBQVEsQ0FBQ2tCLGFBQVQsQ0FBdUIsS0FBdkIsQ0FBZDtBQUNBVixpQkFBVyxDQUFDVyxXQUFaLEdBQTBCLDRCQUExQjtBQUNBWCxpQkFBVyxDQUFDWSxLQUFaLENBQWtCQyxJQUFsQixHQUF5QkwsVUFBVSxDQUFDSyxJQUFYLEdBQWdCLElBQXpDO0FBQ0FiLGlCQUFXLENBQUNZLEtBQVosQ0FBa0JFLEdBQWxCLEdBQXdCTixVQUFVLENBQUNNLEdBQVgsR0FBZU4sVUFBVSxDQUFDTyxNQUExQixHQUFpQ3hCLE1BQWpDLEdBQXdDLElBQWhFO0FBQ0FTLGlCQUFXLENBQUNnQixTQUFaLENBQXNCQyxHQUF0QixDQUEwQixhQUExQjtBQUNBekIsY0FBUSxDQUFDMEIsSUFBVCxDQUFjQyxXQUFkLENBQTBCbkIsV0FBMUI7QUFDQSxLQXpCRDs7QUEyQkFMLFNBQUssQ0FBQ3lCLGdCQUFOLENBQXVCLE9BQXZCLEVBQWdDckIsZUFBaEM7QUFDQUQsZ0JBQVksQ0FBQ3NCLGdCQUFiLENBQThCLE9BQTlCLEVBQXVDckIsZUFBdkM7QUFDQSxHQTlDRDtBQStDQTtBQUVNLFNBQVNzQixjQUFULEdBQTBCO0FBQ2hDN0IsVUFBUSxDQUFDQyxnQkFBVCxDQUEwQixvQkFBMUIsRUFBZ0RDLE9BQWhELENBQXdELFVBQVNDLEtBQVQsRUFBZ0I7QUFDdkUsUUFBSUEsS0FBSyxDQUFDVSxLQUFOLEtBQWdCLEVBQXBCLEVBQXdCO0FBQ3ZCVixXQUFLLENBQUNVLEtBQU4sR0FBYyxPQUFkO0FBQ0E7QUFDRCxHQUpEO0FBS0E7QUFFYywyRUFBVztBQUN0QmYsb0JBQWtCO0FBQ3JCK0IsZ0JBQWM7QUFDZCxDIiwiZmlsZSI6Ii4vc3JjL3NldHVwLWZvcm0uanMuanMiLCJzb3VyY2VzQ29udGVudCI6WyIvKlxuQ29weXJpZ2h0IDIwMTkgVG9tIFBldGVyc1xuXG5MaWNlbnNlZCB1bmRlciB0aGUgQXBhY2hlIExpY2Vuc2UsIFZlcnNpb24gMi4wICh0aGUgXCJMaWNlbnNlXCIpO1xueW91IG1heSBub3QgdXNlIHRoaXMgZmlsZSBleGNlcHQgaW4gY29tcGxpYW5jZSB3aXRoIHRoZSBMaWNlbnNlLlxuWW91IG1heSBvYnRhaW4gYSBjb3B5IG9mIHRoZSBMaWNlbnNlIGF0XG5cbiAgIGh0dHA6Ly93d3cuYXBhY2hlLm9yZy9saWNlbnNlcy9MSUNFTlNFLTIuMFxuXG5Vbmxlc3MgcmVxdWlyZWQgYnkgYXBwbGljYWJsZSBsYXcgb3IgYWdyZWVkIHRvIGluIHdyaXRpbmcsIHNvZnR3YXJlXG5kaXN0cmlidXRlZCB1bmRlciB0aGUgTGljZW5zZSBpcyBkaXN0cmlidXRlZCBvbiBhbiBcIkFTIElTXCIgQkFTSVMsXG5XSVRIT1VUIFdBUlJBTlRJRVMgT1IgQ09ORElUSU9OUyBPRiBBTlkgS0lORCwgZWl0aGVyIGV4cHJlc3Mgb3IgaW1wbGllZC5cblNlZSB0aGUgTGljZW5zZSBmb3IgdGhlIHNwZWNpZmljIGxhbmd1YWdlIGdvdmVybmluZyBwZXJtaXNzaW9ucyBhbmRcbmxpbWl0YXRpb25zIHVuZGVyIHRoZSBMaWNlbnNlLlxuKi9cblxuZXhwb3J0IGZ1bmN0aW9uIHNldHVwUGFzc3dvcmRJbnB1dCgpIHtcblx0Y29uc3QgYnVmZmVyID0gMSAvLyBzcGFjaW5nIGZvciBlcnJvciBtZXNzYWdlXG5cblx0ZG9jdW1lbnQucXVlcnlTZWxlY3RvckFsbCgnaW5wdXRbdHlwZT1cInBhc3N3b3JkXCJdJykuZm9yRWFjaChmdW5jdGlvbihpbnB1dCkge1xuXHRcdGNvbnN0IGlkID0gaW5wdXQuZ2V0QXR0cmlidXRlKCdpZCcpO1xuXHRcdGxldCBjb25maXJtSW5wdXQsIGNoZWNrUGFzc3dvcmRGbiwgbm9NYXRjaEVsZW1cblxuXHRcdGlmIChpZC5pbmRleE9mKCdjb25maXJtLScpID09PSAwKSB7XG5cdFx0XHQvLyBpZiBhIGNvbmZpcm0tIGlzIHByZXNlbnQsIHRoYXQgbWVhbnMgdGhhdCB3ZSBhcmUgZXhwZWN0aW5nIHVzZXIgdG8gaW5wdXRcblx0XHRcdC8vIGEgYnJhbmQtbmV3IHBhc3N3b3JkLiBEbyBub3QgbGV0IEZpcmVmb3ggYXV0byBmaWxsIHRoaXNcblx0XHRcdGRvY3VtZW50LmdldEVsZW1lbnRCeUlkKGlkLnN1YnN0cignY29uZmlybS0nLmxlbmd0aCkpLnZhbHVlID0gJydcblx0XHRcdGlucHV0LnZhbHVlID0gJydcblx0XHRcdHJldHVyblxuXHRcdH1cblxuXHRcdGNvbmZpcm1JbnB1dCA9IGRvY3VtZW50LmdldEVsZW1lbnRCeUlkKCdjb25maXJtLScraWQpXG5cdFx0aWYgKCFjb25maXJtSW5wdXQpIHtcblx0XHRcdHJldHVyblxuXHRcdH1cblxuXHRcdGNoZWNrUGFzc3dvcmRGbiA9IGZ1bmN0aW9uKCkge1xuXHRcdFx0aWYgKGlucHV0LnZhbHVlID09PSBjb25maXJtSW5wdXQudmFsdWUpIHtcblx0XHRcdFx0aWYgKG5vTWF0Y2hFbGVtKSB7XG5cdFx0XHRcdFx0bm9NYXRjaEVsZW0ucmVtb3ZlKClcblx0XHRcdFx0XHRub01hdGNoRWxlbSA9IG51bGxcblx0XHRcdFx0fVxuXG5cdFx0XHRcdGNvbmZpcm1JbnB1dC5zZXRDdXN0b21WYWxpZGl0eShcIlwiKVxuXHRcdFx0XHRyZXR1cm5cblx0XHRcdH1cblxuXHRcdFx0Y29uZmlybUlucHV0LnNldEN1c3RvbVZhbGlkaXR5KFwiUGFzc3dvcmRzIGRvIG5vdCBtYXRjaFwiKVxuXG5cdFx0XHRpZiAobm9NYXRjaEVsZW0pIHtcblx0XHRcdFx0cmV0dXJuXG5cdFx0XHR9XG5cblx0XHRcdGNvbnN0IGNsaWVudFJlY3QgPSBjb25maXJtSW5wdXQuZ2V0Qm91bmRpbmdDbGllbnRSZWN0KClcblxuXHRcdFx0bm9NYXRjaEVsZW0gPSBkb2N1bWVudC5jcmVhdGVFbGVtZW50KCdkaXYnKVxuXHRcdFx0bm9NYXRjaEVsZW0udGV4dENvbnRlbnQgPSAnVGhlIHBhc3N3b3JkcyBkbyBub3QgbWF0Y2gnXG5cdFx0XHRub01hdGNoRWxlbS5zdHlsZS5sZWZ0ID0gY2xpZW50UmVjdC5sZWZ0KydweCdcblx0XHRcdG5vTWF0Y2hFbGVtLnN0eWxlLnRvcCA9IGNsaWVudFJlY3QudG9wK2NsaWVudFJlY3QuaGVpZ2h0K2J1ZmZlcisncHgnXG5cdFx0XHRub01hdGNoRWxlbS5jbGFzc0xpc3QuYWRkKCdpbnB1dC1lcnJvcicpXG5cdFx0XHRkb2N1bWVudC5ib2R5LmFwcGVuZENoaWxkKG5vTWF0Y2hFbGVtKVxuXHRcdH1cblxuXHRcdGlucHV0LmFkZEV2ZW50TGlzdGVuZXIoJ2tleXVwJywgY2hlY2tQYXNzd29yZEZuKVxuXHRcdGNvbmZpcm1JbnB1dC5hZGRFdmVudExpc3RlbmVyKCdrZXl1cCcsIGNoZWNrUGFzc3dvcmRGbilcblx0fSlcbn1cblxuZXhwb3J0IGZ1bmN0aW9uIHNldHVwVGltZUlucHV0KCkge1xuXHRkb2N1bWVudC5xdWVyeVNlbGVjdG9yQWxsKCdpbnB1dFt0eXBlPVwidGltZVwiXScpLmZvckVhY2goZnVuY3Rpb24oaW5wdXQpIHtcblx0XHRpZiAoaW5wdXQudmFsdWUgPT09ICcnKSB7XG5cdFx0XHRpbnB1dC52YWx1ZSA9ICcwMDowMCdcblx0XHR9XG5cdH0pXG59XG5cbmV4cG9ydCBkZWZhdWx0IGZ1bmN0aW9uKCkge1xuICAgIHNldHVwUGFzc3dvcmRJbnB1dCgpXG5cdHNldHVwVGltZUlucHV0KClcbn1cbiJdLCJzb3VyY2VSb290IjoiIn0=\n//# sourceURL=webpack-internal:///./src/setup-form.js\n");

/***/ })

/******/ });