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

import axios from 'axios'
import EventBus from './EventBus'
import Loading from '../loading'
import JWT from 'jsonwebtoken'

class API {
    constructor() {
        this._axios = null
        this._jwt = null
        this._token = null
    }

    get jwt() {
        if (this._jwt) {
            return Promise.resolve(this._jwt)
        }

        return new Promise(resolve =>
            axios.get(`/pool/${this.token}/jwt`)
                .then(res => resolve(this._jwt = res.data.result))
        )
    }

    get axios() {
        if (this._axios) {
            return Promise.resolve(this._axios)
        }

        return this.jwt
            .then(jwt => {
                return this._axios = axios.create({
                    'headers': { 'Authorization': `Bearer ${this._jwt}` }
                })
            })
    }

    get token() {
        if (!this._token) {
            throw new Error('must set token first')
        }

        return this._token;
    }

    set token(value) {
        this._token = value;
    }

    async request(config) {
        Loading.show()
        return this.axios
            .then(client => client.request(config))
            .catch(err => {
                if (!err.response) {
                    return Promise.reject(err)
                }

                if (err.response.status !== 401) {
                    return Promise.reject(err)
                }

                this.clearClient()
                return this.axios
                    .then(client => client.request(config))
            })
            .then(res => res.data.status === "OK" ? res.data.result : Promise.reject(new Error('unknown status')))
            .finally(() => Loading.hide())
    }

    get(path) {
        return this.request({
            method: 'GET',
            url: path,
        })
    }

    delete(url) {
        return this.request({
            method: 'DELETE',
            url,
        })
    }

    post(path, data, emitEvent = "data-updated") {
        return this.request({
            method: 'POST',
            url: path,
            headers: {
                'Content-Type': 'application/json',
            },
            data,
        })
            .then(res => {
                EventBus.$emit(emitEvent)
                return res
            })
    }

    getSquares() {
        return this.get(`/api/pool/${this.token}/squares`)
    }

    getPool() {
        return this.get(`/api/pool/${this.token}`)
    }

    getPoolGrids() {
        return this.get(`/api/pool/${this.token}/game`)
    }

    createGrid() {
        return this.post(`/api/pool/${this.token}/game`)
    }

    deleteGrid(gridID) {
        return this.delete(`/api/pool/${this.token}/game/${gridID}`)
    }

    getGrid(gridID) {
        return this.get(`/api/pool/${this.token}/game/${gridID}`)
    }

    saveGrid(gridID, data) {
        return this.post(`/api/pool/${this.token}/game/${gridID}`, {
            action: 'save',
            data,
        }, 'grid-updated')
    }

    getSquare(sqId) {
        return this.get(`/api/pool/${this.token}/squares/${sqId}`)
    }

    getLogs() {
        return this.get(`/api/pool/${this.token}/logs`)
    }

    updateSquare(squareID, data) {
        return this.post(`/api/pool/${this.token}/squares/${squareID}`, data)
    }

    claimSquare(squareID, claimant) {
        return this.updateSquare(squareID, {claimant})
    }

    unclaimSquare(squareID) {
        return this.updateSquare(squareID, {unclaim: true})
    }

    setSquareState(squareID, state, note) {
        return this.updateSquare(squareID, {state, note})
    }

    renameSquare(squareID, claimant) {
        return this.updateSquare(squareID, {
            claimant,
            rename: true,
        })
    }

    drawNumbers(gridID) {
        return this.post(`/api/pool/${this.token}/game/${gridID}`, {
            action: 'drawNumbers',
        })
    }

    lockPool() {
        return this.post(`/api/pool/${this.token}`, {
            action: 'LOCK',
        })
    }

    clearClient() {
        this._axios = null
        this._jwt = null
    }

    decodedJWT() {
        return this.jwt
            .then(jwt => JWT.decode(jwt))
    }
}

export default new API()