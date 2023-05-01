import { usePreviousDistinct, useSyncedRef } from '@react-hookz/web'
import { produce } from 'immer'
import { atom, useAtom, useAtomValue } from 'jotai'
import { atomWithStorage } from 'jotai/utils'
import { get } from 'lodash-es'
import { useCallback, useEffect, useMemo, useRef } from 'react'
import { type Get } from 'type-fest'

import { Language, locales, type Lang, getDefaultLanguage, type LocalizedType } from '@i18n'
import { useWarpImmerSetter, type WritableDraft } from '@lib/jotai'
import type * as API from '@lib/request'
import { StreamReader } from '@lib/streamer'
import { type Infer } from '@lib/type'
import { type Log } from '@models/Log'

import { useAPIInfo } from './request'

export const identityAtom = atom(true)

export const languageAtom = atomWithStorage<Lang | undefined>('language', undefined)

export function useI18n () {
    const [defaultLang, setLang] = useAtom(languageAtom)
    const lang = useMemo(() => defaultLang ?? getDefaultLanguage(), [defaultLang])

    const translation = useCallback(
        function <Namespace extends keyof LocalizedType>(namespace: Namespace) {
            function t<Path extends Infer<LocalizedType[Namespace]>> (path: Path) {
                return get(Language[lang][namespace], path) as unknown as Get<LocalizedType[Namespace], Path>
            }
            return { t }
        },
        [lang],
    )

    return { lang, locales, setLang, translation }
}

export const configAtom = atomWithStorage('profile', {
    logLevel: '',
})

export function useConfig () {
    const [data, set] = useAtom(configAtom)

    const setter = useCallback((f: WritableDraft<typeof data>) => {
        set(produce(data, f))
    }, [data, set])

    return { data, set: useWarpImmerSetter(setter) }
}

const logsAtom = atom(new StreamReader<Log>({ bufferLength: 200 }))

export function useLogsStreamReader () {
    const apiInfo = useAPIInfo()
    const { data: { logLevel } } = useConfig()
    const item = useAtomValue(logsAtom)

    const level = logLevel
    const previousKey = usePreviousDistinct(
        `${apiInfo.protocol}//${apiInfo.hostname}:${apiInfo.port}/api/logs?level=${level}&secret=${encodeURIComponent(apiInfo.secret)}`,
    )

    const apiInfoRef = useSyncedRef(apiInfo)

    useEffect(() => {
        if (level) {
            const apiInfo = apiInfoRef.current
            const protocol = apiInfo.protocol === 'http:' ? 'ws:' : 'wss:'
            const logUrl = `${protocol}//${apiInfo.hostname}:${apiInfo.port}/api/logs?level=${level}&token=${encodeURIComponent(apiInfo.secret)}`
            item.connect(logUrl)
        }
    }, [apiInfoRef, item, level, previousKey])

    return item
}

export function useConnectionStreamReader () {
    const apiInfo = useAPIInfo()

    const connection = useRef(new StreamReader<API.Snapshot>({ bufferLength: 200 }))

    const protocol = apiInfo.protocol === 'http:' ? 'ws:' : 'wss:'
    const url = `${protocol}//${apiInfo.hostname}:${apiInfo.port}/api/connections?token=${encodeURIComponent(apiInfo.secret)}`

    useEffect(() => {
        connection.current.connect(url)
    }, [url])

    return connection.current
}
