const EN = {
    SideBar: {
        Info: 'Info Version',
        Overview: 'Overview',
        Logs: 'Logs',
        Connections: 'Connections',
    },
    Logs: {
        title: 'Logs',
        levelLabel: 'Log level',
    },
    Connections: {
        title: 'Connections',
        keepClosed: 'Keep closed connections',
        total: {
            text: 'total',
            upload: 'upload',
            download: 'download',
        },
        closeAll: {
            title: 'Warning',
            content: 'This would close all connections',
        },
        filter: {
            all: 'All',
        },
        columns: {
            client: 'Client',
            network: 'Network',
            type: 'Type',
            speed: 'Speed',
            upload: 'Upload',
            download: 'Download',
            time: 'Time',
        },
        info: {
            title: 'Connection',
            id: 'ID',
            client: 'Client',
            source: 'Source',
            target: 'Target',
            upload: 'Up',
            download: 'Down',
            network: 'Network',
            type: 'Type',
            status: 'Status',
            opening: 'Open',
            closed: 'Closed',
            closeConnection: 'Close',
        },
    },
    Modal: {
        ok: 'Ok',
        cancel: 'Cancel',
    },
} as const

export default EN
