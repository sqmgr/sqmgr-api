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

eval("/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nwindow.addEventListener('load', function () {\n  const buffer = 100;\n  const notes = document.getElementById('notes');\n  let remainingEl = null;\n\n  const checkRemaining = function () {\n    const remainder = SqMGR.NotesMaxLength - this.value.length;\n\n    if (remainder <= buffer) {\n      if (!remainingEl) {\n        remainingEl = document.createElement('div');\n        remainingEl.classList.add('remaining');\n        this.parentNode.insertBefore(remainingEl, this.nextSibling);\n      }\n\n      remainingEl.textContent = remainder;\n    } else {\n      if (remainingEl) {\n        remainingEl.remove();\n        remainingEl = null;\n      }\n    }\n  };\n\n  notes.onkeyup = notes.onpaste = checkRemaining;\n  checkRemaining.apply(notes);\n  const homeTeamName = document.getElementById('home-team-name');\n  const awayTeamName = document.getElementById('away-team-name');\n  const gridName = document.getElementById('grid-name');\n\n  homeTeamName.oninput = awayTeamName.oninput = () => {\n    gridName.textContent = awayTeamName.value + ' vs. ' + homeTeamName.value;\n  };\n\n  homeTeamName.oninput(null);\n});//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvZ3JpZC1jdXN0b21pemUuanM/Y2VkMiJdLCJuYW1lcyI6WyJ3aW5kb3ciLCJhZGRFdmVudExpc3RlbmVyIiwiYnVmZmVyIiwibm90ZXMiLCJkb2N1bWVudCIsImdldEVsZW1lbnRCeUlkIiwicmVtYWluaW5nRWwiLCJjaGVja1JlbWFpbmluZyIsInJlbWFpbmRlciIsIlNxTUdSIiwiTm90ZXNNYXhMZW5ndGgiLCJ2YWx1ZSIsImxlbmd0aCIsImNyZWF0ZUVsZW1lbnQiLCJjbGFzc0xpc3QiLCJhZGQiLCJwYXJlbnROb2RlIiwiaW5zZXJ0QmVmb3JlIiwibmV4dFNpYmxpbmciLCJ0ZXh0Q29udGVudCIsInJlbW92ZSIsIm9ua2V5dXAiLCJvbnBhc3RlIiwiYXBwbHkiLCJob21lVGVhbU5hbWUiLCJhd2F5VGVhbU5hbWUiLCJncmlkTmFtZSIsIm9uaW5wdXQiXSwibWFwcGluZ3MiOiJBQUFBOzs7Ozs7Ozs7Ozs7Ozs7QUFnQkFBLE1BQU0sQ0FBQ0MsZ0JBQVAsQ0FBd0IsTUFBeEIsRUFBZ0MsWUFBVztBQUMxQyxRQUFNQyxNQUFNLEdBQUcsR0FBZjtBQUNBLFFBQU1DLEtBQUssR0FBR0MsUUFBUSxDQUFDQyxjQUFULENBQXdCLE9BQXhCLENBQWQ7QUFDQSxNQUFJQyxXQUFXLEdBQUcsSUFBbEI7O0FBQ0EsUUFBTUMsY0FBYyxHQUFHLFlBQVc7QUFDakMsVUFBTUMsU0FBUyxHQUFHQyxLQUFLLENBQUNDLGNBQU4sR0FBdUIsS0FBS0MsS0FBTCxDQUFXQyxNQUFwRDs7QUFDQSxRQUFJSixTQUFTLElBQUlOLE1BQWpCLEVBQXlCO0FBQ3hCLFVBQUksQ0FBQ0ksV0FBTCxFQUFrQjtBQUNqQkEsbUJBQVcsR0FBR0YsUUFBUSxDQUFDUyxhQUFULENBQXVCLEtBQXZCLENBQWQ7QUFDQVAsbUJBQVcsQ0FBQ1EsU0FBWixDQUFzQkMsR0FBdEIsQ0FBMEIsV0FBMUI7QUFDQSxhQUFLQyxVQUFMLENBQWdCQyxZQUFoQixDQUE2QlgsV0FBN0IsRUFBMEMsS0FBS1ksV0FBL0M7QUFDQTs7QUFFRFosaUJBQVcsQ0FBQ2EsV0FBWixHQUEwQlgsU0FBMUI7QUFDQSxLQVJELE1BUU87QUFDTixVQUFJRixXQUFKLEVBQWlCO0FBQ2hCQSxtQkFBVyxDQUFDYyxNQUFaO0FBQ0FkLG1CQUFXLEdBQUcsSUFBZDtBQUNBO0FBQ0Q7QUFDRCxHQWhCRDs7QUFrQkFILE9BQUssQ0FBQ2tCLE9BQU4sR0FBZ0JsQixLQUFLLENBQUNtQixPQUFOLEdBQWdCZixjQUFoQztBQUNBQSxnQkFBYyxDQUFDZ0IsS0FBZixDQUFxQnBCLEtBQXJCO0FBRUEsUUFBTXFCLFlBQVksR0FBR3BCLFFBQVEsQ0FBQ0MsY0FBVCxDQUF3QixnQkFBeEIsQ0FBckI7QUFDQSxRQUFNb0IsWUFBWSxHQUFHckIsUUFBUSxDQUFDQyxjQUFULENBQXdCLGdCQUF4QixDQUFyQjtBQUNHLFFBQU1xQixRQUFRLEdBQUd0QixRQUFRLENBQUNDLGNBQVQsQ0FBd0IsV0FBeEIsQ0FBakI7O0FBQ0FtQixjQUFZLENBQUNHLE9BQWIsR0FBdUJGLFlBQVksQ0FBQ0UsT0FBYixHQUF1QixNQUFNO0FBQ25ERCxZQUFRLENBQUNQLFdBQVQsR0FBdUJNLFlBQVksQ0FBQ2QsS0FBYixHQUFxQixPQUFyQixHQUErQmEsWUFBWSxDQUFDYixLQUFuRTtBQUNILEdBRkU7O0FBR0hhLGNBQVksQ0FBQ0csT0FBYixDQUFxQixJQUFyQjtBQUNBLENBaENEIiwiZmlsZSI6Ii4vc3JjL2dyaWQtY3VzdG9taXplLmpzLmpzIiwic291cmNlc0NvbnRlbnQiOlsiLypcbkNvcHlyaWdodCAyMDE5IFRvbSBQZXRlcnNcblxuTGljZW5zZWQgdW5kZXIgdGhlIEFwYWNoZSBMaWNlbnNlLCBWZXJzaW9uIDIuMCAodGhlIFwiTGljZW5zZVwiKTtcbnlvdSBtYXkgbm90IHVzZSB0aGlzIGZpbGUgZXhjZXB0IGluIGNvbXBsaWFuY2Ugd2l0aCB0aGUgTGljZW5zZS5cbllvdSBtYXkgb2J0YWluIGEgY29weSBvZiB0aGUgTGljZW5zZSBhdFxuXG4gICBodHRwOi8vd3d3LmFwYWNoZS5vcmcvbGljZW5zZXMvTElDRU5TRS0yLjBcblxuVW5sZXNzIHJlcXVpcmVkIGJ5IGFwcGxpY2FibGUgbGF3IG9yIGFncmVlZCB0byBpbiB3cml0aW5nLCBzb2Z0d2FyZVxuZGlzdHJpYnV0ZWQgdW5kZXIgdGhlIExpY2Vuc2UgaXMgZGlzdHJpYnV0ZWQgb24gYW4gXCJBUyBJU1wiIEJBU0lTLFxuV0lUSE9VVCBXQVJSQU5USUVTIE9SIENPTkRJVElPTlMgT0YgQU5ZIEtJTkQsIGVpdGhlciBleHByZXNzIG9yIGltcGxpZWQuXG5TZWUgdGhlIExpY2Vuc2UgZm9yIHRoZSBzcGVjaWZpYyBsYW5ndWFnZSBnb3Zlcm5pbmcgcGVybWlzc2lvbnMgYW5kXG5saW1pdGF0aW9ucyB1bmRlciB0aGUgTGljZW5zZS5cbiovXG5cbndpbmRvdy5hZGRFdmVudExpc3RlbmVyKCdsb2FkJywgZnVuY3Rpb24oKSB7XG5cdGNvbnN0IGJ1ZmZlciA9IDEwMFxuXHRjb25zdCBub3RlcyA9IGRvY3VtZW50LmdldEVsZW1lbnRCeUlkKCdub3RlcycpXG5cdGxldCByZW1haW5pbmdFbCA9IG51bGxcblx0Y29uc3QgY2hlY2tSZW1haW5pbmcgPSBmdW5jdGlvbigpIHtcblx0XHRjb25zdCByZW1haW5kZXIgPSBTcU1HUi5Ob3Rlc01heExlbmd0aCAtIHRoaXMudmFsdWUubGVuZ3RoXG5cdFx0aWYgKHJlbWFpbmRlciA8PSBidWZmZXIpIHtcblx0XHRcdGlmICghcmVtYWluaW5nRWwpIHtcblx0XHRcdFx0cmVtYWluaW5nRWwgPSBkb2N1bWVudC5jcmVhdGVFbGVtZW50KCdkaXYnKVxuXHRcdFx0XHRyZW1haW5pbmdFbC5jbGFzc0xpc3QuYWRkKCdyZW1haW5pbmcnKVxuXHRcdFx0XHR0aGlzLnBhcmVudE5vZGUuaW5zZXJ0QmVmb3JlKHJlbWFpbmluZ0VsLCB0aGlzLm5leHRTaWJsaW5nKVxuXHRcdFx0fVxuXG5cdFx0XHRyZW1haW5pbmdFbC50ZXh0Q29udGVudCA9IHJlbWFpbmRlclxuXHRcdH0gZWxzZSB7XG5cdFx0XHRpZiAocmVtYWluaW5nRWwpIHtcblx0XHRcdFx0cmVtYWluaW5nRWwucmVtb3ZlKClcblx0XHRcdFx0cmVtYWluaW5nRWwgPSBudWxsXG5cdFx0XHR9XG5cdFx0fVxuXHR9XG5cblx0bm90ZXMub25rZXl1cCA9IG5vdGVzLm9ucGFzdGUgPSBjaGVja1JlbWFpbmluZ1xuXHRjaGVja1JlbWFpbmluZy5hcHBseShub3RlcylcblxuXHRjb25zdCBob21lVGVhbU5hbWUgPSBkb2N1bWVudC5nZXRFbGVtZW50QnlJZCgnaG9tZS10ZWFtLW5hbWUnKVxuXHRjb25zdCBhd2F5VGVhbU5hbWUgPSBkb2N1bWVudC5nZXRFbGVtZW50QnlJZCgnYXdheS10ZWFtLW5hbWUnKVxuICAgIGNvbnN0IGdyaWROYW1lID0gZG9jdW1lbnQuZ2V0RWxlbWVudEJ5SWQoJ2dyaWQtbmFtZScpXG4gICAgaG9tZVRlYW1OYW1lLm9uaW5wdXQgPSBhd2F5VGVhbU5hbWUub25pbnB1dCA9ICgpID0+IHtcblx0ICAgIGdyaWROYW1lLnRleHRDb250ZW50ID0gYXdheVRlYW1OYW1lLnZhbHVlICsgJyB2cy4gJyArIGhvbWVUZWFtTmFtZS52YWx1ZVxuXHR9XG5cdGhvbWVUZWFtTmFtZS5vbmlucHV0KG51bGwpXG59KVxuIl0sInNvdXJjZVJvb3QiOiIifQ==\n//# sourceURL=webpack-internal:///./src/grid-customize.js\n");

/***/ })

/******/ });