import { atom, useAtom } from 'jotai'
import { useLocation } from 'react-router-dom'

import { Client } from '@lib/request'

export function useAPIInfo () {
    const hostname = location.hostname
    const port = location.port
    const protocol = location.protocol

    const _location = useLocation()
    const qs = new URLSearchParams(_location.search)
    const secret = qs.get('secret') ?? ''

    return { hostname, port, secret, protocol }
}

const clientAtom = atom({
    key: '',
    instance: null as Client | null,
})

export function useClient () {
    const {
        hostname,
        port,
        secret,
        protocol,
    } = useAPIInfo()

    const [item, setItem] = useAtom(clientAtom)
    const key = `${protocol}//${hostname}:${port}?secret=${secret}`
    if (item.key === key) {
        return item.instance!
    }

    const client = new Client(`${protocol}//${hostname}:${port}`, secret)
    setItem({ key, instance: client })

    return client
}
