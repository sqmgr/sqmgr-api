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

eval("/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nwindow.addEventListener('load', function () {\n  const buffer = 100;\n  const notes = document.getElementById('notes');\n  let remainingEl = null;\n\n  const checkRemaining = function () {\n    const remainder = SqMGR.NotesMaxLength - this.value.length;\n\n    if (remainder <= buffer) {\n      if (!remainingEl) {\n        remainingEl = document.createElement('div');\n        remainingEl.classList.add('remaining');\n        this.parentNode.insertBefore(remainingEl, this.nextSibling);\n      }\n\n      remainingEl.textContent = remainder;\n    } else {\n      if (remainingEl) {\n        remainingEl.remove();\n        remainingEl = null;\n      }\n    }\n  };\n\n  notes.onkeyup = notes.onpaste = checkRemaining;\n  checkRemaining.apply(notes);\n  const homeTeamName = document.getElementById('home-team-name');\n  const awayTeamName = document.getElementById('away-team-name');\n  const gridName = document.getElementById('grid-name');\n\n  homeTeamName.oninput = awayTeamName.oninput = element => {\n    gridName.textContent = awayTeamName.value + ' vs. ' + homeTeamName.value;\n\n    if (!element) {\n      return;\n    }\n\n    const matchedTeams = TeamColors(element.target.value);\n\n    if (matchedTeams.length === 0) {\n      return;\n    }\n\n    console.log(TeamColors(element.target.value));\n  };\n\n  homeTeamName.oninput(null);\n});//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvZ3JpZC1jdXN0b21pemUuanM/Y2VkMiJdLCJuYW1lcyI6WyJ3aW5kb3ciLCJhZGRFdmVudExpc3RlbmVyIiwiYnVmZmVyIiwibm90ZXMiLCJkb2N1bWVudCIsImdldEVsZW1lbnRCeUlkIiwicmVtYWluaW5nRWwiLCJjaGVja1JlbWFpbmluZyIsInJlbWFpbmRlciIsIlNxTUdSIiwiTm90ZXNNYXhMZW5ndGgiLCJ2YWx1ZSIsImxlbmd0aCIsImNyZWF0ZUVsZW1lbnQiLCJjbGFzc0xpc3QiLCJhZGQiLCJwYXJlbnROb2RlIiwiaW5zZXJ0QmVmb3JlIiwibmV4dFNpYmxpbmciLCJ0ZXh0Q29udGVudCIsInJlbW92ZSIsIm9ua2V5dXAiLCJvbnBhc3RlIiwiYXBwbHkiLCJob21lVGVhbU5hbWUiLCJhd2F5VGVhbU5hbWUiLCJncmlkTmFtZSIsIm9uaW5wdXQiLCJlbGVtZW50IiwibWF0Y2hlZFRlYW1zIiwiVGVhbUNvbG9ycyIsInRhcmdldCIsImNvbnNvbGUiLCJsb2ciXSwibWFwcGluZ3MiOiJBQUFBOzs7Ozs7Ozs7Ozs7Ozs7QUFnQkFBLE1BQU0sQ0FBQ0MsZ0JBQVAsQ0FBd0IsTUFBeEIsRUFBZ0MsWUFBVztBQUMxQyxRQUFNQyxNQUFNLEdBQUcsR0FBZjtBQUNBLFFBQU1DLEtBQUssR0FBR0MsUUFBUSxDQUFDQyxjQUFULENBQXdCLE9BQXhCLENBQWQ7QUFDQSxNQUFJQyxXQUFXLEdBQUcsSUFBbEI7O0FBQ0EsUUFBTUMsY0FBYyxHQUFHLFlBQVc7QUFDakMsVUFBTUMsU0FBUyxHQUFHQyxLQUFLLENBQUNDLGNBQU4sR0FBdUIsS0FBS0MsS0FBTCxDQUFXQyxNQUFwRDs7QUFDQSxRQUFJSixTQUFTLElBQUlOLE1BQWpCLEVBQXlCO0FBQ3hCLFVBQUksQ0FBQ0ksV0FBTCxFQUFrQjtBQUNqQkEsbUJBQVcsR0FBR0YsUUFBUSxDQUFDUyxhQUFULENBQXVCLEtBQXZCLENBQWQ7QUFDQVAsbUJBQVcsQ0FBQ1EsU0FBWixDQUFzQkMsR0FBdEIsQ0FBMEIsV0FBMUI7QUFDQSxhQUFLQyxVQUFMLENBQWdCQyxZQUFoQixDQUE2QlgsV0FBN0IsRUFBMEMsS0FBS1ksV0FBL0M7QUFDQTs7QUFFRFosaUJBQVcsQ0FBQ2EsV0FBWixHQUEwQlgsU0FBMUI7QUFDQSxLQVJELE1BUU87QUFDTixVQUFJRixXQUFKLEVBQWlCO0FBQ2hCQSxtQkFBVyxDQUFDYyxNQUFaO0FBQ0FkLG1CQUFXLEdBQUcsSUFBZDtBQUNBO0FBQ0Q7QUFDRCxHQWhCRDs7QUFrQkFILE9BQUssQ0FBQ2tCLE9BQU4sR0FBZ0JsQixLQUFLLENBQUNtQixPQUFOLEdBQWdCZixjQUFoQztBQUNBQSxnQkFBYyxDQUFDZ0IsS0FBZixDQUFxQnBCLEtBQXJCO0FBRUEsUUFBTXFCLFlBQVksR0FBR3BCLFFBQVEsQ0FBQ0MsY0FBVCxDQUF3QixnQkFBeEIsQ0FBckI7QUFDQSxRQUFNb0IsWUFBWSxHQUFHckIsUUFBUSxDQUFDQyxjQUFULENBQXdCLGdCQUF4QixDQUFyQjtBQUNHLFFBQU1xQixRQUFRLEdBQUd0QixRQUFRLENBQUNDLGNBQVQsQ0FBd0IsV0FBeEIsQ0FBakI7O0FBQ0FtQixjQUFZLENBQUNHLE9BQWIsR0FBdUJGLFlBQVksQ0FBQ0UsT0FBYixHQUF1QkMsT0FBTyxJQUFJO0FBQ3hERixZQUFRLENBQUNQLFdBQVQsR0FBdUJNLFlBQVksQ0FBQ2QsS0FBYixHQUFxQixPQUFyQixHQUErQmEsWUFBWSxDQUFDYixLQUFuRTs7QUFFSCxRQUFJLENBQUNpQixPQUFMLEVBQWM7QUFDYjtBQUNBOztBQUVELFVBQU1DLFlBQVksR0FBR0MsVUFBVSxDQUFDRixPQUFPLENBQUNHLE1BQVIsQ0FBZXBCLEtBQWhCLENBQS9COztBQUNBLFFBQUlrQixZQUFZLENBQUNqQixNQUFiLEtBQXdCLENBQTVCLEVBQStCO0FBQzlCO0FBQ0E7O0FBRUtvQixXQUFPLENBQUNDLEdBQVIsQ0FBWUgsVUFBVSxDQUFDRixPQUFPLENBQUNHLE1BQVIsQ0FBZXBCLEtBQWhCLENBQXRCO0FBQ04sR0FiRTs7QUFjSGEsY0FBWSxDQUFDRyxPQUFiLENBQXFCLElBQXJCO0FBQ0EsQ0EzQ0QiLCJmaWxlIjoiLi9zcmMvZ3JpZC1jdXN0b21pemUuanMuanMiLCJzb3VyY2VzQ29udGVudCI6WyIvKlxuQ29weXJpZ2h0IDIwMTkgVG9tIFBldGVyc1xuXG5MaWNlbnNlZCB1bmRlciB0aGUgQXBhY2hlIExpY2Vuc2UsIFZlcnNpb24gMi4wICh0aGUgXCJMaWNlbnNlXCIpO1xueW91IG1heSBub3QgdXNlIHRoaXMgZmlsZSBleGNlcHQgaW4gY29tcGxpYW5jZSB3aXRoIHRoZSBMaWNlbnNlLlxuWW91IG1heSBvYnRhaW4gYSBjb3B5IG9mIHRoZSBMaWNlbnNlIGF0XG5cbiAgIGh0dHA6Ly93d3cuYXBhY2hlLm9yZy9saWNlbnNlcy9MSUNFTlNFLTIuMFxuXG5Vbmxlc3MgcmVxdWlyZWQgYnkgYXBwbGljYWJsZSBsYXcgb3IgYWdyZWVkIHRvIGluIHdyaXRpbmcsIHNvZnR3YXJlXG5kaXN0cmlidXRlZCB1bmRlciB0aGUgTGljZW5zZSBpcyBkaXN0cmlidXRlZCBvbiBhbiBcIkFTIElTXCIgQkFTSVMsXG5XSVRIT1VUIFdBUlJBTlRJRVMgT1IgQ09ORElUSU9OUyBPRiBBTlkgS0lORCwgZWl0aGVyIGV4cHJlc3Mgb3IgaW1wbGllZC5cblNlZSB0aGUgTGljZW5zZSBmb3IgdGhlIHNwZWNpZmljIGxhbmd1YWdlIGdvdmVybmluZyBwZXJtaXNzaW9ucyBhbmRcbmxpbWl0YXRpb25zIHVuZGVyIHRoZSBMaWNlbnNlLlxuKi9cblxud2luZG93LmFkZEV2ZW50TGlzdGVuZXIoJ2xvYWQnLCBmdW5jdGlvbigpIHtcblx0Y29uc3QgYnVmZmVyID0gMTAwXG5cdGNvbnN0IG5vdGVzID0gZG9jdW1lbnQuZ2V0RWxlbWVudEJ5SWQoJ25vdGVzJylcblx0bGV0IHJlbWFpbmluZ0VsID0gbnVsbFxuXHRjb25zdCBjaGVja1JlbWFpbmluZyA9IGZ1bmN0aW9uKCkge1xuXHRcdGNvbnN0IHJlbWFpbmRlciA9IFNxTUdSLk5vdGVzTWF4TGVuZ3RoIC0gdGhpcy52YWx1ZS5sZW5ndGhcblx0XHRpZiAocmVtYWluZGVyIDw9IGJ1ZmZlcikge1xuXHRcdFx0aWYgKCFyZW1haW5pbmdFbCkge1xuXHRcdFx0XHRyZW1haW5pbmdFbCA9IGRvY3VtZW50LmNyZWF0ZUVsZW1lbnQoJ2RpdicpXG5cdFx0XHRcdHJlbWFpbmluZ0VsLmNsYXNzTGlzdC5hZGQoJ3JlbWFpbmluZycpXG5cdFx0XHRcdHRoaXMucGFyZW50Tm9kZS5pbnNlcnRCZWZvcmUocmVtYWluaW5nRWwsIHRoaXMubmV4dFNpYmxpbmcpXG5cdFx0XHR9XG5cblx0XHRcdHJlbWFpbmluZ0VsLnRleHRDb250ZW50ID0gcmVtYWluZGVyXG5cdFx0fSBlbHNlIHtcblx0XHRcdGlmIChyZW1haW5pbmdFbCkge1xuXHRcdFx0XHRyZW1haW5pbmdFbC5yZW1vdmUoKVxuXHRcdFx0XHRyZW1haW5pbmdFbCA9IG51bGxcblx0XHRcdH1cblx0XHR9XG5cdH1cblxuXHRub3Rlcy5vbmtleXVwID0gbm90ZXMub25wYXN0ZSA9IGNoZWNrUmVtYWluaW5nXG5cdGNoZWNrUmVtYWluaW5nLmFwcGx5KG5vdGVzKVxuXG5cdGNvbnN0IGhvbWVUZWFtTmFtZSA9IGRvY3VtZW50LmdldEVsZW1lbnRCeUlkKCdob21lLXRlYW0tbmFtZScpXG5cdGNvbnN0IGF3YXlUZWFtTmFtZSA9IGRvY3VtZW50LmdldEVsZW1lbnRCeUlkKCdhd2F5LXRlYW0tbmFtZScpXG4gICAgY29uc3QgZ3JpZE5hbWUgPSBkb2N1bWVudC5nZXRFbGVtZW50QnlJZCgnZ3JpZC1uYW1lJylcbiAgICBob21lVGVhbU5hbWUub25pbnB1dCA9IGF3YXlUZWFtTmFtZS5vbmlucHV0ID0gZWxlbWVudCA9PiB7XG5cdCAgICBncmlkTmFtZS50ZXh0Q29udGVudCA9IGF3YXlUZWFtTmFtZS52YWx1ZSArICcgdnMuICcgKyBob21lVGVhbU5hbWUudmFsdWVcblxuXHRcdGlmICghZWxlbWVudCkge1xuXHRcdFx0cmV0dXJuXG5cdFx0fVxuXG5cdFx0Y29uc3QgbWF0Y2hlZFRlYW1zID0gVGVhbUNvbG9ycyhlbGVtZW50LnRhcmdldC52YWx1ZSlcblx0XHRpZiAobWF0Y2hlZFRlYW1zLmxlbmd0aCA9PT0gMCkge1xuXHRcdFx0cmV0dXJuXG5cdFx0fVxuXG4gICAgICAgXHRjb25zb2xlLmxvZyhUZWFtQ29sb3JzKGVsZW1lbnQudGFyZ2V0LnZhbHVlKSlcblx0fVxuXHRob21lVGVhbU5hbWUub25pbnB1dChudWxsKVxufSlcbiJdLCJzb3VyY2VSb290IjoiIn0=\n//# sourceURL=webpack-internal:///./src/grid-customize.js\n");

/***/ })

/******/ });