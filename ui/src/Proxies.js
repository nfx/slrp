import { Card, LiveFilter, TimeDiff, http, IconHeader, useTitle } from './util'
import { Countries } from './countries';
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
            <IconHeader col="country" icon="flag country" title="Country" />
            <IconHeader col="provider" icon="hdd-network provider" title="Provider" />
            <IconHeader icon='speedometer2' title='Speed' />
            <IconHeader icon='check2-circle' title='Ok' />
            <IconHeader col="rate" icon='activity' title='Rate' />
            <th className='col-offered'>Offered</th>
            <th className='col-succeed'>Succeed</th>
            <th className='col-remove' />
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
  const {FirstSeen, LastSeen, Timeouts, Ok, ReanimateAfter, Speed, Country, Provider, ASN} = props
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
    <td className='col-country' 
      title={Countries[Country]?.name}>{Countries[Country]?.flag}</td>
    <td className='col-provider text-muted'>
      <a href={`https://ipasn.com/asn-downstreams/${ASN}`} 
        title={Provider} rel='noreferrer' target='_blank'>{Provider}</a>
    </td>
    <td>{timeTook(Speed)}</td>
    <td>
      {Ok && <i className='link-success bi bi-check2-circle' />}
      {!Ok && <span>
        <i className="link-warning bi bi-alarm" /> <TimeDiff ts={ReanimateAfter} title='Reanimate after' />
      </span>}
    </td>
    <td className='col-rate'>
      <HourSuccessRate {...props} />
    </td>
    <td className='col-offered'>{props.Offered} {Timeouts > 0 && 
      <sup className='text-muted'><i className='bi bi-hourglass' />{Timeouts}</sup>}</td>
    <td className='col-succeed'>{props.Succeed} <TimeDiff ts={LastSeen*1000} title='Last seen' /></td>
    <td className='col-remove'>
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