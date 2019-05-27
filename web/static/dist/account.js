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
/******/ 	return __webpack_require__(__webpack_require__.s = "./src/account.js");
/******/ })
/************************************************************************/
/******/ ({

/***/ "./src/account.js":
/*!************************!*\
  !*** ./src/account.js ***!
  \************************/
/*! no exports provided */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony import */ var _pagination__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./pagination */ \"./src/pagination.js\");\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\n\nwindow.addEventListener('load', () => new _pagination__WEBPACK_IMPORTED_MODULE_0__[\"default\"](document.querySelector('nav.pagination')));//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvYWNjb3VudC5qcz9lNzI2Il0sIm5hbWVzIjpbIndpbmRvdyIsImFkZEV2ZW50TGlzdGVuZXIiLCJQYWdpbmF0aW9uIiwiZG9jdW1lbnQiLCJxdWVyeVNlbGVjdG9yIl0sIm1hcHBpbmdzIjoiQUFBQTtBQUFBO0FBQUE7Ozs7Ozs7Ozs7Ozs7OztBQWdCQTtBQUVBQSxNQUFNLENBQUNDLGdCQUFQLENBQXdCLE1BQXhCLEVBQWdDLE1BQU0sSUFBSUMsbURBQUosQ0FBZUMsUUFBUSxDQUFDQyxhQUFULENBQXVCLGdCQUF2QixDQUFmLENBQXRDIiwiZmlsZSI6Ii4vc3JjL2FjY291bnQuanMuanMiLCJzb3VyY2VzQ29udGVudCI6WyIvKlxuQ29weXJpZ2h0IDIwMTkgVG9tIFBldGVyc1xuXG5MaWNlbnNlZCB1bmRlciB0aGUgQXBhY2hlIExpY2Vuc2UsIFZlcnNpb24gMi4wICh0aGUgXCJMaWNlbnNlXCIpO1xueW91IG1heSBub3QgdXNlIHRoaXMgZmlsZSBleGNlcHQgaW4gY29tcGxpYW5jZSB3aXRoIHRoZSBMaWNlbnNlLlxuWW91IG1heSBvYnRhaW4gYSBjb3B5IG9mIHRoZSBMaWNlbnNlIGF0XG5cbiAgIGh0dHA6Ly93d3cuYXBhY2hlLm9yZy9saWNlbnNlcy9MSUNFTlNFLTIuMFxuXG5Vbmxlc3MgcmVxdWlyZWQgYnkgYXBwbGljYWJsZSBsYXcgb3IgYWdyZWVkIHRvIGluIHdyaXRpbmcsIHNvZnR3YXJlXG5kaXN0cmlidXRlZCB1bmRlciB0aGUgTGljZW5zZSBpcyBkaXN0cmlidXRlZCBvbiBhbiBcIkFTIElTXCIgQkFTSVMsXG5XSVRIT1VUIFdBUlJBTlRJRVMgT1IgQ09ORElUSU9OUyBPRiBBTlkgS0lORCwgZWl0aGVyIGV4cHJlc3Mgb3IgaW1wbGllZC5cblNlZSB0aGUgTGljZW5zZSBmb3IgdGhlIHNwZWNpZmljIGxhbmd1YWdlIGdvdmVybmluZyBwZXJtaXNzaW9ucyBhbmRcbmxpbWl0YXRpb25zIHVuZGVyIHRoZSBMaWNlbnNlLlxuKi9cblxuaW1wb3J0IFBhZ2luYXRpb24gZnJvbSAnLi9wYWdpbmF0aW9uJ1xuXG53aW5kb3cuYWRkRXZlbnRMaXN0ZW5lcignbG9hZCcsICgpID0+IG5ldyBQYWdpbmF0aW9uKGRvY3VtZW50LnF1ZXJ5U2VsZWN0b3IoJ25hdi5wYWdpbmF0aW9uJykpKVxuIl0sInNvdXJjZVJvb3QiOiIifQ==\n//# sourceURL=webpack-internal:///./src/account.js\n");

/***/ }),

/***/ "./src/pagination.js":
/*!***************************!*\
  !*** ./src/pagination.js ***!
  \***************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, \"default\", function() { return Pagination; });\n/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nclass Pagination {\n  constructor(nav) {\n    let parentNode;\n\n    for (parentNode = nav.parentNode; parentNode.nodeName !== '#document' && !parentNode.getAttribute(\"data-pagination\"); parentNode = parentNode.parentNode);\n\n    if (parentNode.nodeName === '#document') {\n      return;\n    }\n\n    nav.querySelectorAll('a').forEach(function (link) {\n      link.onclick = function () {\n        const request = new XMLHttpRequest();\n        request.open(\"GET\", link.getAttribute(\"href\"));\n\n        request.onload = function () {\n          const div = document.createElement('div');\n          div.innerHTML = request.responseText;\n          new Pagination(div.querySelector('nav.pagination'));\n          parentNode.replaceWith(div.firstElementChild);\n        };\n\n        request.send();\n        return false;\n      };\n    });\n  }\n\n}//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvcGFnaW5hdGlvbi5qcz9kMTM5Il0sIm5hbWVzIjpbIlBhZ2luYXRpb24iLCJjb25zdHJ1Y3RvciIsIm5hdiIsInBhcmVudE5vZGUiLCJub2RlTmFtZSIsImdldEF0dHJpYnV0ZSIsInF1ZXJ5U2VsZWN0b3JBbGwiLCJmb3JFYWNoIiwibGluayIsIm9uY2xpY2siLCJyZXF1ZXN0IiwiWE1MSHR0cFJlcXVlc3QiLCJvcGVuIiwib25sb2FkIiwiZGl2IiwiZG9jdW1lbnQiLCJjcmVhdGVFbGVtZW50IiwiaW5uZXJIVE1MIiwicmVzcG9uc2VUZXh0IiwicXVlcnlTZWxlY3RvciIsInJlcGxhY2VXaXRoIiwiZmlyc3RFbGVtZW50Q2hpbGQiLCJzZW5kIl0sIm1hcHBpbmdzIjoiQUFBQTtBQUFBO0FBQUE7Ozs7Ozs7Ozs7Ozs7OztBQWdCZSxNQUFNQSxVQUFOLENBQWlCO0FBQy9CQyxhQUFXLENBQUNDLEdBQUQsRUFBTTtBQUNoQixRQUFJQyxVQUFKOztBQUVBLFNBQUtBLFVBQVUsR0FBR0QsR0FBRyxDQUFDQyxVQUF0QixFQUFrQ0EsVUFBVSxDQUFDQyxRQUFYLEtBQXdCLFdBQXhCLElBQXVDLENBQUNELFVBQVUsQ0FBQ0UsWUFBWCxDQUF3QixpQkFBeEIsQ0FBMUUsRUFBc0hGLFVBQVUsR0FBR0EsVUFBVSxDQUFDQSxVQUE5SSxDQUNDOztBQUVELFFBQUlBLFVBQVUsQ0FBQ0MsUUFBWCxLQUF3QixXQUE1QixFQUF5QztBQUN4QztBQUNBOztBQUVERixPQUFHLENBQUNJLGdCQUFKLENBQXFCLEdBQXJCLEVBQTBCQyxPQUExQixDQUFrQyxVQUFTQyxJQUFULEVBQWU7QUFDaERBLFVBQUksQ0FBQ0MsT0FBTCxHQUFlLFlBQVc7QUFDekIsY0FBTUMsT0FBTyxHQUFHLElBQUlDLGNBQUosRUFBaEI7QUFDQUQsZUFBTyxDQUFDRSxJQUFSLENBQWEsS0FBYixFQUFvQkosSUFBSSxDQUFDSCxZQUFMLENBQWtCLE1BQWxCLENBQXBCOztBQUNBSyxlQUFPLENBQUNHLE1BQVIsR0FBaUIsWUFBVztBQUMzQixnQkFBTUMsR0FBRyxHQUFHQyxRQUFRLENBQUNDLGFBQVQsQ0FBdUIsS0FBdkIsQ0FBWjtBQUNBRixhQUFHLENBQUNHLFNBQUosR0FBZ0JQLE9BQU8sQ0FBQ1EsWUFBeEI7QUFDQSxjQUFJbEIsVUFBSixDQUFlYyxHQUFHLENBQUNLLGFBQUosQ0FBa0IsZ0JBQWxCLENBQWY7QUFDQWhCLG9CQUFVLENBQUNpQixXQUFYLENBQXVCTixHQUFHLENBQUNPLGlCQUEzQjtBQUNBLFNBTEQ7O0FBTUFYLGVBQU8sQ0FBQ1ksSUFBUjtBQUVBLGVBQU8sS0FBUDtBQUNBLE9BWkQ7QUFhQSxLQWREO0FBZUE7O0FBMUI4QiIsImZpbGUiOiIuL3NyYy9wYWdpbmF0aW9uLmpzLmpzIiwic291cmNlc0NvbnRlbnQiOlsiLypcbkNvcHlyaWdodCAyMDE5IFRvbSBQZXRlcnNcblxuTGljZW5zZWQgdW5kZXIgdGhlIEFwYWNoZSBMaWNlbnNlLCBWZXJzaW9uIDIuMCAodGhlIFwiTGljZW5zZVwiKTtcbnlvdSBtYXkgbm90IHVzZSB0aGlzIGZpbGUgZXhjZXB0IGluIGNvbXBsaWFuY2Ugd2l0aCB0aGUgTGljZW5zZS5cbllvdSBtYXkgb2J0YWluIGEgY29weSBvZiB0aGUgTGljZW5zZSBhdFxuXG4gICBodHRwOi8vd3d3LmFwYWNoZS5vcmcvbGljZW5zZXMvTElDRU5TRS0yLjBcblxuVW5sZXNzIHJlcXVpcmVkIGJ5IGFwcGxpY2FibGUgbGF3IG9yIGFncmVlZCB0byBpbiB3cml0aW5nLCBzb2Z0d2FyZVxuZGlzdHJpYnV0ZWQgdW5kZXIgdGhlIExpY2Vuc2UgaXMgZGlzdHJpYnV0ZWQgb24gYW4gXCJBUyBJU1wiIEJBU0lTLFxuV0lUSE9VVCBXQVJSQU5USUVTIE9SIENPTkRJVElPTlMgT0YgQU5ZIEtJTkQsIGVpdGhlciBleHByZXNzIG9yIGltcGxpZWQuXG5TZWUgdGhlIExpY2Vuc2UgZm9yIHRoZSBzcGVjaWZpYyBsYW5ndWFnZSBnb3Zlcm5pbmcgcGVybWlzc2lvbnMgYW5kXG5saW1pdGF0aW9ucyB1bmRlciB0aGUgTGljZW5zZS5cbiovXG5cbmV4cG9ydCBkZWZhdWx0IGNsYXNzIFBhZ2luYXRpb24ge1xuXHRjb25zdHJ1Y3RvcihuYXYpIHtcblx0XHRsZXQgcGFyZW50Tm9kZVxuXG5cdFx0Zm9yIChwYXJlbnROb2RlID0gbmF2LnBhcmVudE5vZGU7IHBhcmVudE5vZGUubm9kZU5hbWUgIT09ICcjZG9jdW1lbnQnICYmICFwYXJlbnROb2RlLmdldEF0dHJpYnV0ZShcImRhdGEtcGFnaW5hdGlvblwiKTsgcGFyZW50Tm9kZSA9IHBhcmVudE5vZGUucGFyZW50Tm9kZSlcblx0XHRcdDtcblxuXHRcdGlmIChwYXJlbnROb2RlLm5vZGVOYW1lID09PSAnI2RvY3VtZW50Jykge1xuXHRcdFx0cmV0dXJuXG5cdFx0fVxuXG5cdFx0bmF2LnF1ZXJ5U2VsZWN0b3JBbGwoJ2EnKS5mb3JFYWNoKGZ1bmN0aW9uKGxpbmspIHtcblx0XHRcdGxpbmsub25jbGljayA9IGZ1bmN0aW9uKCkge1xuXHRcdFx0XHRjb25zdCByZXF1ZXN0ID0gbmV3IFhNTEh0dHBSZXF1ZXN0KClcblx0XHRcdFx0cmVxdWVzdC5vcGVuKFwiR0VUXCIsIGxpbmsuZ2V0QXR0cmlidXRlKFwiaHJlZlwiKSlcblx0XHRcdFx0cmVxdWVzdC5vbmxvYWQgPSBmdW5jdGlvbigpIHtcblx0XHRcdFx0XHRjb25zdCBkaXYgPSBkb2N1bWVudC5jcmVhdGVFbGVtZW50KCdkaXYnKVxuXHRcdFx0XHRcdGRpdi5pbm5lckhUTUwgPSByZXF1ZXN0LnJlc3BvbnNlVGV4dFxuXHRcdFx0XHRcdG5ldyBQYWdpbmF0aW9uKGRpdi5xdWVyeVNlbGVjdG9yKCduYXYucGFnaW5hdGlvbicpKVxuXHRcdFx0XHRcdHBhcmVudE5vZGUucmVwbGFjZVdpdGgoZGl2LmZpcnN0RWxlbWVudENoaWxkKVxuXHRcdFx0XHR9XG5cdFx0XHRcdHJlcXVlc3Quc2VuZCgpXG5cblx0XHRcdFx0cmV0dXJuIGZhbHNlXG5cdFx0XHR9XG5cdFx0fSlcblx0fVxufVxuXG4iXSwic291cmNlUm9vdCI6IiJ9\n//# sourceURL=webpack-internal:///./src/pagination.js\n");

/***/ })

/******/ });