import { ReactNode, useState } from "react";
import { Card, IconHeader, TimeDiff, http, useInterval, useTitle } from "./util";

const overall = ["Exclusive", "Dirty", "Contribution"] as const;
const pipeline = ["Scheduled", "New", "Probing"] as const;
const stats = ["Found", "Timeouts", "Blacklisted", "Ignored"] as const;
const cols = [...pipeline, ...stats];
type Summary = Partial<ProbeProps>;

function successRate(v: Summary) {
  if (!v["Found"]) {
    return 0;
  }
  return (
    (100 * v["Found"]) /
    stats.reduce((x, col) => {
      const val = v[col];
      return x + (val ? val : 0);
    }, 0)
  );
}

function SourceOverview({ sources }: { sources: ProbeProps[] }) {
  const summary: Summary = {};
  sources.forEach(probe =>
    cols.forEach(col => {
      const oldVal = summary[col];
      const val = probe[col as keyof ProbeProps] as number;
      summary[col] = val + (oldVal ? oldVal : 0);
    })
  );
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
          {sources.map(probe => (
            <Probe key={probe.Name} {...probe} />
          ))}
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

function Cols(props: Summary) {
  // `overall`, `pipeline` and `stats` contain numeric fields
  return (
    <>
      {overall.map(col => (
        <td key={col} className={`metric col-${col}`}>
          {tinyNum(props[col])}
        </td>
      ))}
      {pipeline.map(col => (
        <td key={col} className={`metric col-${col}`}>
          {props[col]}
        </td>
      ))}
      <td key="sr" className={`metric col-Rate`}>
        {successRate(props).toFixed(2)}%
      </td>
      {stats.map(col => (
        <td key={col} className={`metric col-${col}`}>
          {tinyNum(props[col])}
        </td>
      ))}
    </>
  );
}

function tinyNum(n?: number) {
  if (!n) {
    return 0;
  }
  if (n < 1000) {
    return n;
  } else if (n < 1000000) {
    return (n / 1000).toFixed() + "k";
  }
  return (n / 1000000).toFixed(1) + "m";
}

type ProbeProps = {
  Name: string;
  State: string;
  Progress: number;
  Failure: string;
  EstFinish: number;
  NextRefresh: number;
  UrlPrefix: string;
  Homepage: string;
  Scheduled: number;
  New: number;
  Probing: number;
  Found: number;
  Timeouts: number;
  Blacklisted: number;
  Ignored: number;
  Exclusive: number;
  Dirty: number;
  Contribution: number;
};
// type ProbeProps = { [key: string]: number | string };

function Probe(props: ProbeProps) {
  const { Name, State, Progress, Failure, EstFinish, NextRefresh, UrlPrefix, Homepage } = props;
  const style: Record<string, string | number> = {};
  let rowClass = "";
  let running = State === "running";
  if (running && Progress > 1) {
    rowClass = "probe-running";
    const lg = `linear-gradient(90deg, #080 ${Progress}%, #fff 0%)`;
    style.backgroundImage = lg;
  }
  let refresh = running ? <TimeDiff ts={EstFinish} title="Estimated finish" /> : <TimeDiff ts={NextRefresh} title="Next Refresh" />;
  let icons: Record<string, ReactNode> = {
    running: <i className="spinner-border spinner-border-sm text-success" />,
    failed: <i className="bi bi-emoji-dizzy-fill" title={Failure} />,
    idle: <i className="bi bi-alarm text-muted" title="Idle" />
  };
  return (
    <tr className={rowClass} style={style}>
      <td>
        {icons[State]}&nbsp;
        <a href={`/history?filter=URL ~ "${UrlPrefix}" AND StatusCode < 500`} className="app-link" target="_blank" rel="noreferrer">
          {Name}
        </a>
        &nbsp;
        <a href={Homepage} target="_blank" rel="noreferrer">
          {refresh}
        </a>
      </td>
      <Cols {...props} />
    </tr>
  );
}

type Card = {
  Name: string;
  Value: string;
  Increment?: number;
};

export default function Dashboard() {
  useTitle("Overview");
  const [dashboard, setDashboard] = useState<{ Cards: Card[]; Refresh: ProbeProps[] }>();
  const [delay, setDelay] = useState<number | undefined>(1000);
  useInterval(() => {
    http
      .get("/dashboard")
      .then(response => setDashboard(response.data))
      .catch(err => {
        if (err.isAxiosError) {
          console.error(err.response.data);
        }
        setDelay(undefined);
      });
  }, delay);
  return dashboard ? (
    <div>
      <div className="row row-cols-1 row-cols-sm-2 row-cols-md-3">
        {dashboard.Cards.map(card => (
          <Card key={card.Name} label={card.Name} value={card.Value} increment={card.Increment} />
        ))}
      </div>
      <SourceOverview sources={dashboard.Refresh} />
    </div>
  ) : (
    <div className="d-flex justify-content-center">
      <div className="spinner-border" role="status">
        <span className="visually-hidden">Loading...</span>
      </div>
    </div>
  );
}
