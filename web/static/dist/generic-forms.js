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
/******/ 	return __webpack_require__(__webpack_require__.s = "./src/generic-forms.js");
/******/ })
/************************************************************************/
/******/ ({

/***/ "./src/generic-forms.js":
/*!******************************!*\
  !*** ./src/generic-forms.js ***!
  \******************************/
/*! no exports provided */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony import */ var _setup_form__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./setup-form */ \"./src/setup-form.js\");\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\n\nwindow.addEventListener('load', () => Object(_setup_form__WEBPACK_IMPORTED_MODULE_0__[\"default\"])());\n\n//# sourceURL=webpack:///./src/generic-forms.js?");

/***/ }),

/***/ "./src/setup-form.js":
/*!***************************!*\
  !*** ./src/setup-form.js ***!
  \***************************/
/*! exports provided: setupPasswordInput, setupTimeInput, default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, \"setupPasswordInput\", function() { return setupPasswordInput; });\n/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, \"setupTimeInput\", function() { return setupTimeInput; });\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nfunction setupPasswordInput() {\n  const buffer = 1; // spacing for error message\n\n  document.querySelectorAll('input[type=\"password\"]').forEach(function (input) {\n    const id = input.getAttribute('id');\n    let confirmInput, checkPasswordFn, noMatchElem;\n\n    if (id.indexOf('confirm-') === 0) {\n      // if a confirm- is present, that means that we are expecting user to input\n      // a brand-new password. Do not let Firefox auto fill this\n      document.getElementById(id.substr('confirm-'.length)).value = '';\n      input.value = '';\n      return;\n    }\n\n    confirmInput = document.getElementById('confirm-' + id);\n\n    if (!confirmInput) {\n      return;\n    }\n\n    checkPasswordFn = function () {\n      if (input.value === confirmInput.value) {\n        if (noMatchElem) {\n          noMatchElem.remove();\n          noMatchElem = null;\n        }\n\n        confirmInput.setCustomValidity(\"\");\n        return;\n      }\n\n      confirmInput.setCustomValidity(\"Passwords do not match\");\n\n      if (noMatchElem) {\n        return;\n      }\n\n      const clientRect = confirmInput.getBoundingClientRect();\n      noMatchElem = document.createElement('div');\n      noMatchElem.textContent = 'The passwords do not match';\n      noMatchElem.style.left = clientRect.left + 'px';\n      noMatchElem.style.top = clientRect.top + clientRect.height + buffer + 'px';\n      noMatchElem.classList.add('input-error');\n      document.body.appendChild(noMatchElem);\n    };\n\n    input.addEventListener('keyup', checkPasswordFn);\n    confirmInput.addEventListener('keyup', checkPasswordFn);\n  });\n}\nfunction setupTimeInput() {\n  document.querySelectorAll('input[type=\"time\"]').forEach(function (input) {\n    if (input.value === '') {\n      input.value = '00:00';\n    }\n  });\n}\n/* harmony default export */ __webpack_exports__[\"default\"] = (function () {\n  setupPasswordInput();\n  setupTimeInput();\n});\n\n//# sourceURL=webpack:///./src/setup-form.js?");

/***/ })

/******/ });