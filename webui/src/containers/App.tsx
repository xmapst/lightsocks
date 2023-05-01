import classnames from 'classnames'
import { Route, Navigate, Routes, useLocation, Outlet } from 'react-router-dom'

import Connections from '@containers/Connections'
import Logs from '@containers/Logs'
import SideBar from '@containers/Sidebar'
import { useLogsStreamReader } from '@stores'

import '../styles/common.scss'
import '../styles/iconfont.scss'

export default function App () {
    useLogsStreamReader()

    const location = useLocation()

    const routes = [
        { path: '/logs', name: 'Logs', element: <Logs />, noMobile: true },
        { path: '/connections', name: 'Connections', element: <Connections />, noMobile: true },
    ]

    const layout = (
        <div className={classnames('app')}>
            <SideBar routes={routes} />
            <div className="page-container">
                <Outlet />
            </div>
        </div>
    )

    return (
        <Routes>
            <Route path="/" element={layout}>
                <Route path="/" element={<Navigate to={{ pathname: '/connections', search: location.search }} replace />} />
                {
                    routes.map(
                        route => <Route path={route.path} key={route.path} element={route.element} />,
                    )
                }
            </Route>
        </Routes>
    )
}
