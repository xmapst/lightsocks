import classnames from 'classnames'
import { NavLink, useLocation } from 'react-router-dom'

import { type Lang, type Language } from '@i18n'
import { useI18n } from '@stores'
import './style.scss'

interface SidebarProps {
    routes: Array<{
        path: string
        name: string
        noMobile?: boolean
    }>
}

export default function Sidebar (props: SidebarProps) {
    const { routes } = props
    const { translation } = useI18n()
    const { t } = translation('SideBar')
    const location = useLocation()

    const navlinks = routes.map(
        ({ path, name, noMobile }) => (
            <li className={classnames('item', { 'no-mobile': noMobile })} key={name}>
                <NavLink to={{ pathname: path, search: location.search }} className={({ isActive }) => classnames({ active: isActive })}>
                    { t(name as keyof typeof Language[Lang]['SideBar']) }
                </NavLink>
            </li>
        ),
    )

    return (
        <div className="sidebar">
            <ul className="sidebar-menu">
                { navlinks }
            </ul>
        </div>
    )
}
