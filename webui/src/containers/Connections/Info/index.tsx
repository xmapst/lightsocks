import classnames from 'classnames'
import { useMemo } from 'react'

import { formatTraffic } from '@lib/helper'
import { type BaseComponentProps } from '@models'
import { useI18n } from '@stores'

import { type Connection } from '../store'

interface ConnectionsInfoProps extends BaseComponentProps {
    connection: Partial<Connection>
}

export function ConnectionInfo (props: ConnectionsInfoProps) {
    const { translation } = useI18n()
    const t = useMemo(() => translation('Connections').t, [translation])

    return (
        <div className={classnames(props.className, 'flex flex-col overflow-y-auto text-sm')}>
            <div className="my-3 flex">
                <span className="w-20 font-bold">{t('info.id')}</span>
                <span className="font-mono">{props.connection.ID}</span>
            </div>
            <div className="my-3 flex justify-between">
                <div className="flex flex-1">
                    <span className="w-20 font-bold">{t('info.network')}</span>
                    <span className="font-mono">{props.connection.Metadata?.Network.toUpperCase()}</span>
                </div>
                <div className="flex flex-1">
                    <span className="w-20 font-bold">{t('info.type')}</span>
                    <span className="font-mono">{props.connection.Metadata?.Type}</span>
                </div>
            </div>
            <div className="my-3 flex">
                <span className="w-20 font-bold">{t('info.client')}</span>
                <span className="flex-1 break-all font-mono">{props.connection.Metadata?.Client}</span>
            </div>
            <div className="my-3 flex">
                <span className="w-20 font-bold">{t('info.source')}</span>
                <span className="font-mono">{props.connection.Metadata?.Source}</span>
            </div>
            <div className="my-3 flex">
                <span className="w-20 font-bold">{t('info.target')}</span>
                <span className="font-mono">{props.connection.Metadata?.Target}</span>
            </div>
            <div className="my-3 flex justify-between">
                <div className="flex flex-1">
                    <span className="w-20 font-bold">{t('info.upload')}</span>
                    <span className="font-mono">{formatTraffic(props.connection.Upload ?? 0)}</span>
                </div>
                <div className="flex flex-1">
                    <span className="w-20 font-bold">{t('info.download')}</span>
                    <span className="font-mono">{formatTraffic(props.connection.Download ?? 0)}</span>
                </div>
            </div>
            <div className="my-3 flex">
                <span className="w-20 font-bold">{t('info.status')}</span>
                <span className="font-mono">{
                    !props.connection.completed
                        ? <span className="text-green">{t('info.opening')}</span>
                        : <span className="text-red">{t('info.closed')}</span>
                }</span>
            </div>
        </div>
    )
}
