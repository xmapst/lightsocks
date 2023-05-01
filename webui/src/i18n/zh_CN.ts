const CN = {
    SideBar: {
        Overview: '总览',
        Info: '版本信息',
        Logs: '日志',
        Connections: '连接',
    },
    Logs: {
        title: '日志',
        levelLabel: '日志等级',
    },
    Connections: {
        title: '连接',
        keepClosed: '保留关闭连接',
        total: {
            text: '总量',
            upload: '上传',
            download: '下载',
        },
        closeAll: {
            title: '警告',
            content: '将会关闭所有连接',
        },
        filter: {
            all: '全部',
        },
        columns: {
            client: '客户端',
            network: '网络',
            type: '类型',
            speed: '速率',
            upload: '上传',
            download: '下载',
            time: '连接时间',
        },
        info: {
            title: '连接信息',
            id: 'ID',
            client: '客户端',
            source: '来源',
            target: '目的',
            upload: '上传',
            download: '下载',
            network: '网络',
            type: '类型',
            status: '状态',
            opening: '连接中',
            closed: '已关闭',
            closeConnection: '关闭连接',
        },
    },
    Modal: {
        ok: '确 定',
        cancel: '取 消',
    },
} as const

export default CN
