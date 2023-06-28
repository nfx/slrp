import { useState } from "react";
import "./History.css";
import { Facet, IconHeader, LiveFilter, QueryFacets, useTitle } from "./util";

function convertSize(bytes: number) {
  if (bytes < 1024) {
    return `${bytes.toFixed()}b`;
  } else if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed()}kb`;
  }
  return `${(bytes / 1024 / 1024).toFixed()}mb`;
}

type RequestProps = {
  ID: string;
  Serial: number;
  Attempt: number;
  Ts: string;
  Method: string;
  URL: string;
  Status: string;
  StatusCode: number;
  Proxy: string;
  Appeared: number;
  Size: number;
  Took: number;
};

function Request(props: RequestProps) {
  const pos = props.URL.indexOf("/", 9);
  const path = props.URL.substring(pos);

  let color = "text-muted";
  if (props.StatusCode < 300) {
    color = "text-success";
  } else if (props.StatusCode < 500) {
    color = "text-warning";
  }
  // TODO: add links in backend
  return (
    <tr className="list-group-item-action">
      <td className="text-muted">
        <small>{new Date(props.Ts).toLocaleTimeString()}</small>
      </td>
      <td>
        <span className="request">
          {props.Method}{" "}
          <a className="app-link" href={`http://localhost:8089/api/history/${props.ID}?format=text`} rel="noreferrer" target="_blank">
            <abbr title={props.URL}>{path}</abbr>
          </a>
          <sup>
            <a className="text-muted" href={`/history?filter=Serial:${props.Serial}`}>
              {props.Serial}
            </a>
          </sup>
        </span>
      </td>
      <td className={color}>
        {props.StatusCode === 200 ? 200 : <abbr title={props.Status}>{props.StatusCode}</abbr>} <sup>{props.Attempt}</sup>
      </td>
      <td className="text-muted proxy">
        <a className="link-primary app-link" href={`/history?filter=Proxy:"${Proxy}"`}>
          {props.Proxy}
        </a>{" "}
        <sup>{props.Appeared}</sup>
      </td>
      <td className="size">{convertSize(props.Size)}</td>
      <td className="took">{props.Took}s</td>
    </tr>
  );
}

export default function History() {
  useTitle("History");
  const [result, setResult] = useState<{ facets: Facet[]; Records?: RequestProps[] }>();
  return (
    <div id="history-table" className="card history table-responsive">
      <LiveFilter endpoint="/history" onUpdate={setResult} minDelay={2000} />
      {result && (
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
            <tbody>{result.Records && result.Records.map(r => <Request key={r.ID} {...r} />)}</tbody>
          </table>
        </div>
      )}
    </div>
  );
}
