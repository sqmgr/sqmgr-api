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

<template>
   <form method="post">
       <p>This action <strong>cannot</strong> be undone. Please type your email address to confirm.

       <div class="field">
           <input type="email" name="email" autocomplete="off" v-model="email">
       </div>

       <div class="buttons">
           <button type="submit" name="submit" class="destructive" :disabled="disabled">I understand the consequences; delete my account</button>
       </div>
   </form>
</template>

<script>
    export default {
        name: "AccountDeleteConfirmation",
        props: {
            expectedEmail: {
                type: String,
                required: true,
            }
        },
        data() {
            return {
                disabled: true,
                email: '',
            }
        },
        watch: {
            email(newEmail) {
                this.disabled = newEmail !== SqMGR.ExpectedEmail
            }
        }
    }
</script>

<style scoped>
</style>