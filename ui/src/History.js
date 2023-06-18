import { IconHeader, LiveFilter, QueryFacets, useTitle } from "./util";
import { useState } from "react";
import "./History.css";

function convertSize(bytes) {
  if (bytes < 1024) {
    return `${bytes.toFixed()}b`;
  } else if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed()}kb`;
  }
  return `${(bytes / 1024 / 1024).toFixed()}mb`;
}

function Request(props) {
  let { ID, Serial, Attempt, Ts, Method, URL, Status, StatusCode, Proxy, Appeared, Size, Took } = props;

  let pos = URL.indexOf("/", 9);
  let path = URL.substring(pos);

  let color = "text-muted";
  if (StatusCode < 300) {
    color = "text-success";
  } else if (StatusCode < 500) {
    color = "text-warning";
  }
  // TODO: add links in backend
  return (
    <tr className="list-group-item-action">
      <td className="text-muted">
        <small>{new Date(Ts).toLocaleTimeString()}</small>
      </td>
      <td>
        <span className="request">
          {Method}{" "}
          <a className="app-link" href={`http://localhost:8089/api/history/${ID}?format=text`} rel="noreferrer" target="_blank">
            <abbr title={URL}>{path}</abbr>
          </a>
          <sup>
            <a className="text-muted" href={`/history?filter=Serial:${Serial}`}>
              {Serial}
            </a>
          </sup>
        </span>
      </td>
      <td className={color}>
        {StatusCode === 200 ? 200 : <abbr title={Status}>{StatusCode}</abbr>} <sup>{Attempt}</sup>
      </td>
      <td className="text-muted proxy">
        <a className="link-primary app-link" href={`/history?filter=Proxy:"${Proxy}"`}>
          {Proxy}
        </a>{" "}
        <sup>{Appeared}</sup>
      </td>
      <td className="size">{convertSize(Size)}</td>
      <td className="took">{Took}s</td>
    </tr>
  );
}

export default function History() {
  useTitle("History");
  const [result, setResult] = useState(null);
  return (
    <div id="history-table" className="card history table-responsive">
      <LiveFilter endpoint="/history" onUpdate={setResult} minDelay={2000} />
      {result !== null && (
        <div>
          <QueryFacets endpoint="/history" {...result} />
          <table className="table text-start table-sm">
            <thead>
              <tr className="text-uppercase text-muted">
                <th></th>
                <IconHeader icon="filetype-raw" title="Click on link to get pretty dump. Click on number to filter by serial." />
                <IconHeader icon="123" title="HTTP status code" />
                <IconHeader icon="link proxy" title="Proxy used" />
                <IconHeader icon="arrow-left-right size" title="Size" />
                <IconHeader icon="hourglass-bottom took" title="Proxy used" />
              </tr>
            </thead>
            <tbody>{result.Records !== null && result.Records.map(r => <Request key={r.ID} {...r} />)}</tbody>
          </table>
        </div>
      )}
    </div>
  );
}
