import axios, { type AxiosInstance } from 'axios'

export interface Snapshot {
    UploadTotal: number
    DownloadTotal: number
    Connections: Connections[]
}

export interface Connections {
    ID: string
    Metadata: {
        Network: string
        Type: string
        Client: string
        Source: string
        Target: string
    }
    Upload: number
    Download: number
    Start: string
}

export interface Version {
    Name: string
    Version: string
    BuildTime: string
    GO: {
        OS: string
        ARCH: string
        Version: string
    }
    Git: {
        Url: string
        Branch: string
        Commit: string
    }
}

export class Client {
    private readonly axiosClient: AxiosInstance

    constructor (url: string, secret?: string) {
        this.axiosClient = axios.create({
            baseURL: url,
            headers: secret ? { Authorization: `Bearer ${secret}` } : {},
        })
    }

    async closeAllConnections () {
        return await this.axiosClient.delete('api/connections')
    }

    async closeConnection (id: string) {
        return await this.axiosClient.delete(`api/connections/${id}`)
    }

    async getConnections () {
        return await this.axiosClient.get<Snapshot>('api/connections')
    }

    async getVersion () {
        return await this.axiosClient.get<Version>('api/version')
    }
}
