import { IconHeader, LiveFilter, SearchFacet, useTitle, http } from './util'
import { Countries } from './countries';
import { useState } from 'react';

function Item({Proxy, Failure, Sources, Provider, ASN, Country}) {
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
    <td className='country text-muted' title={Countries[Country]?.name}>{Countries[Country]?.flag}</td>
    <td className='provider text-muted'>
      <a href={`https://ipasn.com/asn-downstreams/${ASN}`} rel='nofollow' target='_blank'>{Provider}</a>
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
      <SearchFacet name='Failures' items={blacklist.TopFailures} link={`/blacklist?filter=Failure ~ "$"`} />
      <SearchFacet name='Countries' items={blacklist.TopCountries} link={`/blacklist?filter=Country:$`} />
      <SearchFacet name='Providers' items={blacklist.TopProviders} link={`/blacklist?filter=Provider ~ "$"`} />
      <SearchFacet name='Sources' items={blacklist.TopSources} />
      <table className='table text-start table-sm'>
        <thead>
          <tr className="text-uppercase text-muted">
            <td />
            <IconHeader icon="link proxy" title="Proxy used" />
            <IconHeader icon="flag country" title="Country" />
            <IconHeader icon="hdd-network provider" title="Provider" />
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