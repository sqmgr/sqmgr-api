import axios from 'axios'
import EventBus from './EventBus'
import Loading from '../loading'

class API {
    get token() {
        if (!this._token) {
            throw new Error('must set token first')
        }

        return this._token;
    }

    set token(value) {
        this._token = value;
    }

    constructor() {
        this._axios = null
        this._token = null
    }

    axios() {
        if (this._axios) {
            return Promise.resolve(this._axios)
        }

        return axios.get(`/pool/${this.token}/jwt`)
            .then(res => {
                this._axios = axios.create({
                    'headers': {
                        'Authorization': `Bearer ${res.data.result}`
                    }
                })
                return this._axios
            })
    }

    get(path) {
        Loading.show()
        return this.axios()
            .then(client => client.get(path))
            .then(res => res.data.status === "OK" ? res.data.result : Promise.reject(new Error('unknown status')))
            .finally(() => Loading.hide())
    }

    post(path, data) {
        Loading.show()
        return this.axios()
            .then(client => client.post(path, data, {
                    headers: {
                        'Content-Type': 'application/json'
                    }
                }
            ))
            .then(res => {
                EventBus.$emit('data-updated')
                return res
            })
            .finally(() => Loading.hide())
    }

    getSquares() {
        return this.get(`/api/pool/${this.token}/squares`)
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
        return this.updateSquare(squareID, { state, note })
    }
}

export default new API()