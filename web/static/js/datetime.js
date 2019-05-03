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

SqMGR.DateTime = {}
SqMGR.DateTime.format = function(node) {
    node.querySelectorAll('*[data-datetime]').forEach(function(node) {
       const datetimeStr = node.getAttribute("data-datetime")
       node.removeAttribute("data-datetime")

       const datetime = new Date(datetimeStr)
       if (isNaN(datetime.getTime())) {
           return
       }

       node.textContent = datetime.toLocaleDateString('default', {year: '2-digit', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric'})
    })
}

window.addEventListener('load', function() {
    SqMGR.DateTime.format(document.body)
})