import classnames from 'classnames'
import { useMemo } from 'react'

import { type BaseComponentProps } from '@models'
import { useI18n } from '@stores'
import './style.scss'

interface TypesProps extends BaseComponentProps {
    types: Array<{ label: string, number: number }>
    selected: string
    onChange?: (label: string) => void
}

export function Types (props: TypesProps) {
    const { translation } = useI18n()
    const t = useMemo(() => translation('Connections').t, [translation])

    const { className, style } = props
    const classname = classnames('flex flex-wrap px-1', className)
    function handleSelected (label: string) {
        props.onChange?.(label)
    }

    return (
        <div className={classname} style={style}>
            <div className={classnames('connections-types-item mb-2 pt-2', { selected: props.selected === '' })} onClick={() => handleSelected('')}>
                { t('filter.all') }
            </div>
            {
                props.types.map(
                    type => (
                        <div
                            key={type.label}
                            className={classnames('connections-types-item mb-2 pt-2', { selected: props.selected === type.label })}
                            onClick={() => handleSelected(type.label)}>
                            { type.label } ({ type.number })
                        </div>
                    ),
                )
            }
        </div>
    )
}
