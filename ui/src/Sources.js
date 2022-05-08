import { Card, IconHeader, TimeDiff, useInterval, useTitle, http } from './util'
import { useState } from 'react';

const overall = ['Exclusive', 'Dirty', 'Contribution']
const pipeline = ['Scheduled', 'New', 'Probing']
const stats = ['Found', 'Timeouts', 'Blacklisted', 'Ignored']

function successRate(v) {
  if (v['Found'] === 0) {
    return 0
  }
  return 100 * v['Found'] / stats.reduce((x,col) => x + v[col], 0)
}

function SourceOverview({ sources }) {
  var summary = {'_': 1}
  const cols = pipeline.concat(stats)
  cols.forEach(col => summary[col] = 0)
  sources.forEach(probe => 
    cols.forEach(col => 
      summary[col] += probe[col]))
  return (
    <div className="card probes table-responsive">
      <table className="table text-start caption-top">
        <thead>
          <tr className="text-uppercase text-muted">
            <td>Source</td>
            <IconHeader col="Exclusive" icon="exclude" title="Exclusive" />
            <IconHeader col="Dirty" icon="diamond-half" title="Dirty" />
            <IconHeader col="Contribution" icon="diamond-fill" title="Contributed" />
            <IconHeader col="Scheduled" icon="inboxes-fill" title="Scheduled" />
            <IconHeader col="New" icon="inboxes" title="New" />
            <IconHeader col="Probing" icon="hand-index" title="Probing" />
            <IconHeader col="Rate" icon="check2-circle" title="Percentage of good" />
            <IconHeader col="Found" icon="award" title="Found and working" />
            <IconHeader col="Timeouts" icon="hourglass" title="Not responding" />
            <IconHeader col="Blacklisted" icon="envelope-x" title="Blacklisted" />
            <IconHeader col="Ignored" icon="journal-x" title="Ignored" />
          </tr>
        </thead>
        <tbody>
          {sources.map(probe =>
            <Probe key={probe.Name} {...probe} />
          )}
        </tbody>
        <tfoot className="text-muted">
          <tr>
            <td />
            <Cols {...summary} />
          </tr>
        </tfoot>
      </table>
    </div>
  );
}

function Cols(props) {
  return [
    overall.map(col => 
      <td key={col} className={`metric col-${col}`}>{tinyNum(props[col])}</td>),
    pipeline.map(col => 
      <td key={col} className={`metric col-${col}`}>{props[col]}</td>),
    <td key="sr" className={`metric col-Rate`}>{successRate(props).toFixed(2)}%</td>,
    stats.map(col => 
      <td key={col} className={`metric col-${col}`}>{tinyNum(props[col])}</td>)
  ]
}

function tinyNum(n) {
  if (!n) {
    return 0
  }
  if (n < 1000) {
    return n
  } else if (n < 1000000) {
    return (n/1000).toFixed()+'k'
  }
  return (n/1000000).toFixed(1)+'m'
}

function Probe(props) {
  let {Name, State, Progress, Failure, Updated, NextRefresh, UrlPrefix, Homepage} = props
  let style = {}
  let rowClass = ""
  let running = State === "running"
  if (running && Progress > 1) {
    rowClass = "probe-running"
    const lg = `linear-gradient(90deg, #080 ${Progress}%, #fff 0%)`
    style.backgroundImage = lg
  }
  let refresh = running 
    ? <TimeDiff ts={Updated} title='Updated' />
    : <TimeDiff ts={NextRefresh} title='Next Refresh' />
  let icons = {
    'running': <i className="spinner-border spinner-border-sm text-success" />,
    'failed': <i className="bi bi-emoji-dizzy-fill" title={Failure} />,
    'idle': <i className="bi bi-alarm text-muted" title='Idle' />,
  }
  return <tr className={rowClass} style={style}>
    <td>
      {icons[State]}&nbsp; 
      <a href={`/history?filter=URL ~ "${UrlPrefix}" AND StatusCode < 500`}
        className="app-link" 
        target="_blank"
        rel="noreferrer">{Name}</a>&nbsp;
      <a href={Homepage} target="_blank" rel="noreferrer">{refresh}</a>
    </td>
    <Cols {...props} />
  </tr>
}

export default function Dashboard() {
  useTitle("Overview")
  const [dashboard, setDashboard] = useState(null);
  const [delay, setDelay] = useState(1000);
  useInterval(() => {
    http.get('/dashboard')
      .then(response => setDashboard(response.data))
      .catch(err => {
        if (err.isAxiosError) {
          console.error(err.response.data)
        }
        setDelay(null)
      })
  }, delay)
  if (dashboard == null) {
    return <div className="d-flex justify-content-center">
      <div className="spinner-border" role="status">
        <span className="visually-hidden">Loading...</span>
      </div>
    </div>
  }
  return <div>
    <div className="row row-cols-1 row-cols-sm-2 row-cols-md-3">
      {dashboard.Cards.map(card => 
        <Card key={card.Name} label={card.Name} value={card.Value} 
          increment={card.Increment} /> )}
    </div>
    <SourceOverview sources={dashboard.Refresh} />
  </div>
}