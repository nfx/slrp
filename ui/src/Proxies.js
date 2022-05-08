import { Card, LiveFilter, TimeDiff, http, IconHeader, useTitle } from './util'
import { useState } from 'react';

export default function Proxies() {
  useTitle("Proxies")
  const [pool, setPool] = useState(null);
  return <div>
    {pool != null && <div className="row row-cols-1 row-cols-sm-2 row-cols-md-3">
      {pool.Cards.map(card => 
        <Card key={card.Name} label={card.Name} value={card.Value} /> )}
    </div>}
    <LiveFilter endpoint="/pool" onUpdate={setPool} minDelay={2000} />
    {pool != null && <div className="card table-responsive">
      <table className="table text-start caption-top">
        <thead>
          <tr className="text-uppercase text-muted">
            <th>Proxy</th>
            <IconHeader icon='speedometer2' title='Speed' />
            <IconHeader icon='check2-circle' title='Ok' />
            <IconHeader icon='activity' title='Rate' />
            <th>Offered</th>
            <th>Succeed</th>
            <th />
          </tr>
        </thead>
        <tbody>
        {pool.Entries.map(proxy => 
          <Entry key={proxy.Proxy} {...proxy} />)}
        </tbody>
      </table>
    </div>}
  </div>
}

function timeTook(duration) {
  let ms = duration / 1000000
  if (ms < 1000) {
    return `${ms.toFixed()}ms`
  }
  return `${(ms/1000).toFixed(2)}s`
}

function Entry(props) {
  const proxy = props.Proxy
  const {FirstSeen, LastSeen, Timeouts, Ok, ReanimateAfter, Speed} = props
  const removeProxy = e => {
    http.delete(`/probe/${proxy.replace("//", '')}`)
    return false
  }
  return <tr className='list-group-item-action'>
    <td>
      <a className='link-primary app-link' 
        href={`/history?filter=Proxy:"${proxy}"`} 
        rel="noreferrer" 
        target="_blank">
        {proxy}
      </a> <TimeDiff ts={FirstSeen*1000} title='First seen' />
    </td>
    <td>{timeTook(Speed)}</td>
    <td>
      {Ok && <i className='link-success bi bi-check2-circle' />}
      {!Ok && <span>
        <i className="link-warning bi bi-alarm" /> <TimeDiff ts={ReanimateAfter} title='Reanimate after' />
      </span>}
    </td>
    <td>
      <HourSuccessRate {...props} />
    </td>
    <td>{props.Offered} {Timeouts > 0 && 
      <sup className='text-muted'><i className='bi bi-hourglass' />{Timeouts}</sup>}</td>
    <td>{props.Succeed} <TimeDiff ts={LastSeen*1000} title='Last seen' /></td>
    <td>
      <a href='#remove' onClick={removeProxy}>x</a>
    </td>
  </tr>
}

function HourSuccessRate({HourSucceed, HourOffered}) {
  // https://stackoverflow.com/questions/45514676/react-check-if-element-is-visible-in-dom
  const rate = HourSucceed.map((s, i) => s === 0 ? 0 : 100 * s / HourOffered[i])
  const x = r => {
    let l = r
    let e = 100 - l
    let t = {
      width: '2px',
      height: '20px',
      float: 'left',
      border: '0',
    }
    if (r > 0) {
      t.backgroundColor = 'green'
      t.backgroundImage = `linear-gradient(0deg, #080 ${l}%, #fff ${e}%)`
    }
    return t
  }
  // debugger
  let s = {'height': '100%'}
  return <div style={s}>
    {rate.map((r, i) => 
      <div key={i} style={x(r)} />
    )}
  </div>
} 