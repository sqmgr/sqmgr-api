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

eval("/*\nCopyright 2019 Tom Peters\n\nLicensed under the Apache License, Version 2.0 (the \"License\");\nyou may not use this file except in compliance with the License.\nYou may obtain a copy of the License at\n\n   http://www.apache.org/licenses/LICENSE-2.0\n\nUnless required by applicable law or agreed to in writing, software\ndistributed under the License is distributed on an \"AS IS\" BASIS,\nWITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\nSee the License for the specific language governing permissions and\nlimitations under the License.\n*/\nwindow.addEventListener('load', function () {\n  var buffer = 100;\n  var notes = document.getElementById('notes');\n  var remainingEl = null;\n\n  var checkRemaining = function () {\n    var remainder = SqMGR.NotesMaxLength - this.value.length;\n\n    if (remainder <= buffer) {\n      if (!remainingEl) {\n        remainingEl = document.createElement('div');\n        remainingEl.classList.add('remaining');\n        this.parentNode.insertBefore(remainingEl, this.nextSibling);\n      }\n\n      remainingEl.textContent = remainder;\n    } else {\n      if (remainingEl) {\n        remainingEl.remove();\n        remainingEl = null;\n      }\n    }\n  };\n\n  notes.onkeyup = notes.onpaste = checkRemaining;\n  checkRemaining.apply(notes);\n  document.querySelector('input[name=\"lock-tz\"]').value = new Date().getTimezoneOffset();\n\n  const pad = function (val) {\n    if (val < 10) {\n      return \"0\" + val;\n    }\n\n    return val;\n  };\n\n  document.querySelector('a[class=\"lock-now\"]').onclick = function () {\n    const now = new Date();\n    document.getElementById('lock-date').value = now.getFullYear() + \"-\" + pad(now.getMonth() + 1) + \"-\" + pad(now.getDate());\n    document.getElementById('lock-time').value = pad(now.getHours()) + \":\" + pad(now.getMinutes());\n    return false;\n  };\n});//# sourceURL=[module]\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIndlYnBhY2s6Ly8vLi9zcmMvZ3JpZC1jdXN0b21pemUuanM/Y2VkMiJdLCJuYW1lcyI6WyJ3aW5kb3ciLCJhZGRFdmVudExpc3RlbmVyIiwiYnVmZmVyIiwibm90ZXMiLCJkb2N1bWVudCIsImdldEVsZW1lbnRCeUlkIiwicmVtYWluaW5nRWwiLCJjaGVja1JlbWFpbmluZyIsInJlbWFpbmRlciIsIlNxTUdSIiwiTm90ZXNNYXhMZW5ndGgiLCJ2YWx1ZSIsImxlbmd0aCIsImNyZWF0ZUVsZW1lbnQiLCJjbGFzc0xpc3QiLCJhZGQiLCJwYXJlbnROb2RlIiwiaW5zZXJ0QmVmb3JlIiwibmV4dFNpYmxpbmciLCJ0ZXh0Q29udGVudCIsInJlbW92ZSIsIm9ua2V5dXAiLCJvbnBhc3RlIiwiYXBwbHkiLCJxdWVyeVNlbGVjdG9yIiwiRGF0ZSIsImdldFRpbWV6b25lT2Zmc2V0IiwicGFkIiwidmFsIiwib25jbGljayIsIm5vdyIsImdldEZ1bGxZZWFyIiwiZ2V0TW9udGgiLCJnZXREYXRlIiwiZ2V0SG91cnMiLCJnZXRNaW51dGVzIl0sIm1hcHBpbmdzIjoiQUFBQTs7Ozs7Ozs7Ozs7Ozs7O0FBZ0JBQSxNQUFNLENBQUNDLGdCQUFQLENBQXdCLE1BQXhCLEVBQWdDLFlBQVc7QUFDMUMsTUFBSUMsTUFBTSxHQUFHLEdBQWI7QUFDQSxNQUFJQyxLQUFLLEdBQUdDLFFBQVEsQ0FBQ0MsY0FBVCxDQUF3QixPQUF4QixDQUFaO0FBQ0EsTUFBSUMsV0FBVyxHQUFHLElBQWxCOztBQUNBLE1BQUlDLGNBQWMsR0FBRyxZQUFXO0FBQy9CLFFBQUlDLFNBQVMsR0FBR0MsS0FBSyxDQUFDQyxjQUFOLEdBQXVCLEtBQUtDLEtBQUwsQ0FBV0MsTUFBbEQ7O0FBQ0EsUUFBSUosU0FBUyxJQUFJTixNQUFqQixFQUF5QjtBQUN4QixVQUFJLENBQUNJLFdBQUwsRUFBa0I7QUFDakJBLG1CQUFXLEdBQUdGLFFBQVEsQ0FBQ1MsYUFBVCxDQUF1QixLQUF2QixDQUFkO0FBQ0FQLG1CQUFXLENBQUNRLFNBQVosQ0FBc0JDLEdBQXRCLENBQTBCLFdBQTFCO0FBQ0EsYUFBS0MsVUFBTCxDQUFnQkMsWUFBaEIsQ0FBNkJYLFdBQTdCLEVBQTBDLEtBQUtZLFdBQS9DO0FBQ0E7O0FBRURaLGlCQUFXLENBQUNhLFdBQVosR0FBMEJYLFNBQTFCO0FBQ0EsS0FSRCxNQVFPO0FBQ04sVUFBSUYsV0FBSixFQUFpQjtBQUNoQkEsbUJBQVcsQ0FBQ2MsTUFBWjtBQUNBZCxtQkFBVyxHQUFHLElBQWQ7QUFDQTtBQUNEO0FBQ0QsR0FoQkQ7O0FBa0JBSCxPQUFLLENBQUNrQixPQUFOLEdBQWdCbEIsS0FBSyxDQUFDbUIsT0FBTixHQUFnQmYsY0FBaEM7QUFDQUEsZ0JBQWMsQ0FBQ2dCLEtBQWYsQ0FBcUJwQixLQUFyQjtBQUVBQyxVQUFRLENBQUNvQixhQUFULENBQXVCLHVCQUF2QixFQUFnRGIsS0FBaEQsR0FBd0QsSUFBSWMsSUFBSixHQUFXQyxpQkFBWCxFQUF4RDs7QUFFQSxRQUFNQyxHQUFHLEdBQUcsVUFBU0MsR0FBVCxFQUFjO0FBQ3pCLFFBQUlBLEdBQUcsR0FBRyxFQUFWLEVBQWM7QUFDYixhQUFPLE1BQU1BLEdBQWI7QUFDQTs7QUFFRCxXQUFPQSxHQUFQO0FBQ0EsR0FORDs7QUFRQXhCLFVBQVEsQ0FBQ29CLGFBQVQsQ0FBdUIscUJBQXZCLEVBQThDSyxPQUE5QyxHQUF3RCxZQUFXO0FBQ2xFLFVBQU1DLEdBQUcsR0FBRyxJQUFJTCxJQUFKLEVBQVo7QUFDR3JCLFlBQVEsQ0FBQ0MsY0FBVCxDQUF3QixXQUF4QixFQUFxQ00sS0FBckMsR0FBNkNtQixHQUFHLENBQUNDLFdBQUosS0FBb0IsR0FBcEIsR0FBMEJKLEdBQUcsQ0FBQ0csR0FBRyxDQUFDRSxRQUFKLEtBQWUsQ0FBaEIsQ0FBN0IsR0FBa0QsR0FBbEQsR0FBd0RMLEdBQUcsQ0FBQ0csR0FBRyxDQUFDRyxPQUFKLEVBQUQsQ0FBeEc7QUFDSDdCLFlBQVEsQ0FBQ0MsY0FBVCxDQUF3QixXQUF4QixFQUFxQ00sS0FBckMsR0FBNkNnQixHQUFHLENBQUNHLEdBQUcsQ0FBQ0ksUUFBSixFQUFELENBQUgsR0FBc0IsR0FBdEIsR0FBNEJQLEdBQUcsQ0FBQ0csR0FBRyxDQUFDSyxVQUFKLEVBQUQsQ0FBNUU7QUFDQSxXQUFPLEtBQVA7QUFDQSxHQUxEO0FBTUEsQ0F6Q0QiLCJmaWxlIjoiLi9zcmMvZ3JpZC1jdXN0b21pemUuanMuanMiLCJzb3VyY2VzQ29udGVudCI6WyIvKlxuQ29weXJpZ2h0IDIwMTkgVG9tIFBldGVyc1xuXG5MaWNlbnNlZCB1bmRlciB0aGUgQXBhY2hlIExpY2Vuc2UsIFZlcnNpb24gMi4wICh0aGUgXCJMaWNlbnNlXCIpO1xueW91IG1heSBub3QgdXNlIHRoaXMgZmlsZSBleGNlcHQgaW4gY29tcGxpYW5jZSB3aXRoIHRoZSBMaWNlbnNlLlxuWW91IG1heSBvYnRhaW4gYSBjb3B5IG9mIHRoZSBMaWNlbnNlIGF0XG5cbiAgIGh0dHA6Ly93d3cuYXBhY2hlLm9yZy9saWNlbnNlcy9MSUNFTlNFLTIuMFxuXG5Vbmxlc3MgcmVxdWlyZWQgYnkgYXBwbGljYWJsZSBsYXcgb3IgYWdyZWVkIHRvIGluIHdyaXRpbmcsIHNvZnR3YXJlXG5kaXN0cmlidXRlZCB1bmRlciB0aGUgTGljZW5zZSBpcyBkaXN0cmlidXRlZCBvbiBhbiBcIkFTIElTXCIgQkFTSVMsXG5XSVRIT1VUIFdBUlJBTlRJRVMgT1IgQ09ORElUSU9OUyBPRiBBTlkgS0lORCwgZWl0aGVyIGV4cHJlc3Mgb3IgaW1wbGllZC5cblNlZSB0aGUgTGljZW5zZSBmb3IgdGhlIHNwZWNpZmljIGxhbmd1YWdlIGdvdmVybmluZyBwZXJtaXNzaW9ucyBhbmRcbmxpbWl0YXRpb25zIHVuZGVyIHRoZSBMaWNlbnNlLlxuKi9cblxud2luZG93LmFkZEV2ZW50TGlzdGVuZXIoJ2xvYWQnLCBmdW5jdGlvbigpIHtcblx0dmFyIGJ1ZmZlciA9IDEwMFxuXHR2YXIgbm90ZXMgPSBkb2N1bWVudC5nZXRFbGVtZW50QnlJZCgnbm90ZXMnKVxuXHR2YXIgcmVtYWluaW5nRWwgPSBudWxsXG5cdHZhciBjaGVja1JlbWFpbmluZyA9IGZ1bmN0aW9uKCkge1xuXHRcdHZhciByZW1haW5kZXIgPSBTcU1HUi5Ob3Rlc01heExlbmd0aCAtIHRoaXMudmFsdWUubGVuZ3RoXG5cdFx0aWYgKHJlbWFpbmRlciA8PSBidWZmZXIpIHtcblx0XHRcdGlmICghcmVtYWluaW5nRWwpIHtcblx0XHRcdFx0cmVtYWluaW5nRWwgPSBkb2N1bWVudC5jcmVhdGVFbGVtZW50KCdkaXYnKVxuXHRcdFx0XHRyZW1haW5pbmdFbC5jbGFzc0xpc3QuYWRkKCdyZW1haW5pbmcnKVxuXHRcdFx0XHR0aGlzLnBhcmVudE5vZGUuaW5zZXJ0QmVmb3JlKHJlbWFpbmluZ0VsLCB0aGlzLm5leHRTaWJsaW5nKVxuXHRcdFx0fVxuXG5cdFx0XHRyZW1haW5pbmdFbC50ZXh0Q29udGVudCA9IHJlbWFpbmRlclxuXHRcdH0gZWxzZSB7XG5cdFx0XHRpZiAocmVtYWluaW5nRWwpIHtcblx0XHRcdFx0cmVtYWluaW5nRWwucmVtb3ZlKClcblx0XHRcdFx0cmVtYWluaW5nRWwgPSBudWxsXG5cdFx0XHR9XG5cdFx0fVxuXHR9XG5cblx0bm90ZXMub25rZXl1cCA9IG5vdGVzLm9ucGFzdGUgPSBjaGVja1JlbWFpbmluZ1xuXHRjaGVja1JlbWFpbmluZy5hcHBseShub3RlcylcblxuXHRkb2N1bWVudC5xdWVyeVNlbGVjdG9yKCdpbnB1dFtuYW1lPVwibG9jay10elwiXScpLnZhbHVlID0gbmV3IERhdGUoKS5nZXRUaW1lem9uZU9mZnNldCgpXG5cblx0Y29uc3QgcGFkID0gZnVuY3Rpb24odmFsKSB7XG5cdFx0aWYgKHZhbCA8IDEwKSB7XG5cdFx0XHRyZXR1cm4gXCIwXCIgKyB2YWxcblx0XHR9XG5cblx0XHRyZXR1cm4gdmFsXG5cdH1cblxuXHRkb2N1bWVudC5xdWVyeVNlbGVjdG9yKCdhW2NsYXNzPVwibG9jay1ub3dcIl0nKS5vbmNsaWNrID0gZnVuY3Rpb24oKSB7XG5cdFx0Y29uc3Qgbm93ID0gbmV3IERhdGUoKVxuXHQgICAgZG9jdW1lbnQuZ2V0RWxlbWVudEJ5SWQoJ2xvY2stZGF0ZScpLnZhbHVlID0gbm93LmdldEZ1bGxZZWFyKCkgKyBcIi1cIiArIHBhZChub3cuZ2V0TW9udGgoKSsxKSArIFwiLVwiICsgcGFkKG5vdy5nZXREYXRlKCkpXG5cdFx0ZG9jdW1lbnQuZ2V0RWxlbWVudEJ5SWQoJ2xvY2stdGltZScpLnZhbHVlID0gcGFkKG5vdy5nZXRIb3VycygpKSArIFwiOlwiICsgcGFkKG5vdy5nZXRNaW51dGVzKCkpXG5cdFx0cmV0dXJuIGZhbHNlXG5cdH1cbn0pXG4iXSwic291cmNlUm9vdCI6IiJ9\n//# sourceURL=webpack-internal:///./src/grid-customize.js\n");

/***/ })

/******/ });