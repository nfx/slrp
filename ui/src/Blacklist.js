import { IconHeader, LiveFilter, SearchFacet, useTitle, http } from './util'
import { useState } from 'react';

function Item({Proxy, Failure, Sources}) {
  const removeProxy = e => {
    http.delete(`/blacklist/${Proxy.replace("//", '')}`)
    return false
  }
  return <tr>
    <td>
      <a href='#remove' onClick={removeProxy}>x</a>
    </td>
    <td className='proxy'>
      <del>{Proxy}</del> <sup className='text-muted' title={Sources.join(', ')}>{Sources.length} sources</sup>
      {/* <small className='text-muted'>{Sources.join(', ')}</small> */}
    </td>
    <td className='failure text-muted'>{Failure}</td>
  </tr>
}

export default function Blacklist() {
  useTitle("Blacklist")
  const [blacklist, setBlacklist] = useState(null);
  return <div className="card blacklist table-responsive">
    <LiveFilter endpoint="/blacklist" onUpdate={setBlacklist} minDelay={10000} />
    {blacklist != null && <div>
      <SearchFacet name='Common failures' items={blacklist.TopFailures} link={`/blacklist?filter=Failure ~ "$"`} />
      <SearchFacet name='Sources' items={blacklist.TopSources} />
      <table className='table text-start table-sm'>
        <thead>
          <tr className="text-uppercase text-muted">
            <td />
            <IconHeader icon="link proxy" title="Proxy used" />
            <IconHeader icon="emoji-dizzy failure" title="Failure" />
          </tr>
        </thead>
        <tbody>
          {blacklist.Items.map(r => <Item key={r.Proxy} {...r} />)}
        </tbody>
      </table>
    </div>}
  </div>
}