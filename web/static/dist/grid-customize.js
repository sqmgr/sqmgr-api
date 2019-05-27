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
/******/ 	return __webpack_require__(__webpack_require__.s = "./src/grid-customize.js");
/******/ })
/************************************************************************/
/******/ ({

/***/ "./src/grid-customize.js":
/*!*******************************!*\
  !*** ./src/grid-customize.js ***!
  \*******************************/
/*! no static exports found */
/***/ (function(module, exports) {

eval("/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nwindow.addEventListener('load', function () {\n  var buffer = 100;\n  var notes = document.getElementById('notes');\n  var remainingEl = null;\n\n  var checkRemaining = function () {\n    var remainder = SqMGR.NotesMaxLength - this.value.length;\n\n    if (remainder <= buffer) {\n      if (!remainingEl) {\n        remainingEl = document.createElement('div');\n        remainingEl.classList.add('remaining');\n        this.parentNode.insertBefore(remainingEl, this.nextSibling);\n      }\n\n      remainingEl.textContent = remainder;\n    } else {\n      if (remainingEl) {\n        remainingEl.remove();\n        remainingEl = null;\n      }\n    }\n  };\n\n  notes.onkeyup = notes.onpaste = checkRemaining;\n  checkRemaining.apply(notes);\n});//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvZ3JpZC1jdXN0b21pemUuanM/Y2VkMiJdLCJuYW1lcyI6WyJ3aW5kb3ciLCJhZGRFdmVudExpc3RlbmVyIiwiYnVmZmVyIiwibm90ZXMiLCJkb2N1bWVudCIsImdldEVsZW1lbnRCeUlkIiwicmVtYWluaW5nRWwiLCJjaGVja1JlbWFpbmluZyIsInJlbWFpbmRlciIsIlNxTUdSIiwiTm90ZXNNYXhMZW5ndGgiLCJ2YWx1ZSIsImxlbmd0aCIsImNyZWF0ZUVsZW1lbnQiLCJjbGFzc0xpc3QiLCJhZGQiLCJwYXJlbnROb2RlIiwiaW5zZXJ0QmVmb3JlIiwibmV4dFNpYmxpbmciLCJ0ZXh0Q29udGVudCIsInJlbW92ZSIsIm9ua2V5dXAiLCJvbnBhc3RlIiwiYXBwbHkiXSwibWFwcGluZ3MiOiJBQUFBOzs7Ozs7Ozs7Ozs7Ozs7QUFnQkFBLE1BQU0sQ0FBQ0MsZ0JBQVAsQ0FBd0IsTUFBeEIsRUFBZ0MsWUFBVztBQUMxQyxNQUFJQyxNQUFNLEdBQUcsR0FBYjtBQUNBLE1BQUlDLEtBQUssR0FBR0MsUUFBUSxDQUFDQyxjQUFULENBQXdCLE9BQXhCLENBQVo7QUFDQSxNQUFJQyxXQUFXLEdBQUcsSUFBbEI7O0FBQ0EsTUFBSUMsY0FBYyxHQUFHLFlBQVc7QUFDL0IsUUFBSUMsU0FBUyxHQUFHQyxLQUFLLENBQUNDLGNBQU4sR0FBdUIsS0FBS0MsS0FBTCxDQUFXQyxNQUFsRDs7QUFDQSxRQUFJSixTQUFTLElBQUlOLE1BQWpCLEVBQXlCO0FBQ3hCLFVBQUksQ0FBQ0ksV0FBTCxFQUFrQjtBQUNqQkEsbUJBQVcsR0FBR0YsUUFBUSxDQUFDUyxhQUFULENBQXVCLEtBQXZCLENBQWQ7QUFDQVAsbUJBQVcsQ0FBQ1EsU0FBWixDQUFzQkMsR0FBdEIsQ0FBMEIsV0FBMUI7QUFDQSxhQUFLQyxVQUFMLENBQWdCQyxZQUFoQixDQUE2QlgsV0FBN0IsRUFBMEMsS0FBS1ksV0FBL0M7QUFDQTs7QUFFRFosaUJBQVcsQ0FBQ2EsV0FBWixHQUEwQlgsU0FBMUI7QUFDQSxLQVJELE1BUU87QUFDTixVQUFJRixXQUFKLEVBQWlCO0FBQ2hCQSxtQkFBVyxDQUFDYyxNQUFaO0FBQ0FkLG1CQUFXLEdBQUcsSUFBZDtBQUNBO0FBQ0Q7QUFDRCxHQWhCRDs7QUFrQkFILE9BQUssQ0FBQ2tCLE9BQU4sR0FBZ0JsQixLQUFLLENBQUNtQixPQUFOLEdBQWdCZixjQUFoQztBQUNBQSxnQkFBYyxDQUFDZ0IsS0FBZixDQUFxQnBCLEtBQXJCO0FBQ0EsQ0F4QkQiLCJmaWxlIjoiLi9zcmMvZ3JpZC1jdXN0b21pemUuanMuanMiLCJzb3VyY2VzQ29udGVudCI6WyIvKlxuQ29weXJpZ2h0IDIwMTkgVG9tIFBldGVyc1xuXG5MaWNlbnNlZCB1bmRlciB0aGUgQXBhY2hlIExpY2Vuc2UsIFZlcnNpb24gMi4wICh0aGUgXCJMaWNlbnNlXCIpO1xueW91IG1heSBub3QgdXNlIHRoaXMgZmlsZSBleGNlcHQgaW4gY29tcGxpYW5jZSB3aXRoIHRoZSBMaWNlbnNlLlxuWW91IG1heSBvYnRhaW4gYSBjb3B5IG9mIHRoZSBMaWNlbnNlIGF0XG5cbiAgIGh0dHA6Ly93d3cuYXBhY2hlLm9yZy9saWNlbnNlcy9MSUNFTlNFLTIuMFxuXG5Vbmxlc3MgcmVxdWlyZWQgYnkgYXBwbGljYWJsZSBsYXcgb3IgYWdyZWVkIHRvIGluIHdyaXRpbmcsIHNvZnR3YXJlXG5kaXN0cmlidXRlZCB1bmRlciB0aGUgTGljZW5zZSBpcyBkaXN0cmlidXRlZCBvbiBhbiBcIkFTIElTXCIgQkFTSVMsXG5XSVRIT1VUIFdBUlJBTlRJRVMgT1IgQ09ORElUSU9OUyBPRiBBTlkgS0lORCwgZWl0aGVyIGV4cHJlc3Mgb3IgaW1wbGllZC5cblNlZSB0aGUgTGljZW5zZSBmb3IgdGhlIHNwZWNpZmljIGxhbmd1YWdlIGdvdmVybmluZyBwZXJtaXNzaW9ucyBhbmRcbmxpbWl0YXRpb25zIHVuZGVyIHRoZSBMaWNlbnNlLlxuKi9cblxud2luZG93LmFkZEV2ZW50TGlzdGVuZXIoJ2xvYWQnLCBmdW5jdGlvbigpIHtcblx0dmFyIGJ1ZmZlciA9IDEwMFxuXHR2YXIgbm90ZXMgPSBkb2N1bWVudC5nZXRFbGVtZW50QnlJZCgnbm90ZXMnKVxuXHR2YXIgcmVtYWluaW5nRWwgPSBudWxsXG5cdHZhciBjaGVja1JlbWFpbmluZyA9IGZ1bmN0aW9uKCkge1xuXHRcdHZhciByZW1haW5kZXIgPSBTcU1HUi5Ob3Rlc01heExlbmd0aCAtIHRoaXMudmFsdWUubGVuZ3RoXG5cdFx0aWYgKHJlbWFpbmRlciA8PSBidWZmZXIpIHtcblx0XHRcdGlmICghcmVtYWluaW5nRWwpIHtcblx0XHRcdFx0cmVtYWluaW5nRWwgPSBkb2N1bWVudC5jcmVhdGVFbGVtZW50KCdkaXYnKVxuXHRcdFx0XHRyZW1haW5pbmdFbC5jbGFzc0xpc3QuYWRkKCdyZW1haW5pbmcnKVxuXHRcdFx0XHR0aGlzLnBhcmVudE5vZGUuaW5zZXJ0QmVmb3JlKHJlbWFpbmluZ0VsLCB0aGlzLm5leHRTaWJsaW5nKVxuXHRcdFx0fVxuXG5cdFx0XHRyZW1haW5pbmdFbC50ZXh0Q29udGVudCA9IHJlbWFpbmRlclxuXHRcdH0gZWxzZSB7XG5cdFx0XHRpZiAocmVtYWluaW5nRWwpIHtcblx0XHRcdFx0cmVtYWluaW5nRWwucmVtb3ZlKClcblx0XHRcdFx0cmVtYWluaW5nRWwgPSBudWxsXG5cdFx0XHR9XG5cdFx0fVxuXHR9XG5cblx0bm90ZXMub25rZXl1cCA9IG5vdGVzLm9ucGFzdGUgPSBjaGVja1JlbWFpbmluZ1xuXHRjaGVja1JlbWFpbmluZy5hcHBseShub3Rlcylcbn0pXG4iXSwic291cmNlUm9vdCI6IiJ9\n//# sourceURL=webpack-internal:///./src/grid-customize.js\n");

/***/ })

/******/ });