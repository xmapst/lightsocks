import { useIntersectionObserver, useSyncedRef, useUnmountEffect } from '@react-hookz/web'
import { useReactTable, getSortedRowModel, getFilteredRowModel, getCoreRowModel, flexRender, createColumnHelper } from '@tanstack/react-table'
import classnames from 'classnames'
import { groupBy } from 'lodash-es'
import { useMemo, useLayoutEffect, useRef, useState, useEffect } from 'react'

import { Header, Checkbox, Modal, Icon, Drawer, Card, Button } from '@components'
import { fromNow } from '@lib/date'
import { formatTraffic } from '@lib/helper'
import { useObject, useVisible } from '@lib/hook'
import type * as API from '@lib/request'
import { useClient, useConnectionStreamReader, useI18n } from '@stores'

import { ConnectionInfo } from './Info'
import { Types } from './Types'
import { type Connection, type FormatConnection, useConnections } from './store'
import './style.scss'

const Columns = {
    Client: 'client',
    Network: 'network',
    Type: 'type',
    Speed: 'speed',
    Upload: 'upload',
    Download: 'download',
    Time: 'time',
} as const

const shouldCenter = new Set<string>([Columns.Client, Columns.Network, Columns.Type, Columns.Speed, Columns.Upload, Columns.Download, Columns.Time])

function formatSpeed (upload: number, download: number) {
    switch (true) {
        case upload === 0 && download === 0:
            return '-'
        case upload !== 0 && download !== 0:
            return `↑ ${formatTraffic(upload)}/s ↓ ${formatTraffic(download)}/s`
        case upload !== 0:
            return `↑ ${formatTraffic(upload)}/s`
        default:
            return `↓ ${formatTraffic(download)}/s`
    }
}

const columnHelper = createColumnHelper<FormatConnection>()

export default function Connections () {
    const { translation, lang } = useI18n()
    const t = useMemo(() => translation('Connections').t, [translation])
    const connStreamReader = useConnectionStreamReader()
    const readerRef = useSyncedRef(connStreamReader)
    const client = useClient()
    const cardRef = useRef<HTMLDivElement>(null)

    // total
    const [traffic, setTraffic] = useObject({
        uploadTotal: 0,
        downloadTotal: 0,
    })

    // close all connections
    const { visible, show, hide } = useVisible()
    function handleCloseConnections () {
        client.closeAllConnections().finally(() => hide())
    }

    // connections
    const { connections, feed, save, toggleSave } = useConnections()
    const data: FormatConnection[] = useMemo(() => connections.map(
        c => ({
            id: c.ID,
            client: c.Metadata.Client,
            source: c.Metadata.Source,
            target: c.Metadata.Target,
            time: new Date(c.Start).getTime(),
            upload: c.Upload,
            download: c.Download,
            type: c.Metadata.Type,
            network: c.Metadata.Network.toUpperCase(),
            speed: { upload: c.uploadSpeed, download: c.downloadSpeed },
            completed: !!c.completed,
            original: c,
        }),
    ), [connections])
    const types = useMemo(() => {
        const gb = groupBy(connections, 'Metadata.Type')
        return Object.keys(gb)
            .map(key => ({ label: key, number: gb[key].length }))
            .sort((a, b) => a.label.localeCompare(b.label))
    }, [connections])

    // table
    const pinRef = useRef<HTMLTableCellElement>(null)
    const intersection = useIntersectionObserver(pinRef, { threshold: [1] })
    const columns = useMemo(
        () => [
            columnHelper.accessor(Columns.Client, { minSize: 260, size: 260, header: t(`columns.${Columns.Client}`) }),
            columnHelper.accessor(Columns.Network, { minSize: 80, size: 80, header: t(`columns.${Columns.Network}`) }),
            columnHelper.accessor(Columns.Type, { minSize: 100, size: 100, header: t(`columns.${Columns.Type}`), filterFn: 'equals' }),
            columnHelper.accessor(
                row => [row.speed.upload, row.speed.download],
                {
                    id: Columns.Speed,
                    header: t(`columns.${Columns.Speed}`),
                    minSize: 200,
                    size: 200,
                    sortDescFirst: true,
                    sortingFn (rowA, rowB) {
                        const speedA = rowA.original?.speed ?? { upload: 0, download: 0 }
                        const speedB = rowB.original?.speed ?? { upload: 0, download: 0 }
                        return speedA.download === speedB.download
                            ? speedA.upload - speedB.upload
                            : speedA.download - speedB.download
                    },
                    cell: cell => formatSpeed(cell.getValue()[0], cell.getValue()[1]),
                },
            ),
            columnHelper.accessor(Columns.Upload, { minSize: 100, size: 100, header: t(`columns.${Columns.Upload}`), cell: cell => formatTraffic(cell.getValue()) }),
            columnHelper.accessor(Columns.Download, { minSize: 100, size: 100, header: t(`columns.${Columns.Download}`), cell: cell => formatTraffic(cell.getValue()) }),
            columnHelper.accessor(
                Columns.Time,
                {
                    minSize: 120,
                    size: 120,
                    header: t(`columns.${Columns.Time}`),
                    cell: cell => fromNow(new Date(cell.getValue()), lang),
                    sortingFn: (rowA, rowB) => (rowB.original?.time ?? 0) - (rowA.original?.time ?? 0),
                },
            ),
        ],
        [lang, t],
    )

    useLayoutEffect(() => {
        function handleConnection (snapshots: API.Snapshot[]) {
            for (const snapshot of snapshots) {
                setTraffic({
                    uploadTotal: snapshot.UploadTotal,
                    downloadTotal: snapshot.DownloadTotal,
                })

                feed(snapshot.Connections)
            }
        }

        connStreamReader?.subscribe('data', handleConnection)
        return () => {
            connStreamReader?.unsubscribe('data', handleConnection)
        }
    }, [connStreamReader, feed, setTraffic])
    useUnmountEffect(() => {
        readerRef.current?.destory()
    })

    const instance = useReactTable({
        data,
        columns,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
        getFilteredRowModel: getFilteredRowModel(),
        initialState: {
            sorting: [{ id: Columns.Time, desc: false }],
        },
        columnResizeMode: 'onChange',
        enableColumnResizing: true,
    })

    const headerGroup = instance.getHeaderGroups()[0]

    // filter
    const [type, setType] = useState('')
    function handleTypeSelected (label: string) {
        setType(label)
        instance.getColumn(Columns.Type)?.setFilterValue(label || undefined)
    }

    // click item
    const [drawerState, setDrawerState] = useObject({
        visible: false,
        selectedID: '',
        connection: {} as Partial<Connection>,
    })
    function handleConnectionClosed () {
        setDrawerState(d => { d.connection.completed = true })
        client.closeConnection(drawerState.selectedID).finally(() => hide())
    }
    const latestConnection = useSyncedRef(drawerState.connection)
    useEffect(() => {
        const conn = data.find(c => c.id === drawerState.selectedID)?.original
        if (conn) {
            setDrawerState(d => {
                d.connection = { ...conn }
                if (drawerState.selectedID === latestConnection.current.ID) {
                    d.connection.completed = latestConnection.current.completed
                }
            })
        } else if (Object.keys(latestConnection.current).length !== 0 && !latestConnection.current.completed) {
            setDrawerState(d => { d.connection.completed = true })
        }
    }, [data, drawerState.selectedID, latestConnection, setDrawerState])

    const scrolled = useMemo(() => (intersection?.intersectionRatio ?? 0) < 1, [intersection])
    const headers = headerGroup.headers.map((header, idx) => {
        const column = header.column
        const id = column.id
        return (
            <th
                className={classnames('connections-th', {
                    resizing: column.getIsResizing(),
                    fixed: column.id === Columns.Client,
                    shadow: scrolled && column.id === Columns.Client,
                })}
                style={{ width: header.getSize() }}
                ref={column.id === Columns.Client ? pinRef : undefined}
                key={id}>
                <div onClick={column.getToggleSortingHandler()}>
                    { flexRender(header.column.columnDef.header, header.getContext()) }
                    {
                        column.getIsSorted() !== false
                            ? column.getIsSorted() === 'desc' ? ' ↓' : ' ↑'
                            : null
                    }
                </div>
                { idx !== headerGroup.headers.length - 1 &&
                    <div
                        onMouseDown={header.getResizeHandler()}
                        onTouchStart={header.getResizeHandler()}
                        className="connections-resizer" />
                }
            </th>
        )
    })

    const content = instance.getRowModel().rows.map(row => {
        return (
            <tr
                className="cursor-default select-none"
                key={row.original?.id}
                onClick={() => setDrawerState({ visible: true, selectedID: row.original?.id })}>
                {
                    row.getAllCells().map(cell => {
                        const classname = classnames(
                            'connections-block',
                            { 'text-center': shouldCenter.has(cell.column.id), completed: row.original?.completed },
                            {
                                fixed: cell.column.id === Columns.Client,
                                shadow: scrolled && cell.column.id === Columns.Client,
                            },
                        )
                        return (
                            <td
                                className={classname}
                                style={{ width: cell.column.getSize() }}
                                key={cell.column.id}>
                                { flexRender(cell.column.columnDef.cell, cell.getContext()) }
                            </td>
                        )
                    })
                }
            </tr>
        )
    })

    return (
        <div className="page !h-100vh">
            <Header title={t('title')}>
                <span className="connections-filter flex-1 cursor-default">
                    {`(${t('total.text')}: ${t('total.upload')} ${formatTraffic(traffic.uploadTotal)} ${t('total.download')} ${formatTraffic(traffic.downloadTotal)})`}
                </span>
                <Checkbox className="connections-filter" checked={save} onChange={toggleSave}>{t('keepClosed')}</Checkbox>
                <Icon className="connections-filter dangerous" onClick={show} type="close-all" size={20} />
            </Header>
            { types.length > 1 && <Types types={types} selected={type} onChange={handleTypeSelected} /> }
            <Card ref={cardRef} className="connections-card relative">
                <div className="min-h-full min-w-full overflow-auto">
                    <table>
                        <thead>
                            <tr className="connections-header">
                                { headers }
                            </tr>
                        </thead>
                        <tbody>
                            { content }
                        </tbody>
                    </table>
                </div>
            </Card>
            <Modal title={t('closeAll.title')} show={visible} onClose={hide} onOk={handleCloseConnections}>{t('closeAll.content')}</Modal>
            <Drawer containerRef={cardRef} bodyClassName="flex flex-col" visible={drawerState.visible} width={450}>
                <div className="h-8 flex items-center justify-between">
                    <span className="pl-3 font-bold">{t('info.title')}</span>
                    <Icon type="close" size={16} className="cursor-pointer" onClick={() => setDrawerState('visible', false)} />
                </div>
                <ConnectionInfo className="mt-3 px-5" connection={drawerState.connection} />
                <div className="mt-3 flex justify-end pr-3">
                    <Button type="danger" disabled={drawerState.connection.completed} onClick={() => handleConnectionClosed()}>{ t('info.closeConnection') }</Button>
                </div>
            </Drawer>
        </div>
    )
}
